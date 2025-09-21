import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import Cluster from '#models/cluster'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'
import ClusterDnsVerifyJob from './cluster_dns_verify_job.js'

interface ProvisionKubernetesBootJobPayload {
  clusterId: string
}

export default class ProvisionKubernetesBootJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionKubernetesBootJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.kubernetesBootStartedAt = DateTime.now()
    cluster.kubernetesBootCompletedAt = null
    cluster.kubernetesBootErrorAt = null

    await cluster.save()

    try {
      // Get kubeconfig components from database (encrypted)
      if (!cluster.kubeconfig) {
        throw new Error('Kubeconfig not found in cluster database record')
      }

      console.log('Starting Kubernetes boot deployment process...')

      const kubeconfigComponents = cluster.kubeconfig

      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.KUBERNETES_BOOT)

      const executor = new TerraformExecutor(cluster.id, 'kubernetes-boot').vars({
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

      cluster.kubernetesBootCompletedAt = DateTime.now()

      await cluster.save()

      // Dispatch DNS verification job to automatically check DNS configuration
      await queue.dispatch(ClusterDnsVerifyJob, payload)
    } catch (error) {
      cluster.kubernetesBootErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  async rescue(_payload: ProvisionKubernetesBootJobPayload) {}
}