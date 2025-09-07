import ClusterNode from '#models/cluster_node'
import logger from '@adonisjs/core/services/logger'
import { RedisStream } from '#utils/redis_stream'

interface TalosDisk {
  bus_path: string
  cdrom: boolean
  dev_path: string
  io_size: number
  modalias: string
  model: string
  pretty_size: string
  readonly: boolean
  rotational: boolean
  secondary_disks: string[]
  sector_size: number
  serial: string
  size: number
  sub_system: string
  symlinks: string[]
  transport: string
  uuid: string
  wwid: string
}

interface VolumesOutput {
  all_worker_disks: {
    value: Record<string, TalosDisk[]>
  }
}

export class TalosDetectionService {
  constructor(
    private clusterId: string,
    private streamName?: string
  ) {}


  /**
   * Match cluster node storage with volume outputs directly
   */
  private async matchStorageWithVolumes(node: ClusterNode, volumeOutputs: VolumesOutput): Promise<number> {
    let matchedCount = 0

    for (const storage of node.storages || []) {
      let matchingDisk: TalosDisk | null = null
      let diskSymlink: string | null = null

      if (volumeOutputs?.all_worker_disks?.value) {
        // Get the disks for this specific node using node slug as key
        const nodeDisks = volumeOutputs.all_worker_disks.value[node.slug]
        
        if (nodeDisks && Array.isArray(nodeDisks)) {
          // Search within this node's disks for volume symlink matching
          matchingDisk = nodeDisks.find((disk: TalosDisk) => 
            disk.symlinks?.some((symlink: string) =>
              symlink.includes(`volume-${storage.slug}`)
            )
          ) || null
          
          if (matchingDisk) {
            diskSymlink = matchingDisk.symlinks.find((symlink: string) =>
              symlink.includes(`volume-${storage.slug}`)
            ) || null
          }
        } else {
          await this.logToStream('node_disks_not_found', `No disk data found for node ${node.slug} in volume outputs`)
        }
      }
      
      if (matchingDisk && diskSymlink) {
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
          method: 'volume_outputs_direct'
        })
      } else {
        await this.logToStream('disk_not_matched', `No matching disk found for storage ${storage.slug} on node ${node.slug}`)
        
        logger.warn('No matching disk found for storage', {
          clusterId: this.clusterId,
          nodeId: node.id,
          storageId: storage.id,
          storageSlug: storage.slug,
          nodeHasDisksInOutput: !!(volumeOutputs?.all_worker_disks?.value?.[node.slug]),
        })
      }
    }

    return matchedCount
  }

  /**
   * Match and update disks only (without network interface detection)
   */
  async matchAndUpdateDisks(node: ClusterNode, volumeOutputs: VolumesOutput): Promise<{ success: boolean; error?: string; matchedStorageCount: number }> {
    try {
      await this.logToStream('disk_matching', `Matching disks for node ${node.slug}`)

      const matchedStorageCount = await this.matchStorageWithVolumes(node, volumeOutputs)

      await this.logToStream('disk_matching_complete', `Matched ${matchedStorageCount} disks for node ${node.slug}`)
      
      logger.info('Matched disks for node', {
        clusterId: this.clusterId,
        nodeId: node.id,
        matchedStorageCount,
      })

      return {
        success: true,
        matchedStorageCount
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      await this.logToStream('error', `Failed to match disks for ${node.slug}: ${errorMessage}`)
      
      logger.error('Failed to match disks for node', {
        clusterId: this.clusterId,
        nodeId: node.id,
        error: errorMessage,
      })

      return { success: false, error: errorMessage, matchedStorageCount: 0 }
    }
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