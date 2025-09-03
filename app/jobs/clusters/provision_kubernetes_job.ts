import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { DateTime } from 'luxon'
import logger from '@adonisjs/core/services/logger'
import { TalosCtl } from '#services/talos/talos_ctl'

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

      for (const node of cluster.nodes) {
        const [[_disks, disksError], [addresses, addressesError]] = await Promise.all([
          new TalosCtl().getDisks({ nodes: [node.ipv4Address as string] }),
          new TalosCtl().getAddresses({ nodes: [node.ipv4Address as string] })
        ])

        if (disksError || addressesError) {
          logger.error('Error fetching disks or addresses', {
            clusterId: cluster.id,
            nodeId: node.id,
            disksError,
            addressesError,
          })


          throw new Error(`${disksError?.message} ${addressesError?.message}`)
        }

        console.dir({ addresses }, { depth: null })
        break
      }

    } finally {
      cluster.kubernetesClusterCompletedAt = null
      cluster.kubernetesClusterErrorAt = DateTime.now()

      await cluster.save()
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
