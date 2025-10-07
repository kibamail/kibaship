import Cluster from '#models/cluster'
import ClusterLoadBalancer from '#models/cluster_load_balancer'
import logger from '@adonisjs/core/services/logger'
import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import { DateTime } from 'luxon'
import { PingIpv4Address } from '#services/ping/ping_ipv4_address'
import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'
import { TerraformStage } from '#services/terraform/terraform_executor'
import { TalosBareMetalNetworkDetectionService } from '#services/talos/talos_bare_metal_network_detection_service'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { createExecutor } from '#services/terraform/main'
import yaml from 'yaml'
import drive from '@adonisjs/drive/services/main'
import ProvisionKubernetesConfigJob from './provision_kubernetes_config_job.js'

interface ProvisionBareMetalServersBootstrapJobPayload {
  clusterId: string
}

export default class ProvisionBareMetalServersBootstrapJob extends Job {
  // Redis stream properties
  private streamName: string = ''
  private clusterId: string = ''
  private stage: TerraformStage = 'bare-metal-servers-bootstrap'

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
      'Servers bootstrap started - waiting for Talos nodes to come online'
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

  /**
   * Calculate private gateway from vSwitch subnet IP range
   * Returns the first usable IP in the range (network address + 1)
   */
  private calculatePrivateGateway(vswitchSubnetIpRange: string | null): string {
    if (!vswitchSubnetIpRange) {
      throw new Error('vSwitch subnet IP range is not configured')
    }

    // Parse IP range (e.g., "10.0.1.0/24")
    const [networkAddress] = vswitchSubnetIpRange.split('/')
    const parts = networkAddress.split('.')

    // First usable IP is network address + 1
    // For 10.0.1.0/24, the first IP is 10.0.1.1
    parts[3] = '1'

    return parts.join('.')
  }

  async handle(payload: ProvisionBareMetalServersBootstrapJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.info('Cluster not found. Might have been deleted.', payload.clusterId)
      return
    }

    await this.initializeStream(cluster.id)

    cluster.serversStartedAt = DateTime.now()
    cluster.serversCompletedAt = null
    cluster.serversErrorAt = null
    await cluster.save()

    try {
      const nodes = cluster.nodes

      await this.logToStream(
        'info',
        `Waiting for ${nodes.length} Talos node(s) to boot and become reachable (max 10 minutes)`
      )
      logger.info(`Waiting for ${nodes.length} Talos nodes to come online...`)

      const pingService = new PingIpv4Address()
      const maxAttempts = 30 // 30 attempts
      const intervalSeconds = 20 // 20 seconds between attempts = 10 minutes max

      const pingPromises = nodes.map(async (node) => {
        if (!node.ipv4Address) {
          throw new Error(`Node ${node.slug} does not have an IPv4 address`)
        }

        logger.info(`Waiting for node ${node.slug} (${node.ipv4Address}) to come online...`)

        const isReachable = await pingService.waitUntilReachable(
          node.ipv4Address,
          maxAttempts,
          intervalSeconds
        )

        if (!isReachable) {
          await this.logToStream(
            'error',
            `Node ${node.slug} (${node.ipv4Address}) did not come online within 10 minutes`
          )
          throw new Error(
            `Node ${node.slug} (${node.ipv4Address}) did not come online within 10 minutes`
          )
        }

        await this.logToStream('success', `Node ${node.slug} is online and reachable`)
        logger.info(`Node ${node.slug} is online and reachable`)
        return true
      })

      await Promise.all(pingPromises)

      await this.logToStream('success', 'All Talos nodes are online and reachable')
      logger.info('All Talos nodes are online and reachable')

      // Detect network configuration on each node
      await this.logToStream('info', 'Starting network detection on all nodes')
      logger.info('Starting network detection on all nodes')

      const networkDetectionService = new TalosBareMetalNetworkDetectionService()

      // Calculate private gateway from vswitch subnet (first IP in range)
      const privateGateway = this.calculatePrivateGateway(cluster.vswitchSubnetIpRange)
      await this.logToStream('info', `Calculated private gateway: ${privateGateway}`)

      for (const node of nodes) {
        if (!node.ipv4Address) continue

        await this.logToStream('info', `Detecting network configuration on node ${node.slug}`)

        try {
          const networkConfig = await networkDetectionService.detect(node.ipv4Address)

          // Store network configuration to node
          if (networkConfig.publicInterface) {
            node.publicNetworkInterface = networkConfig.publicInterface.name
            node.publicIpv4Gateway = networkConfig.publicInterface.gateway
            node.publicAddressSubnet = networkConfig.publicInterface.addressSubnet
          }

          node.privateIpv4Gateway = privateGateway
          await node.save()

          // Log complete detection results
          await this.logToStream(
            'network_detection',
            `Node ${node.slug} network detection complete:\n` +
              `  Public Interface: ${networkConfig.publicInterface?.name || 'not found'}\n` +
              `  Public Address/Subnet: ${networkConfig.publicInterface?.addressSubnet || 'not found'}\n` +
              `  Public Gateway: ${networkConfig.publicInterface?.gateway || 'not found'}\n` +
              `  Private Gateway: ${privateGateway}\n` +
              `  Total Links: ${networkConfig.links.length}\n` +
              `  Total Addresses: ${networkConfig.addresses.length}\n` +
              `  Total Routes: ${networkConfig.routes.length}`
          )

          logger.info('Network detection complete for node', {
            clusterId: cluster.id,
            nodeSlug: node.slug,
            nodeIp: node.ipv4Address,
            publicInterface: networkConfig.publicInterface,
            privateGateway,
            linksCount: networkConfig.links.length,
            addressesCount: networkConfig.addresses.length,
            routesCount: networkConfig.routes.length,
          })
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          await this.logToStream(
            'error',
            `Failed to detect network on node ${node.slug}: ${errorMessage}`
          )
          logger.error(`Failed to detect network on node ${node.slug}:`, error)
          console.error(error)
        }
      }

      await this.logToStream('success', 'Network detection completed on all nodes')

      // Bootstrap Talos on all nodes
      await this.logToStream('info', 'Starting Talos bootstrap on all nodes')
      logger.info('Starting Talos bootstrap on all nodes')

      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.BARE_METAL_TALOS_IMAGE)

      const executor = await createExecutor(cluster.id, 'bare-metal-talos-image')

      // Get the Kubernetes API load balancer
      const kubeLoadBalancer = await ClusterLoadBalancer.query()
        .where('cluster_id', cluster.id)
        .where('type', 'cluster')
        .firstOrFail()

      executor.vars({
        cluster_name: cluster.subdomainIdentifier,
        cluster_endpoint: `https://${kubeLoadBalancer.publicIpv4Address}:6443`,
        vlan_id: cluster.vlanId as number,
        vswitch_subnet_ip_range: cluster.vswitchSubnetIpRange as string,
        cluster_network_ip_range: cluster.networkIpRange as string,
      })

      await this.logToStream('info', 'Initial`izing Terraform for Talos bootstrap')
      await executor.init()

      await this.logToStream('info', 'Applying Talos configuration to all nodes')
      await executor.apply({ autoApprove: true })

      await this.logToStream('success', 'Talos bootstrap completed successfully')

      // Get outputs from terraform
      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string)

      // Store talos config and kubeconfig (similar to provision_servers_job.ts)
      if (output.talos_config?.value) {
        cluster.talosConfig = output.talos_config.value
      }

      if (output.kubeconfig?.value) {
        const kubeconfigData = yaml.parse(output.kubeconfig.value)
        const clusterData = kubeconfigData.clusters[0].cluster
        const userData = kubeconfigData.users[0].user

        cluster.kubeconfig = {
          host: clusterData.server,
          clientCertificate: userData['client-certificate-data'],
          clientKey: userData['client-key-data'],
          clusterCaCertificate: clusterData['certificate-authority-data'],
        }
      }

      // Write talos config and kubeconfig to YAML files in cluster's terraform directory
      const disk = drive.use('fs')
      const clusterBasePath = `terraform/clusters/${cluster.id}`

      try {
        const talosConfig = output.talos_config?.value as Cluster['talosConfig']
        const kubeConfigRaw = output.kubeconfig?.value as string

        if (talosConfig && cluster.nodes) {
          const controlPlaneEndpoints = cluster.nodes
            .filter((node) => node.type === 'master')
            .map((node) => node.ipv4Address)
            .filter(Boolean)

          const talosConfigYaml = {
            context: cluster.subdomainIdentifier,
            contexts: {
              [cluster.subdomainIdentifier]: {
                endpoints: controlPlaneEndpoints,
                ca: talosConfig.ca_certificate,
                crt: talosConfig.client_certificate,
                key: talosConfig.client_key,
              },
            },
          }

          await disk.put(`${clusterBasePath}/talosconfig.yaml`, yaml.stringify(talosConfigYaml))
          await this.logToStream('success', 'Talos config written to talosconfig.yaml')
        }

        // Generate proper kubeconfig YAML structure
        if (kubeConfigRaw && cluster.kubeconfig) {
          const kubeconfigYaml = {
            apiVersion: 'v1',
            kind: 'Config',
            'current-context': `admin@${cluster.subdomainIdentifier}`,
            clusters: [
              {
                name: cluster.subdomainIdentifier,
                cluster: {
                  'certificate-authority-data': cluster.kubeconfig.clusterCaCertificate,
                  server: cluster.kubeconfig.host,
                },
              },
            ],
            contexts: [
              {
                name: `admin@${cluster.subdomainIdentifier}`,
                context: {
                  cluster: cluster.subdomainIdentifier,
                  namespace: 'default',
                  user: `admin@${cluster.subdomainIdentifier}`,
                },
              },
            ],
            users: [
              {
                name: `admin@${cluster.subdomainIdentifier}`,
                user: {
                  'client-certificate-data': cluster.kubeconfig.clientCertificate,
                  'client-key-data': cluster.kubeconfig.clientKey,
                },
              },
            ],
          }

          await disk.put(`${clusterBasePath}/kubeconfig.yaml`, yaml.stringify(kubeconfigYaml))
          await this.logToStream('success', 'Kubeconfig written to kubeconfig.yaml')
        }
      } catch (error) {
        console.error(error)
      }

      cluster.serversCompletedAt = DateTime.now()
      await cluster.save()

      await this.logToStream('success', 'Servers bootstrap completed')

      // Dispatch Kubernetes configuration job
      await queue.dispatch(ProvisionKubernetesConfigJob, payload)
    } catch (error) {
      await this.logToStream('error', `Servers bootstrap failed: ${error.message}`)
      logger.error('Error in ProvisionBareMetalServersBootstrapJob:', error)
      cluster.status = 'unhealthy'
      cluster.serversErrorAt = DateTime.now()
      console.error(error)

      await cluster.save()
      throw error
    }
  }

  async rescue(payload: ProvisionBareMetalServersBootstrapJobPayload) {
    logger.error('Failed to bootstrap servers after all retries')

    const cluster = await Cluster.findOrFail(payload.clusterId)
    cluster.serversErrorAt = DateTime.now()
    await cluster.save()

    try {
      // Try to log the failure to stream
      this.clusterId = payload.clusterId
      this.streamName = RedisStreamConfig.getClusterStream(payload.clusterId)

      await this.logToStream('error', 'Servers bootstrap failed after all retries')
    } catch (error) {
      // If we can't log to stream, just log locally
      logger.error('Failed to log rescue to stream:', error)
    }
  }
}
