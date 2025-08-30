import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import ClusterLoadBalancer from '#models/cluster_load_balancer'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionServersJob from './provision_servers_job.js'

interface ProvisionLoadBalancersJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface LoadBalancersOutput {
  ingress_load_balancer_id: TerraformOutputValue
  ingress_load_balancer_name: TerraformOutputValue
  ingress_load_balancer_public_ip: TerraformOutputValue
  kube_load_balancer_id: TerraformOutputValue
  kube_load_balancer_name: TerraformOutputValue
  kube_load_balancer_public_ip: TerraformOutputValue
  // Hetzner-specific outputs (optional)
  ingress_load_balancer_private_ip?: TerraformOutputValue
  kube_load_balancer_private_ip?: TerraformOutputValue
}

export default class ProvisionLoadBalancersJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionLoadBalancersJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.loadBalancersStartedAt = DateTime.now()
    cluster.loadBalancersErrorAt = null
    cluster.loadBalancersCompletedAt = null
    await cluster.save()

    try {
      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.LOAD_BALANCERS)

      const executor = new TerraformExecutor(cluster.id, 'load-balancers')
        .vars({
          ...cluster.cloudProvider?.getTerraformCredentials(),
          cluster_name: cluster.subdomainIdentifier,
          location: cluster.location,
          network_id: cluster.providerNetworkId || ''
        })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as LoadBalancersOutput

      await this.createOrUpdateLoadBalancer(
        cluster.id,
        'ingress',
        output.ingress_load_balancer_id.value as string,
        output.ingress_load_balancer_public_ip.value as string,
        (output.ingress_load_balancer_private_ip?.value as string) || (output.ingress_load_balancer_public_ip.value as string)
      )

      await this.createOrUpdateLoadBalancer(
        cluster.id,
        'cluster',
        output.kube_load_balancer_id.value as string,
        output.kube_load_balancer_public_ip.value as string,
        (output.kube_load_balancer_private_ip?.value as string) || (output.kube_load_balancer_public_ip.value as string)
      )

      cluster.loadBalancersCompletedAt = DateTime.now()

      await cluster.save()

      await queue.dispatch(ProvisionServersJob, payload)

    } catch (error) {
      this.logger.error(error)

      throw error
    }
  }

  async rescue(_payload: ProvisionLoadBalancersJobPayload) {
  }

  private async createOrUpdateLoadBalancer(
    clusterId: string,
    type: 'ingress' | 'cluster',
    providerId: string,
    publicIpv4Address: string,
    privateIpv4Address: string
  ): Promise<ClusterLoadBalancer> {
    let loadBalancer = await ClusterLoadBalancer.query()
      .where('cluster_id', clusterId)
      .where('type', type)
      .first()

    if (!loadBalancer) {
      loadBalancer = new ClusterLoadBalancer()
    }

    loadBalancer.clusterId = clusterId
    loadBalancer.type = type
    loadBalancer.providerId = providerId
    loadBalancer.publicIpv4Address = publicIpv4Address
    loadBalancer.privateIpv4Address = privateIpv4Address
    await loadBalancer.save()

    return loadBalancer
  }
}