import ClusterNode from '#models/cluster_node'
import logger from '@adonisjs/core/services/logger'
import { TalosCtl, TalosRoute } from '#services/talos/talos_ctl'
import { NetworkInterfaceDetector } from '#services/network/network_interface_detector'
import { RedisStream } from '#utils/redis_stream'

export interface TalosDetectionResult {
  success: boolean
  error?: string
  nodeData?: {
    privateInterface: string | null
    publicInterface: string | null
    publicIpv4Gateway: string | null
    matchedStorageCount: number
  }
}

export class TalosDetectionService {
  constructor(
    private clusterId: string,
    private streamName?: string
  ) {}

  /**
   * Detect and update network interfaces and disk storage for a cluster node
   */
  async detectAndUpdateNode(node: ClusterNode): Promise<TalosDetectionResult> {
    try {
      await this.logToStream('node_analysis', `Analyzing network interfaces for node ${node.slug}`)

      const [[disks, disksError], [addresses, addressesError], [routes, routesError]] = await Promise.all([
        new TalosCtl().getDisks({ nodes: [node.ipv4Address as string] }),
        new TalosCtl().getAddresses({ nodes: [node.ipv4Address as string] }),
        new TalosCtl().getRoutes({ nodes: [node.ipv4Address as string] })
      ])

      if (disksError || addressesError || routesError) {
        const errorMsg = `Error fetching node data: ${disksError?.message} ${addressesError?.message} ${routesError?.message}`
        await this.logToStream('error', errorMsg)
        
        logger.error('Error fetching disks, addresses, or routes', {
          clusterId: this.clusterId,
          nodeId: node.id,
          disksError,
          addressesError,
          routesError,
        })

        return { success: false, error: errorMsg }
      }

      const matchedStorageCount = await this.matchStorageWithDisks(node, disks || [])

      const { privateInterface, publicInterface } = NetworkInterfaceDetector.detectNetworkInterfaces(addresses || [])
      
      const publicIpv4Gateway = this.findPublicIpv4Gateway(routes || [], publicInterface)
      
      await this.logToStream('interface_detected', `Detected interfaces for node ${node.slug}: private=${privateInterface}, public=${publicInterface}, gateway=${publicIpv4Gateway}`)
      
      logger.info('Detected network interfaces and gateway for node', {
        clusterId: this.clusterId,
        nodeId: node.id,
        privateInterface,
        publicInterface,
        publicIpv4Gateway,
      })

      await node.merge({
        privateNetworkInterface: privateInterface,
        publicNetworkInterface: publicInterface,
        publicIpv4Gateway: publicIpv4Gateway,
      }).save()

      await this.logToStream('node_updated', `Updated node ${node.slug} with network interface and gateway information`)
      
      logger.info('Updated node with network interface and gateway information', {
        clusterId: this.clusterId,
        nodeId: node.id,
        privateInterface,
        publicInterface,
        publicIpv4Gateway,
        matchedStorageCount,
      })

      return {
        success: true,
        nodeData: {
          privateInterface,
          publicInterface,
          publicIpv4Gateway,
          matchedStorageCount
        }
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      await this.logToStream('error', `Failed to detect node data for ${node.slug}: ${errorMessage}`)
      
      logger.error('Failed to detect node data', {
        clusterId: this.clusterId,
        nodeId: node.id,
        error: errorMessage,
      })

      return { success: false, error: errorMessage }
    }
  }

  /**
   * Match cluster node storage with disk symlinks
   */
  private async matchStorageWithDisks(node: ClusterNode, disks: any[]): Promise<number> {
    let matchedCount = 0

    for (const storage of node.storages || []) {
      // Find disk with symlink containing the storage slug
      const matchingDisk = disks.find((disk: any) => 
        disk.spec?.symlinks?.some((symlink: string) =>
          symlink.includes(storage.slug)
        )
      )
      
      if (matchingDisk) {
        const diskSymlink = matchingDisk.spec.symlinks.find((symlink: string) =>
          symlink.includes(storage.slug)
        )
        
        if (diskSymlink) {
          storage.diskName = diskSymlink
          await storage.save()
          matchedCount++
          
          await this.logToStream('disk_matched', `Matched storage ${storage.slug} with disk ${diskSymlink} for node ${node.slug}`)
          
          logger.info('Matched cluster node storage with disk', {
            clusterId: this.clusterId,
            nodeId: node.id,
            storageId: storage.id,
            storageSlug: storage.slug,
            diskSymlink,
          })
        }
      }
    }

    return matchedCount
  }

  /**
   * Find the public IPv4 gateway from routes
   */
  private findPublicIpv4Gateway(routes: TalosRoute[], publicInterface: string | null): string | null {
    if (!publicInterface) {
      return null
    }

    // Find route with empty dst (default gateway), inet4 family, and matching public interface
    const defaultRoute = routes.find(route => 
      route.spec.dst === '' && 
      route.spec.family === 'inet4' && 
      route.spec.outLinkName === publicInterface
    )

    logger.info('Detected public IPv4 gateway from routes', {
      clusterId: this.clusterId,
      publicInterface,
      publicIpv4Gateway: defaultRoute?.spec.gateway || null,
    })

    return defaultRoute?.spec.gateway || null
  }

  /**
   * Log a message to the Redis stream if available
   */
  private async logToStream(logType: string, message: string): Promise<void> {
    if (!this.streamName) return
    
    try {
      await new RedisStream()
        .stream(this.streamName)
        .fields({
          type: logType,
          message: message.trim(),
          timestamp: new Date().toISOString(),
          cluster_id: this.clusterId,
          stage: 'kubernetes'
        })
        .add()
    } catch (error) {
      console.error('Failed to log to stream:', error)
    }
  }
}