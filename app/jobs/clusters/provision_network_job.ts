import Cluster from '#models/cluster'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import logger from '@adonisjs/core/services/logger'
import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import { DateTime } from 'luxon'
import ProvisionSshKeysJob from './provision_ssh_keys_job.js'

interface ProvisionNetworkJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface NetworkOutput {
  network_id: TerraformOutputValue
  network_name: TerraformOutputValue
  network_ip_range: TerraformOutputValue
  network_labels: TerraformOutputValue
  // Hetzner-specific outputs (optional)
  subnet_id?: TerraformOutputValue
  subnet_ip_range?: TerraformOutputValue
  subnet_network_zone?: TerraformOutputValue
  // DigitalOcean-specific outputs (optional)
  network_region?: TerraformOutputValue
}

export default class ProvisionNetworkJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionNetworkJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.info('Cluster not found. Might have been deleted.', payload.clusterId)

      return
    }

    cluster.networkingStartedAt = DateTime.now()
    cluster.networkingCompletedAt = null
    cluster.networkingErrorAt = null
    await cluster.save()

    const terraform = new TerraformService(payload.clusterId)

    try {
      await terraform.generate(cluster, TerraformTemplate.NETWORK)

      const executor = new TerraformExecutor(cluster.id, 'network')
        .vars({
          ...cluster.cloudProvider?.getTerraformCredentials(),
          cluster_name: cluster.subdomainIdentifier,
          network_zone: cluster.cloudProvider?.getNetworkZone(cluster.location) || 'eu-central',
          location: cluster.location
        })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as NetworkOutput

      cluster.networkIpRange = output.network_ip_range.value as string
      cluster.providerNetworkId = output.network_id.value as string
      cluster.subnetIpRange = (output.subnet_ip_range?.value as string || output.network_ip_range.value as string) || null
      cluster.providerSubnetId = (output.subnet_id?.value || output.network_id.value) as string || null

      cluster.networkingCompletedAt = DateTime.now()

      await cluster.save()

      await queue.dispatch(ProvisionSshKeysJob, payload)
    } catch (error) {
      console.error(error)
      cluster.status = 'unhealthy'
      cluster.networkingErrorAt = DateTime.now()

      await cluster.save()
    }
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(_payload: ProvisionNetworkJobPayload) {
  }
}
