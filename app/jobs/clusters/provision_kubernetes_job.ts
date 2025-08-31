import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { AnsibleExecutor } from '#services/ansible/ansible_executor'
import { DateTime } from 'luxon'
import logger from '@adonisjs/core/services/logger'

interface ProvisionKubernetesJobPayload {
  clusterId: string
}

export default class ProvisionKubernetesJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionKubernetesJobPayload) {
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

    cluster.kubernetesClusterStartedAt = DateTime.now()
    cluster.kubernetesClusterCompletedAt = null
    cluster.kubernetesClusterErrorAt = null
    await cluster.save()

    try {
      const ansibleExecutor = new AnsibleExecutor(payload.clusterId, 'kubernetes')

      await ansibleExecutor.init(cluster)

      await ansibleExecutor.executePlaybook()

      cluster.kubernetesClusterCompletedAt = DateTime.now()
      cluster.status = 'healthy'

      await cluster.save()

      logger.info('Kubernetes provisioning completed successfully', {
        clusterId: cluster.id,
        streamName: ansibleExecutor.getStreamName(),
      })
    } catch (error) {
      cluster.kubernetesClusterErrorAt = DateTime.now()
      cluster.status = 'unhealthy'
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
}
