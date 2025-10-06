import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import Cluster from '#models/cluster'
import CloudProvider from '#models/cloud_provider'
import { DateTime } from 'luxon'
import { hetznerRobot } from '#services/hetzner-robot/provider'
import logger from '@adonisjs/core/services/logger'
import ProvisionBareMetalCloudLoadBalancerJob from './provision_bare_metal_cloud_load_balancer_job.js'
import type { components } from '#services/hetzner-robot/openapi/schema'
import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'

interface ProvisionHetznerBareMetalClusterPayload {
  clusterId: string
}

export default class ProvisionHetznerBareMetalCluster extends Job {
  private streamName: string = ''
  private clusterId: string = ''
  private stage = 'bare-metal-networking' as const

  constructor() {
    super()
  }

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
      'Bare metal cluster provisioning started - preparing vSwitch and server networking'
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

  async handle(payload: ProvisionHetznerBareMetalClusterPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    await this.initializeStream(cluster.id)

    cluster.provisioningStartedAt = DateTime.now()
    cluster.bareMetalNetworkingStartedAt = DateTime.now()
    await cluster.save()

    await this.logToStream('info', 'Loading cloud provider credentials')

    const cloudProvider = await CloudProvider.findOrFail(cluster.cloudProviderId)

    const robot = hetznerRobot({
      username: cloudProvider.credentials.username as string,
      password: cloudProvider.credentials.password as string,
    })

    await this.logToStream('info', 'Connected to Hetzner Robot API')

    let vswitchId = cluster.vswitchId
    let vlanId = cluster.vlanId
    let currentVswitch: components['schemas']['VSwitchDetailed'] | null = null

    if (!vswitchId || !vlanId) {
      await this.logToStream('info', 'Checking for existing vSwitch configuration')
      logger.info('Creating new vSwitch for cluster', { clusterId: cluster.id })

      const vswitchName = cluster.subdomainIdentifier

      const existingVswitches = await robot.vswitches().list()
      await this.logToStream('info', `Found ${existingVswitches.length} existing vSwitches`)

      const existingVswitch = existingVswitches.find((v) => v.name === vswitchName)

      if (existingVswitch) {
        await this.logToStream(
          'info',
          `Found existing vSwitch "${existingVswitch.name}" (VLAN ${existingVswitch.vlan}) - reusing it`
        )
        logger.info('vSwitch with this name already exists, using existing vSwitch', {
          clusterId: cluster.id,
          vswitchId: existingVswitch.id,
          vlanId: existingVswitch.vlan,
          vswitchName: existingVswitch.name,
        })

        cluster.vlanId = existingVswitch.vlan
        cluster.vswitchId = existingVswitch.id
        await cluster.save()
        currentVswitch = await robot.vswitches().get(existingVswitch.id)
      } else {
        // Create new vSwitch
        const usedVlans = existingVswitches.map((v) => v.vlan)

        let newVlanId = 4000
        while (usedVlans.includes(newVlanId) && newVlanId <= 4091) {
          newVlanId++
        }

        if (newVlanId > 4091) {
          await this.logToStream('error', 'No available VLAN IDs (4000-4091 range exhausted)')
          throw new Error('No available VLAN IDs. All VLANs from 4000-4091 are in use.')
        }

        await this.logToStream(
          'info',
          `Creating new vSwitch "${vswitchName}" with VLAN ID ${newVlanId}`
        )

        const vswitch = await robot.vswitches().create(vswitchName, newVlanId)

        if (!vswitch) {
          await this.logToStream('error', 'Failed to create vSwitch via Hetzner Robot API')
          throw new Error('Failed to create vSwitch')
        }

        cluster.vlanId = vswitch.vlan
        cluster.vswitchId = vswitch.id
        await cluster.save()
        currentVswitch = vswitch

        await this.logToStream(
          'success',
          `vSwitch created successfully (ID: ${vswitch.id}, VLAN: ${vswitch.vlan})`
        )
        logger.info('vSwitch created successfully', {
          clusterId: cluster.id,
          vswitchId: vswitch.id,
          vlanId: vswitch.vlan,
          vswitchName: vswitch.name,
        })
      }
    } else {
      await this.logToStream('info', `Using configured vSwitch (ID: ${vswitchId}, VLAN: ${vlanId})`)

      // Verify the vSwitch still exists
      currentVswitch = await robot.vswitches().get(vswitchId)

      if (!currentVswitch) {
        await this.logToStream('error', `vSwitch ID ${vswitchId} not found in Hetzner Robot`)
        logger.warn('Stored vSwitch ID not found, will create a new one', {
          clusterId: cluster.id,
          vswitchId,
        })

        // Reset cluster vSwitch info so it gets recreated
        cluster.vlanId = null
        cluster.vswitchId = null
        await cluster.save()

        throw new Error('vSwitch not found. Please retry the job to create a new vSwitch.')
      }

      await this.logToStream('info', 'Verified vSwitch exists and is accessible')
      logger.info('Using existing vSwitch', {
        clusterId: cluster.id,
        vswitchId: cluster.vswitchId,
        vlanId: cluster.vlanId,
      })
    }

    // Step 2: Add servers to vSwitch (idempotent)
    if (!cluster.vswitchId) {
      await this.logToStream('error', 'vSwitch ID not found in cluster configuration')
      throw new Error('vSwitch ID not found in cluster')
    }

    await this.logToStream(
      'info',
      `Preparing to add ${cluster.nodes.length} bare metal servers to vSwitch`
    )

    logger.info('Checking vSwitch server membership', {
      clusterId: cluster.id,
      vswitchId: cluster.vswitchId,
      vlanId: cluster.vlanId,
    })

    const nodes = cluster.nodes
    const serverIdentifiers = nodes
      .map((node) => node.providerId)
      .filter((id): id is string => id !== null)

    if (serverIdentifiers.length === 0) {
      await this.logToStream('error', 'No servers found in cluster nodes')
      throw new Error('No servers found in cluster nodes')
    }

    await this.logToStream('info', `Found ${serverIdentifiers.length} server(s) to configure`)

    // Use the vSwitch we already fetched
    const vswitch = currentVswitch

    if (!vswitch) {
      await this.logToStream('error', 'vSwitch configuration not available')
      throw new Error('vSwitch not found')
    }

    // Extract server IPs that are already on the vSwitch
    const currentServerIps = vswitch.server?.map((s) => s.server_ip) || []

    // Filter out servers that are already on the vSwitch
    const serversToAdd = serverIdentifiers.filter(
      (serverId) => !currentServerIps.includes(serverId)
    )

    if (serversToAdd.length === 0) {
      await this.logToStream(
        'success',
        `All ${serverIdentifiers.length} server(s) already attached to vSwitch`
      )
      logger.info('All servers are already on the vSwitch', {
        clusterId: cluster.id,
        vswitchId: cluster.vswitchId,
        serverCount: serverIdentifiers.length,
      })
    } else {
      await this.logToStream(
        'info',
        `Attaching ${serversToAdd.length} server(s) to vSwitch (${currentServerIps.length} already attached)`
      )

      logger.info('Adding servers to vSwitch', {
        clusterId: cluster.id,
        vswitchId: cluster.vswitchId,
        serversToAdd,
        alreadyOnVswitch: currentServerIps,
      })

      const success = await robot.vswitches().addServers(cluster.vswitchId, serversToAdd)

      if (!success) {
        await this.logToStream('error', 'Failed to add servers to vSwitch')
        throw new Error('Failed to add servers to vSwitch')
      }

      await this.logToStream(
        'success',
        `Successfully attached ${serversToAdd.length} server(s) to vSwitch`
      )

      logger.info('Servers added to vSwitch successfully', {
        clusterId: cluster.id,
        vswitchId: cluster.vswitchId,
        serverCount: serversToAdd.length,
        serverIdentifiers: serversToAdd,
      })
    }

    // 3. trigger rescue mode on all servers
    // 4. wait for rescue mode to be ready
    // 5. execute terraform that downloads terraform in each of the servers
    // 6. reboot servers
    // 7. wait for servers to be ready
    // 8. execute terraform that applies talos configs
    // 9. wait for talos to be ready
    // 10. discover all disks on the server
    // 11. create node storages for each disk
    // 12. bootstrap kubernetes cluster

    // Dispatch the next job to provision cloud networking
    await this.logToStream(
      'success',
      'Bare metal server preparation complete - proceeding to cloud networking'
    )

    cluster.bareMetalNetworkingCompletedAt = DateTime.now()
    await cluster.save()

    await queue.dispatch(ProvisionBareMetalCloudLoadBalancerJob, {
      clusterId: cluster.id,
    })
  }

  async rescue(payload: ProvisionHetznerBareMetalClusterPayload) {
    logger.error('Failed to provision Hetzner bare metal cluster after all retries')

    const cluster = await Cluster.findOrFail(payload.clusterId)
    cluster.bareMetalNetworkingErrorAt = DateTime.now()
    await cluster.save()

    try {
      // Try to log the failure to stream
      this.clusterId = payload.clusterId
      this.streamName = RedisStreamConfig.getClusterStream(payload.clusterId)

      await this.logToStream('error', 'Bare metal cluster provisioning failed after all retries')
    } catch (error) {
      // If we can't log to stream, just log locally
      logger.error('Failed to log rescue to stream:', error)
    }
  }
}
