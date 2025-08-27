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
    const cluster = await Cluster.query()
      .where('id', payload.clusterId)
      .preload('cloudProvider')
      .preload('nodes')
      .preload('sshKeys')
      .preload('nodes')
      .firstOrFail()

    const terraformService = new TerraformService(cluster.id)

    // Generate all templates individually
    await terraformService.generate(cluster, TerraformTemplate.NETWORK)
    await terraformService.generate(cluster, TerraformTemplate.SSH_KEYS)
    await terraformService.generate(cluster, TerraformTemplate.LOAD_BALANCERS)
    await terraformService.generate(cluster, TerraformTemplate.SERVERS)
    await terraformService.generate(cluster, TerraformTemplate.VOLUMES)
  }

  async rescue(_payload: ProvisionClusterJobPayload) {
    console.log('Cluster provisioning job failed after all retries')
  }
}