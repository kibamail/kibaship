import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionServersJob from './provision_servers_job.js'

interface ProvisionKubernetesConfigJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string
}

interface KubernetesConfigOutput {
  talos_config: TerraformOutputValue
  control_plane_machine_configuration: TerraformOutputValue
  worker_machine_configuration: TerraformOutputValue
}

export default class ProvisionKubernetesConfigJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionKubernetesConfigJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.kubernetesConfigStartedAt = DateTime.now()
    cluster.kubernetesConfigErrorAt = null
    cluster.kubernetesConfigCompletedAt = null
    await cluster.save()

    try {
      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.KUBERNETES_CONFIG)

      const executor = new TerraformExecutor(cluster.id, 'kubernetes-config')
        .vars({
          ...cluster.cloudProvider?.getTerraformCredentials(),
          cluster_name: cluster.subdomainIdentifier,
          location: cluster.location,
        })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as KubernetesConfigOutput

      // Store all configurations in the cluster
      cluster.talosConfig = output.talos_config.value
      cluster.controlPlaneConfig = output.control_plane_machine_configuration.value
      cluster.workerConfig = output.worker_machine_configuration.value
      
      cluster.kubernetesConfigCompletedAt = DateTime.now()
      await cluster.save()

      await queue.dispatch(ProvisionServersJob, payload)

    } catch (error) {
      cluster.kubernetesConfigErrorAt = DateTime.now()
      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionKubernetesConfigJobPayload) {
  }
}