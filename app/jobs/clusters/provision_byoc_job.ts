import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { DateTime } from 'luxon'
import { createExecutor } from '#services/terraform/main'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'

interface ProvisionByocJobPayload {
  clusterId: string
}

/**
 * ProvisionByocJob
 *
 * Responsibilities:
 * - Marks BYOC discovery as started
 * - Builds the BYOC Terraform template via TerraformService
 * - Executes `terraform init` and `terraform apply`
 * - On success: marks byocCompletedAt; on failure: sets byocErrorAt
 */
export default class ProvisionByocJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionByocJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)
    if (!cluster) return

    cluster.byocStartedAt = DateTime.now()
    cluster.byocCompletedAt = null
    cluster.byocErrorAt = null
    await cluster.save()

    try {
      const terraform = new TerraformService(cluster.id)
      await terraform.generate(cluster, TerraformTemplate.KUBERNETES_BYOC)

      const executor = (await createExecutor(cluster.id, 'kubernetes-byoc')).vars({
        talos_ca_certificate: cluster.talosConfig?.ca_certificate || '',
        talos_client_certificate: cluster.talosConfig?.client_certificate || '',
        talos_client_key: cluster.talosConfig?.client_key || '',
      })

      await executor.init()
      await executor.apply({ autoApprove: true })

      cluster.byocErrorAt = DateTime.now()
      await cluster.save()
    } catch (error) {
      cluster.byocErrorAt = DateTime.now()
      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionByocJobPayload) {}
}

