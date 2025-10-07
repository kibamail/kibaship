import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { createExecutor } from '#services/terraform/main'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import logger from '@adonisjs/core/services/logger'
import { hetznerRobot } from '#services/hetzner-robot/provider'

interface DestroyBareMetalHetznerJobPayload {
  clusterId: string
}

export default class DestroyBareMetalHetznerJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: DestroyBareMetalHetznerJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.warn(`Cluster ${payload.clusterId} not found, skipping destruction`)
      return
    }

    logger.info(`Starting destruction of bare metal Hetzner cluster ${cluster.id}`)

    cluster.status = 'unhealthy'
    await cluster.save()

    try {
      // Step 1: Detach all servers from vSwitch
      await this.detachServersFromVSwitch(cluster)

      // Step 2: Destroy Hetzner Robot networking (load balancers, network, etc.)
      await this.destroyHetznerRobotNetworking(cluster)

      // Step 3: Clean up terraform files
      await this.cleanupTerraformFiles(cluster.id)

      // Step 4: Delete the cluster from database
      await cluster.delete()

      logger.info(`Successfully destroyed bare metal Hetzner cluster ${cluster.id}`)
    } catch (error) {
      logger.error(`Failed to destroy bare metal Hetzner cluster ${cluster.id}:`, error)
      throw error
    }
  }

  private async detachServersFromVSwitch(cluster: Cluster): Promise<void> {
    logger.info(`Detaching servers from vSwitch for cluster ${cluster.id}`)

    try {
      if (!cluster.vswitchId) {
        logger.info(`No vSwitch ID found for cluster ${cluster.id}, skipping server detachment`)
        return
      }

      // Get the cloud provider (Hetzner Robot credentials)
      if (!cluster.cloudProvider) {
        logger.warn(`Cloud provider not found for cluster ${cluster.id}`)
        return
      }

      const robot = hetznerRobot({
        username: cluster.cloudProvider.credentials.username as string,
        password: cluster.cloudProvider.credentials.password as string,
      })

      // Get server identifiers from cluster nodes
      const serverIdentifiers = cluster.nodes
        .map((node) => node.providerId)
        .filter((id): id is string => id !== null)

      if (serverIdentifiers.length === 0) {
        logger.info(`No servers found to detach from vSwitch for cluster ${cluster.id}`)
        return
      }

      logger.info(`Detaching ${serverIdentifiers.length} server(s) from vSwitch ${cluster.vswitchId}`, {
        clusterId: cluster.id,
        vswitchId: cluster.vswitchId,
        serverIdentifiers,
      })

      const success = await robot.vswitches().removeServers(cluster.vswitchId, serverIdentifiers)

      if (!success) {
        logger.error(`Failed to detach servers from vSwitch for cluster ${cluster.id}`)
        // Don't throw - continue with destruction even if this fails
      } else {
        logger.info(`Successfully detached ${serverIdentifiers.length} server(s) from vSwitch`)
      }
    } catch (error) {
      logger.error(`Error detaching servers from vSwitch for cluster ${cluster.id}:`, error)
      // Don't throw - continue with destruction even if this fails
    }
  }

  private async destroyHetznerRobotNetworking(cluster: Cluster): Promise<void> {
    logger.info(`Destroying Hetzner Robot networking for cluster ${cluster.id}`)

    try {
      // Generate terraform files for Hetzner Robot networking
      const terraform = new TerraformService(cluster.id)
      await terraform.generate(cluster, TerraformTemplate.HETZNER_ROBOT_NETWORKING)

      // Get the robot cloud provider (Hetzner cloud provider used for networking)
      const robotCloudProvider = await cluster.robotCloudProvider

      if (!robotCloudProvider) {
        logger.warn(`Robot cloud provider not found for cluster ${cluster.id}`)
        return
      }

      const executor = (
        await createExecutor(cluster.id, 'bare-metal-cloud-load-balancer')
      ).vars({
        ...robotCloudProvider.getTerraformCredentials(),
        cluster_name: cluster.subdomainIdentifier,
        location: cluster.location,
        network_zone: robotCloudProvider.getNetworkZone(cluster.location) || 'eu-central',
        vswitch_id: cluster.vswitchId as number,
      })

      await executor.init()
      await executor.destroy({ autoApprove: true })
      logger.info(`Successfully destroyed Hetzner Robot networking for cluster ${cluster.id}`)
    } catch (error) {
      logger.error(
        `Failed to destroy Hetzner Robot networking for cluster ${cluster.id}, continuing...`,
        error
      )
    }
  }

  private async cleanupTerraformFiles(clusterId: string): Promise<void> {
    try {
      const terraformService = new TerraformService(clusterId)
      await terraformService.cleanup()
      logger.info(`Cleaned up Terraform files for cluster ${clusterId}`)
    } catch (error) {
      logger.error(`Failed to cleanup Terraform files for cluster ${clusterId}:`, error)
    }
  }

  async rescue(payload: DestroyBareMetalHetznerJobPayload) {
    logger.error(
      `Failed to destroy bare metal Hetzner cluster ${payload.clusterId} after all retries`
    )
  }
}
