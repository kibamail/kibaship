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

    // Step 1: Fetch all vSwitches from Hetzner API
    await this.logToStream('info', 'Fetching all vSwitches from Hetzner Robot')
    const allVswitches = await robot.vswitches().list()
    await this.logToStream('info', `Found ${allVswitches.length} existing vSwitches`)

    logger.info('Fetched vSwitches from Hetzner', {
      clusterId: cluster.id,
      vswitchCount: allVswitches.length,
    })

    let currentVswitch: components['schemas']['VSwitchDetailed'] | null = null

    // Step 2: Check if user selected a vSwitch
    if (cluster.vswitchId) {
      await this.logToStream('info', `Verifying selected vSwitch (ID: ${cluster.vswitchId}) exists`)

      // Check if selected vSwitch is in the fetched list
      const selectedVswitch = allVswitches.find((v) => v.id === cluster.vswitchId)

      if (!selectedVswitch) {
        await this.logToStream(
          'error',
          `Selected vSwitch ID ${cluster.vswitchId} not found in Hetzner Robot account`
        )
        throw new Error(
          `Selected vSwitch ID ${cluster.vswitchId} does not exist in your Hetzner Robot account`
        )
      }

      // Set vswitch_id and vlan_id on cluster
      cluster.vswitchId = selectedVswitch.id
      cluster.vlanId = selectedVswitch.vlan
      await cluster.save()

      await this.logToStream(
        'success',
        `Using selected vSwitch "${selectedVswitch.name}" (ID: ${selectedVswitch.id}, VLAN: ${selectedVswitch.vlan})`
      )

      logger.info('Using selected vSwitch', {
        clusterId: cluster.id,
        vswitchId: selectedVswitch.id,
        vlanId: selectedVswitch.vlan,
        vswitchName: selectedVswitch.name,
      })

      // Fetch detailed vSwitch info
      currentVswitch = await robot.vswitches().get(selectedVswitch.id)
    } else {
      // Step 3: No vSwitch provided - create a new one
      await this.logToStream('info', 'No vSwitch selected - creating new vSwitch')
      logger.info('Creating new vSwitch for cluster', { clusterId: cluster.id })

      const vswitchName = cluster.subdomainIdentifier

      // Find an unused VLAN ID from 4000 to 4091
      const usedVlans = allVswitches.map((v) => v.vlan)

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

      // Step 4: Create vSwitch
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

    // Step 5: Fetch servers attached to the vSwitch and attach missing ones
    if (!cluster.vswitchId) {
      await this.logToStream('error', 'vSwitch ID not found in cluster configuration')
      throw new Error('vSwitch ID not found in cluster')
    }

    if (!currentVswitch) {
      await this.logToStream('error', 'vSwitch configuration not available')
      throw new Error('vSwitch not found')
    }

    await this.logToStream(
      'info',
      `Preparing to attach ${cluster.nodes.length} bare metal servers to vSwitch`
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

    await this.logToStream('info', `Attaching ${serverIdentifiers.length} server(s) to vSwitch`)

    logger.info('Adding servers to vSwitch', {
      clusterId: cluster.id,
      vswitchId: cluster.vswitchId,
      servers: serverIdentifiers,
    })

    // Batch attach all servers at once (idempotent - will skip already attached servers)
    const success = await robot.vswitches().addServers(cluster.vswitchId, serverIdentifiers)

    if (!success) {
      await this.logToStream('error', 'Failed to add servers to vSwitch')
      throw new Error('Failed to add servers to vSwitch')
    }

    await this.logToStream(
      'success',
      `Successfully attached ${serverIdentifiers.length} server(s) to vSwitch`
    )

    logger.info('Servers added to vSwitch successfully', {
      clusterId: cluster.id,
      vswitchId: cluster.vswitchId,
      serverCount: serverIdentifiers.length,
      serverIdentifiers,
    })

    // Wait 2 minutes for server attachment to complete on Hetzner's side
    await this.logToStream(
      'info',
      'Waiting 2 minutes for vSwitch server attachment to complete...'
    )
    await new Promise((resolve) => setTimeout(resolve, 120000)) // 120 seconds = 2 minutes
    await this.logToStream('success', 'vSwitch attachment wait period completed')

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
