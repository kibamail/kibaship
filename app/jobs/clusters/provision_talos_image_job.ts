import Cluster from '#models/cluster'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import logger from '@adonisjs/core/services/logger'
import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import { DateTime } from 'luxon'
import ProvisionNetworkJob from './provision_network_job.js'
import ProvisionSshKeysJob from './provision_ssh_keys_job.js'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'

interface ProvisionTalosImageJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface TalosImageOutput {
  talos_image_id: TerraformOutputValue
}

export default class ProvisionTalosImageJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionTalosImageJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.info('Cluster not found. Might have been deleted.', payload.clusterId)
      return
    }

    if (cluster.cloudProvider.type === CloudProviderDefinitions.HETZNER) {
      const serverTypes = CloudProviderDefinitions.serverTypes('hetzner')

      if (serverTypes[cluster.serverType].arch === 'arm') {
        cluster.providerImageId = cluster.cloudProvider.providerImageArm64
      } else {
        cluster.providerImageId = cluster.cloudProvider.providerImageAmd64
      }

      cluster.talosImageCompletedAt = DateTime.now()
      await cluster.save()

      await queue.dispatch(ProvisionNetworkJob, payload)

      return
    }

    cluster.talosImageStartedAt = DateTime.now()
    cluster.talosImageCompletedAt = null
    cluster.talosImageErrorAt = null
    await cluster.save()

    const terraform = new TerraformService(payload.clusterId)

    try {
      await terraform.generate(cluster, TerraformTemplate.TALOS_IMAGE)

      const executor = new TerraformExecutor(cluster.id, 'talos-image').vars({
        ...cluster.cloudProvider?.getTerraformCredentials(),
      })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as TalosImageOutput

      cluster.providerImageId = output.talos_image_id.value as string
      cluster.talosImageCompletedAt = DateTime.now()

      await cluster.save()

      if (cluster.cloudProvider.type === 'digital_ocean') {
        cluster.networkingCompletedAt = DateTime.now()
        await queue.dispatch(ProvisionSshKeysJob, payload)
      } else {
        await queue.dispatch(ProvisionNetworkJob, payload)
      }

      await cluster.save()
    } catch (error) {
      console.error(error)
      cluster.status = 'unhealthy'
      cluster.talosImageErrorAt = DateTime.now()

      await cluster.save()
    }
  }

  async rescue(_payload: ProvisionTalosImageJobPayload) {}
}
