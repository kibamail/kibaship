import Cluster from '#models/cluster'
import { Job } from '@rlanz/bull-queue'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'

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
    const cluster = await Cluster.completeFirstOrFail(payload.clusterId)

    const terraformService = new TerraformService(cluster.id)

    await terraformService.generate(cluster, TerraformTemplate.NETWORK)

  }

  async rescue(_payload: ProvisionClusterJobPayload) {
    console.log('Cluster provisioning job failed after all retries')
  }
}