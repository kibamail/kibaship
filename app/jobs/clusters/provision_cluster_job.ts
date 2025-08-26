import Cluster from '#models/cluster'
import { Job } from '@rlanz/bull-queue'
import { TerraformService } from '#services/terraform/terraform_service'

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
    const cluster = await Cluster.query()
      .where('id', payload.clusterId)
      .preload('cloudProvider')
      .preload('nodes')
      .preload('sshKeys')
      .preload('nodes')
      .firstOrFail()

    await new TerraformService(cluster.id).generate(cluster)
  }

  async rescue(_payload: ProvisionClusterJobPayload) {
    console.log('Cluster provisioning job failed after all retries')
  }
}