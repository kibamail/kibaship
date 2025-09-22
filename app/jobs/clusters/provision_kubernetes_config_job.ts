import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import Cluster from '#models/cluster'
import { createExecutor } from '#services/terraform/main'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { KubernetesService } from '#services/kubernetes/kubernetes_service'
import { DateTime } from 'luxon'
import ProvisionKubernetesBootJob from './provision_kubernetes_boot_job.js'

interface ProvisionKubernetesConfigJobPayload {
  clusterId: string
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
    cluster.kubernetesConfigCompletedAt = null
    cluster.kubernetesConfigErrorAt = null

    await cluster.save()

    try {
      // Get kubeconfig components from database (encrypted)
      if (!cluster.kubeconfig) {
        throw new Error('Kubeconfig not found in cluster database record')
      }

      // Calculate expected node count (control planes + workers)
      const expectedNodeCount = cluster.nodes?.length || 0
      if (expectedNodeCount === 0) {
        throw new Error('No nodes found for cluster')
      }

      // Wait for all cluster nodes to be ready before proceeding
      const kubernetesService = new KubernetesService(cluster)
      
      console.log(`Waiting for ${expectedNodeCount} nodes to be ready...`)
      const nodesReady = await kubernetesService.waitForNodesDiscovered(expectedNodeCount, 100000)
      
      if (!nodesReady) {
        throw new Error(`Timeout waiting for ${expectedNodeCount} nodes to be ready`)
      }

      console.log(`All ${expectedNodeCount} nodes are ready. Proceeding with Kubernetes configuration...`)

      const kubeconfigComponents = cluster.kubeconfig

      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.KUBERNETES_CONFIG)

      const executor = (await createExecutor(cluster.id, 'kubernetes-config')).vars({
        ...cluster.cloudProvider?.getTerraformCredentials(),
        cluster_name: cluster.subdomainIdentifier,
        location: cluster.location,
        kube_host: kubeconfigComponents.host,
        kube_client_certificate: kubeconfigComponents.clientCertificate,
        kube_client_key: kubeconfigComponents.clientKey,
        kube_cluster_ca_certificate: kubeconfigComponents.clusterCaCertificate,
      })

      await executor.init()
      await executor.apply({ autoApprove: true })

      cluster.kubernetesConfigCompletedAt = DateTime.now()

      await cluster.save()

      await queue.dispatch(ProvisionKubernetesBootJob, payload)
    } catch (error) {
      console.error('Error in ProvisionKubernetesConfigJob:', error)
      cluster.kubernetesConfigErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionKubernetesConfigJobPayload) {}
}