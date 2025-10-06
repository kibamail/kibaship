import Cluster from '#models/cluster'
import ClusterNodeStorage from '#models/cluster_node_storage'
import logger from '@adonisjs/core/services/logger'
import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import { DateTime } from 'luxon'
import { hetznerRobot } from '#services/hetzner-robot/provider'
import { PingIpv4Address } from '#services/ping/ping_ipv4_address'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import ProvisionBareMetalServersBootstrapJob from './provision_bare_metal_servers_bootstrap_job.js'
import { createExecutor, TerraformStage } from '#services/terraform/main'
import { writeFile } from 'node:fs/promises'
import bytes from 'bytes'
import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'

interface ProvisionBareMetalTalosImageJobPayload {
  clusterId: string
}

interface ServerRescueInfo {
  serverNumber: number
  ipAddress: string
  password: string
  needsReset: boolean
}

interface DiskDevice {
  name: string
  size: string
  type: string
  disk_by_id: string
}

interface TalosInstallation {
  device: string
  disk_by_id: string
  full_path: string
  disk_by_id_path: string
}

interface NodeDiskDiscovery {
  all_devices: DiskDevice[]
  talos_installation: TalosInstallation
}

interface DiskDiscoveryOutput {
  [key: string]: {
    value: NodeDiskDiscovery
  }
}

export default class ProvisionBareMetalTalosImageJob extends Job {
  // Store rescue passwords in memory for later use
  private serverRescuePasswords: Map<number, string> = new Map()

  // Store disk discovery results
  private diskDiscoveryResults: DiskDiscoveryOutput | null = null

  // Redis stream properties
  private streamName: string = ''
  private clusterId: string = ''
  private stage: TerraformStage = 'bare-metal-talos-image'

  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Initialize the Redis stream for logging
   */
  private async initializeStream(clusterId: string): Promise<void> {
    this.clusterId = clusterId
    this.streamName = RedisStreamConfig.getClusterStream(clusterId)

    await this.logToStream(
      'stream_initialized',
      'Talos image installation started - preparing servers in rescue mode'
    )
  }

  /**
   * Log a message to the Redis stream
   */
  private async logToStream(logType: string, message: string): Promise<void> {
    try {
      await new RedisStream()
        .stream(this.streamName)
        .fields({
          type: logType,
          message: message.trim(),
          timestamp: new Date().toISOString(),
          cluster_id: this.clusterId,
          stage: this.stage,
        })
        .add()
    } catch (error) {
      logger.error('Failed to log to stream:', error)
    }
  }

  async handle(payload: ProvisionBareMetalTalosImageJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.info('Cluster not found. Might have been deleted.', payload.clusterId)
      return
    }

    await this.initializeStream(cluster.id)

    cluster.talosImageStartedAt = DateTime.now()
    cluster.talosImageCompletedAt = null
    cluster.talosImageErrorAt = null

    await cluster.save()

    try {
      if (!cluster.cloudProvider) {
        throw new Error('Robot cloud provider ID not found in cluster')
      }

      await this.logToStream('info', 'Loading cloud provider credentials')

      const robot = hetznerRobot({
        username: cluster.cloudProvider.credentials.username as string,
        password: cluster.cloudProvider.credentials.password as string,
      })

      await this.logToStream('info', 'Connected to Hetzner Robot API')

      const nodes = cluster.nodes
      const serverRescueInfo: ServerRescueInfo[] = []

      await this.logToStream('info', `Checking rescue mode status for ${nodes.length} servers`)
      logger.info(`Checking rescue mode status for ${nodes.length} servers`)

      for (const node of nodes) {
        if (!node.providerId) {
          throw new Error(`Server provider ID not found for node ${node.id}`)
        }

        const serverNumber = parseInt(node.providerId, 10)

        logger.info(`Checking rescue mode for server ${serverNumber} (${node.ipv4Address})`)

        const rescueStatus = await robot.boot().getRescueOptions(serverNumber)

        let password: string
        let serverIp: string

        let needsReset = false

        if (rescueStatus?.rescue?.active) {
          await this.logToStream(
            'info',
            `Server ${serverNumber} (${node.ipv4Address}) already in rescue mode - no reset needed`
          )
          logger.info(`Rescue mode already active for server ${serverNumber}, no reset needed`)

          if (!rescueStatus.rescue.password) {
            throw new Error(
              `Rescue mode is active but password not available for server ${serverNumber}`
            )
          }

          password = rescueStatus.rescue.password
          serverIp = rescueStatus.rescue.server_ip || (node.ipv4Address as string)
          needsReset = false
        } else {
          await this.logToStream('info', `Activating rescue mode for server ${serverNumber}`)
          logger.info(`Activating rescue mode for server ${serverNumber}`)

          const rescueResponse = await robot.boot().activateRescue(serverNumber, 'linux', {
            arch: 64,
          })

          if (!rescueResponse?.rescue) {
            await this.logToStream(
              'error',
              `Failed to activate rescue mode for server ${serverNumber}`
            )
            throw new Error(`Failed to activate rescue mode for server ${serverNumber}`)
          }

          await this.logToStream('success', `Rescue mode activated for server ${serverNumber}`)
          password = rescueResponse.rescue.password
          serverIp = rescueResponse.rescue.server_ip
          needsReset = true
        }

        this.serverRescuePasswords.set(serverNumber, password)

        serverRescueInfo.push({
          serverNumber,
          ipAddress: serverIp,
          password,
          needsReset,
        })

        logger.info(`Rescue mode ready for server ${serverNumber}, password stored in memory`)
      }

      await this.logToStream(
        'success',
        `Rescue mode ready for all ${this.serverRescuePasswords.size} servers`
      )
      logger.info(
        `Rescue mode ready for all servers. Passwords stored for ${this.serverRescuePasswords.size} servers`
      )

      // Trigger hardware reset only on servers that just had rescue mode activated
      const serversToReset = serverRescueInfo.filter((info) => info.needsReset)

      if (serversToReset.length > 0) {
        await this.logToStream(
          'info',
          `Triggering hardware reset on ${serversToReset.length} server(s) to boot into rescue mode`
        )
        logger.info(
          `Triggering hardware reset on ${serversToReset.length} server(s) to boot into rescue mode...`
        )

        for (const serverInfo of serversToReset) {
          logger.info(`Resetting server ${serverInfo.serverNumber}`)

          const resetResult = await robot.boot().resetServer(serverInfo.serverNumber, 'hw')

          if (!resetResult?.reset) {
            await this.logToStream('error', `Failed to reset server ${serverInfo.serverNumber}`)
            throw new Error(`Failed to reset server ${serverInfo.serverNumber}`)
          }

          await this.logToStream('info', `Server ${serverInfo.serverNumber} reset triggered`)
          logger.info(`Server ${serverInfo.serverNumber} reset triggered successfully`)
        }

        await this.logToStream('success', `All ${serversToReset.length} server(s) reset triggered`)
        logger.info('All required servers reset triggered.')

        // Wait 120 seconds for servers to start booting into rescue mode
        await this.logToStream(
          'info',
          'Waiting 120 seconds for servers to start booting into rescue mode'
        )
        logger.info('Waiting 120 seconds for servers to start booting...')
        await this.sleep(120000)
      } else {
        await this.logToStream('info', 'No servers need reset - all already in rescue mode')
        logger.info('No servers need reset - all already in rescue mode')
      }

      await this.logToStream(
        'info',
        `Waiting for ${serverRescueInfo.length} server(s) to come online (max 8 minutes)`
      )
      logger.info('Waiting for servers to come online in rescue mode...')

      const pingService = new PingIpv4Address()
      const maxAttempts = 24
      const intervalSeconds = 20

      const pingPromises = serverRescueInfo.map(async (serverInfo) => {
        const isReachable = await pingService.waitUntilReachable(
          serverInfo.ipAddress,
          maxAttempts,
          intervalSeconds
        )

        if (!isReachable) {
          await this.logToStream(
            'error',
            `Server ${serverInfo.serverNumber} (${serverInfo.ipAddress}) did not come online within 8 minutes`
          )
          throw new Error(
            `Server ${serverInfo.serverNumber} (${serverInfo.ipAddress}) did not come online within 8 minutes`
          )
        }

        await this.logToStream(
          'success',
          `Server ${serverInfo.serverNumber} is online and reachable`
        )
        logger.info(`Server ${serverInfo.serverNumber} is online and reachable`)
        return true
      })

      await Promise.all(pingPromises)

      await this.logToStream('success', 'All servers are online and reachable in rescue mode')
      logger.info('All servers are online and reachable')

      // Generate and execute disk discovery terraform
      await this.logToStream('info', 'Starting Talos image installation and disk discovery')
      logger.info('Starting disk discovery terraform execution...')

      const terraformService = new TerraformService(cluster.id)
      await terraformService.generate(cluster, TerraformTemplate.BARE_METAL_DISK_DISCOVERY)

      await this.logToStream('info', 'Generated Terraform template for Talos installation')
      logger.info('Disk discovery terraform template generated')

      const executor = await createExecutor(cluster.id, 'bare-metal-disk-discovery')

      // Build terraform variables with server passwords
      const terraformVariables: Record<string, string> = {}
      for (const [serverNumber, password] of this.serverRescuePasswords.entries()) {
        const node = nodes.find((n) => parseInt(n.providerId!, 10) === serverNumber)
        if (node) {
          terraformVariables[`server_${node.slug}_password`] = password
        }
      }

      await this.logToStream('info', 'Initializing Terraform')
      logger.info('Initializing disk discovery terraform...')

      await executor.init()

      await this.logToStream('info', 'Installing Talos image on all servers and discovering disks')
      logger.info('Executing disk discovery terraform with server credentials...')

      await executor.vars(terraformVariables).apply({ autoApprove: true })

      const output = await executor.output()

      // Parse disk discovery output
      this.diskDiscoveryResults = JSON.parse(output.stdout)

      if (this.diskDiscoveryResults) {
        await this.logToStream(
          'success',
          `Talos image installed on ${Object.keys(this.diskDiscoveryResults).length} nodes`
        )
        logger.info(
          `Disk discovery completed for ${Object.keys(this.diskDiscoveryResults).length} nodes`,
          this.diskDiscoveryResults
        )
      }

      // Create node storage entries from disk discovery results
      await this.logToStream('info', 'Creating node storage entries from discovered disks')
      await this.createNodeStorages(cluster)
      await this.logToStream('success', 'Node storage entries created successfully')

      cluster.talosImageCompletedAt = DateTime.now()
      await cluster.save()

      await this.logToStream(
        'success',
        'Talos image installation completed - servers rebooting into Talos'
      )

      await queue.dispatch(ProvisionBareMetalServersBootstrapJob, payload)
    } catch (error) {
      await this.logToStream('error', `Talos image installation failed: ${error.message}`)
      logger.error('Error in ProvisionBareMetalTalosImageJob:', error)
      try {
        await writeFile('logger.error.log', error.toString())
      } catch {}
      cluster.status = 'unhealthy'
      cluster.talosImageErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  async rescue(payload: ProvisionBareMetalTalosImageJobPayload) {
    logger.error('Failed to provision Talos image after all retries')

    const cluster = await Cluster.findOrFail(payload.clusterId)
    cluster.talosImageErrorAt = DateTime.now()
    await cluster.save()

    try {
      // Try to log the failure to stream
      this.clusterId = payload.clusterId
      this.streamName = RedisStreamConfig.getClusterStream(payload.clusterId)

      await this.logToStream('error', 'Talos image installation failed after all retries')
    } catch (error) {
      // If we can't log to stream, just log locally
      logger.error('Failed to log rescue to stream:', error)
    }
  }

  /**
   * Create node storage entries from disk discovery results
   */
  private async createNodeStorages(cluster: Cluster): Promise<void> {
    if (!this.diskDiscoveryResults) {
      throw new Error('Disk discovery results not available')
    }

    const nodes = cluster.nodes

    for (const node of nodes) {
      const discoveryKey = `node_${node.slug}_disk_discovery`
      const diskDiscovery = this.diskDiscoveryResults[discoveryKey]

      if (!diskDiscovery) {
        logger.warn(`No disk discovery results found for node ${node.slug}`)
        continue
      }

      const { all_devices, talos_installation } = diskDiscovery.value

      logger.info(
        `Creating storage entries for node ${node.slug} with ${all_devices.length} devices`
      )

      for (const device of all_devices) {
        const isInstallationDisk = device.disk_by_id === talos_installation.disk_by_id

        // Parse size from "476.5G" to 476.5
        const sizeInGB = this.parseDiskSize(device.size)

        await ClusterNodeStorage.create({
          clusterNodeId: node.id,
          providerId: device.disk_by_id,
          providerMountId: `/dev/disk/by-id/${device.disk_by_id}`,
          installationDisk: isInstallationDisk,
          status: 'healthy',
          size: sizeInGB,
          diskName: device.name,
        })

        logger.info(
          `Created storage for node ${node.slug}: ${device.name} (${sizeInGB}GB)${isInstallationDisk ? ' [INSTALLATION DISK]' : ''}`
        )
      }
    }

    logger.info('Node storage creation completed')
  }

  /**
   * Parse disk size from format like "476.5G" to number (in GB)
   */
  private parseDiskSize(sizeString: string): number {
    // Parse size string (e.g., "476.5G") to bytes, then convert to GB
    const sizeInBytes = bytes.parse(sizeString + 'B') // Add 'B' to make it "476.5GB"

    if (!sizeInBytes) {
      throw new Error(`Invalid disk size format: ${sizeString}`)
    }

    // Convert bytes to GB
    const sizeInGB = sizeInBytes / (1024 * 1024 * 1024)

    return sizeInGB
  }

  /**
   * Sleep for a specified number of milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms))
  }
}
