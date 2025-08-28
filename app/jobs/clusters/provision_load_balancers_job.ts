import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import ClusterLoadBalancer from '#models/cluster_load_balancer'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'

interface ProvisionLoadBalancersJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface HetznerLoadBalancersOutput {
  ingress_load_balancer_id: TerraformOutputValue
  ingress_load_balancer_name: TerraformOutputValue
  ingress_load_balancer_public_ip: TerraformOutputValue
  ingress_load_balancer_private_ip: TerraformOutputValue
  kube_load_balancer_id: TerraformOutputValue
  kube_load_balancer_name: TerraformOutputValue
  kube_load_balancer_public_ip: TerraformOutputValue
  kube_load_balancer_private_ip: TerraformOutputValue
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

      const output = JSON.parse(stdout as string) as HetznerLoadBalancersOutput

      if (cluster.cloudProvider?.type === 'hetzner') {
        const ingressLoadBalancer = new ClusterLoadBalancer()
        ingressLoadBalancer.clusterId = cluster.id
        ingressLoadBalancer.type = 'ingress'
        ingressLoadBalancer.providerId = output.ingress_load_balancer_id.value as string
        ingressLoadBalancer.publicIpv4Address = output.ingress_load_balancer_public_ip.value as string
        ingressLoadBalancer.privateIpv4Address = output.ingress_load_balancer_private_ip.value as string
        await ingressLoadBalancer.save()

        const kubeLoadBalancer = new ClusterLoadBalancer()
        kubeLoadBalancer.clusterId = cluster.id
        kubeLoadBalancer.type = 'cluster'
        kubeLoadBalancer.providerId = output.kube_load_balancer_id.value as string
        kubeLoadBalancer.publicIpv4Address = output.kube_load_balancer_public_ip.value as string
        kubeLoadBalancer.privateIpv4Address = output.kube_load_balancer_private_ip.value as string
        await kubeLoadBalancer.save()
      }

      cluster.loadBalancersCompletedAt = DateTime.now()

      await cluster.save()

    } catch (error) {
      cluster.loadBalancersError = `${cluster.loadBalancersError || ''}\n ${error?.message}`

      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionLoadBalancersJobPayload) {
  }
}