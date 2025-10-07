import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { DateTime } from 'luxon'

interface ProvisionKibashipOperatorJobPayload {
  clusterId: string
}

export default class ProvisionKibashipOperatorJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionKibashipOperatorJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.kibashipOperatorStartedAt = DateTime.now()
    cluster.kibashipOperatorCompletedAt = null
    cluster.kibashipOperatorErrorAt = null

    await cluster.save()

    try {
      // TODO: Implement Kibaship operator provisioning
      throw new Error('Kibaship operator provisioning not yet implemented')
    } catch (error) {
      cluster.kibashipOperatorErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionKibashipOperatorJobPayload) {}
}
