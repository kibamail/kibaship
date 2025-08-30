import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { TerraformExecutor, TerraformStage } from '#services/terraform/terraform_executor'
import { TerraformService } from '#services/terraform/terraform_service'
import logger from '@adonisjs/core/services/logger'

interface DestroyClusterJobPayload {
  clusterId: string
}

export default class DestroyClusterJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: DestroyClusterJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      logger.warn(`Cluster ${payload.clusterId} not found, skipping destruction`)
      return
    }

    logger.info(`Starting destruction of cluster ${cluster.id}`)

    cluster.status = 'unhealthy'
    await cluster.save()

    const stages: TerraformStage[] = ['volumes', 'servers', 'load-balancers', 'ssh-keys', 'network']

    for (const stage of stages) {
      try {
        await this.destroyStage(cluster, stage)
      } catch (error) {
        logger.error(`Failed to destroy ${stage} for cluster ${cluster.id}:`, error)
        throw error
      }
    }

    await this.cleanupTerraformFiles(cluster.id)

    await cluster.delete()

    logger.info(`Successfully destroyed cluster ${cluster.id}`)
  }

  private async destroyStage(cluster: Cluster, stage: TerraformStage): Promise<void> {
    logger.info(`Destroying ${stage} for cluster ${cluster.id}`)

    const ingressLoadBalancer = cluster.loadBalancers.find(lb => lb.type === 'ingress')
    const kubeLoadBalancer = cluster.loadBalancers.find(lb => lb.type === 'cluster')
    const controlPlaneServerIds = this.buildServerIdsMap(cluster.nodes.filter(n => n.type === 'master'))
    const workerServerIds = this.buildServerIdsMap(cluster.nodes.filter(n => n.type === 'worker'))

    const executor = new TerraformExecutor(cluster.id, stage)
      .vars({
        ...cluster.cloudProvider?.getTerraformCredentials(),
        cluster_name: cluster.subdomainIdentifier,
        location: cluster.location,
        network_zone: cluster.cloudProvider?.getNetworkZone(cluster.location) || 'eu-central',
        public_key: cluster.sshKey?.publicKey || '',
        network_id: cluster.providerNetworkId || '',
        server_type: cluster.serverType,
        ssh_key_id: cluster.sshKey?.providerId || '',
        kube_load_balancer_id: kubeLoadBalancer?.providerId || '',
        ingress_load_balancer_id: ingressLoadBalancer?.providerId || '',
        control_planes_volume_size: cluster.controlPlanesVolumeSize,
        workers_volume_size: cluster.workersVolumeSize,
        control_plane_server_ids: JSON.stringify(controlPlaneServerIds),
        worker_server_ids: JSON.stringify(workerServerIds)
      })

    try {
      await executor.destroy({ autoApprove: true })
      logger.info(`Successfully destroyed ${stage} for cluster ${cluster.id}`)
    } catch (error) {
      logger.error(`Failed to destroy ${stage} for cluster ${cluster.id}:`, error)
      throw error
    }
  }

  private buildServerIdsMap(nodes: any[]): Record<string, string> {
    const serverIds: Record<string, string> = {}

    nodes.forEach(node => {
      if (node.providerId) {
        serverIds[node.id] = node.providerId
      }
    })

    return serverIds
  }

  private async cleanupTerraformFiles(clusterId: string): Promise<void> {
    try {
      const terraformService = new TerraformService(clusterId)
      await terraformService.cleanup()
      logger.info(`Cleaned up Terraform files for cluster ${clusterId}`)
    } catch (error) {
      logger.error(`Failed to cleanup Terraform files for cluster ${clusterId}:`, error)
    }
  }

  async rescue(payload: DestroyClusterJobPayload) {
    logger.error(`Failed to destroy cluster ${payload.clusterId} after all retries`)
  }
}