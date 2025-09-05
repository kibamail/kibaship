import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { DateTime } from 'luxon'
import logger from '@adonisjs/core/services/logger'
import { TalosDetectionService } from '#services/talos/talos_detection_service'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'

interface ProvisionKubernetesJobPayload {
  clusterId: string
}

export default class ProvisionKubernetesJob extends Job {
  private streamName?: string

  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionKubernetesJobPayload) {
    this.streamName = RedisStreamConfig.getClusterStream(payload.clusterId)
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.error('Cluster not found for Kubernetes provisioning', {
        clusterId: payload.clusterId,
      })
      return
    }

    logger.info('Starting Kubernetes provisioning for cluster', {
      clusterId: cluster.id,
      clusterName: cluster.subdomainIdentifier,
    })

    await this.logToStream('k8s_start', `Starting Kubernetes provisioning for cluster ${cluster.subdomainIdentifier}`)

    cluster.kubernetesClusterStartedAt = DateTime.now()
    cluster.kubernetesClusterCompletedAt = null
    cluster.kubernetesClusterErrorAt = null

    await cluster.save()

    try {
      const detectionService = new TalosDetectionService(cluster.id, this.streamName)

      for (const node of cluster.nodes) {
        const result = await detectionService.detectAndUpdateNode(node)
        
        if (!result.success) {
          throw new Error(result.error)
        }
      }

      await this.logToStream('k8s_complete', 'Network interface detection completed successfully')

      await this.logToStream('talos_template', 'Generating Talos Terraform template')
      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.KUBERNETES)

      await this.logToStream('talos_plan', 'Planning Talos cluster Terraform execution')
      const executor = new TerraformExecutor(cluster.id, 'kubernetes').vars({
        ...cluster.cloudProvider?.getTerraformCredentials(),
      })

      await executor.init()
      await executor.plan()

      await this.logToStream('talos_plan_complete', 'Talos Terraform plan completed successfully')
      
      logger.info('Talos Terraform plan completed', {
        clusterId: cluster.id,
        clusterName: cluster.subdomainIdentifier,
      })

      cluster.kubernetesClusterErrorAt = DateTime.now()
      await cluster.save()

    } catch (error) {
      cluster.kubernetesClusterErrorAt = DateTime.now()
      await cluster.save()
      throw error
    }
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(payload: ProvisionKubernetesJobPayload) {
    logger.error('ProvisionKubernetesJob failed after all retries', {
      clusterId: payload.clusterId,
    })
  }

  /**
   * Log a message to the Redis stream
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
          cluster_id: this.streamName.split(':')[2], // Extract cluster ID from stream name
          stage: 'kubernetes'
        })
        .add()
    } catch (error) {
      console.error('Failed to log to stream:', error)
    }
  }
}
