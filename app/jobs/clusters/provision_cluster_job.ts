import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionNetworkJob from './provision_network_job.js'
import Cluster from '#models/cluster'
import { DateTime } from 'luxon'

interface ProvisionClusterJobPayload {
  clusterId: string
}

export default class ProvisionClusterJob extends Job {
  constructor() {
    super()
  }

  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionClusterJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.provisioningStartedAt = DateTime.now()

    await cluster.save()

    await queue.dispatch(ProvisionNetworkJob, payload)
  }

  async rescue(_payload: ProvisionClusterJobPayload) {
  }
}
