import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import Cluster from '#models/cluster'
import CloudProvider from '#models/cloud_provider'
import { createExecutor } from '#services/terraform/main'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'
import ProvisionBareMetalTalosImageJob from './provision_bare_metal_talos_image_job.js'

interface ProvisionBareMetalCloudLoadBalancerJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface NetworkingOutput {
  network_id: TerraformOutputValue
  network_name: TerraformOutputValue
  network_ip_range: TerraformOutputValue
  subnet_id: TerraformOutputValue
  vswitch_id: TerraformOutputValue
  ingress_load_balancer_id: TerraformOutputValue
  ingress_load_balancer_name: TerraformOutputValue
  ingress_load_balancer_public_ip: TerraformOutputValue
  ingress_load_balancer_public_ipv6: TerraformOutputValue
}

export default class ProvisionBareMetalCloudLoadBalancerJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionBareMetalCloudLoadBalancerJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.networkingStartedAt = DateTime.now()
    cluster.networkingCompletedAt = null
    cluster.networkingErrorAt = null
    await cluster.save()

    try {
      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.HETZNER_ROBOT_NETWORKING)

      if (!cluster.robotCloudProviderId) {
        throw new Error('Robot cloud provider ID not found in cluster')
      }

      const robotCloudProvider = await CloudProvider.findOrFail(cluster.robotCloudProviderId)

      const executor = await createExecutor(cluster.id, 'bare-metal-cloud-load-balancer')

      executor.vars({
        ...robotCloudProvider.getTerraformCredentials(),
        cluster_name: cluster.subdomainIdentifier,
        location: cluster.location,
        network_zone: robotCloudProvider.getNetworkZone(cluster.location) || 'eu-central',
        vswitch_id: cluster.vswitchId as number,
      })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as NetworkingOutput

      cluster.providerNetworkId = output.network_id.value as string
      cluster.providerSubnetId = output.subnet_id.value as string

      cluster.networkingCompletedAt = DateTime.now()
      await cluster.save()

      // Dispatch kubernetes config job
      await queue.dispatch(ProvisionBareMetalTalosImageJob, {
        clusterId: cluster.id,
      })
    } catch (error) {
      cluster.networkingErrorAt = DateTime.now()
      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionBareMetalCloudLoadBalancerJobPayload) {}
}
