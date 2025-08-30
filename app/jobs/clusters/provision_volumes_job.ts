import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import ClusterNodeStorage from '#models/cluster_node_storage'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'

interface ProvisionVolumesJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface VolumesOutput {
  [key: string]: TerraformOutputValue
}

export default class ProvisionVolumesJob extends Job {
  // This is the path to the file that is used to create the job
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionVolumesJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.volumesStartedAt = DateTime.now()
    cluster.volumesCompletedAt = null
    cluster.volumesErrorAt = null

    await cluster.save()

    try {
      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.VOLUMES)

      const controlPlaneServerIds = this.buildServerIdsMap(cluster.nodes.filter(n => n.type === 'master'))
      const workerServerIds = this.buildServerIdsMap(cluster.nodes.filter(n => n.type === 'worker'))

      const executor = new TerraformExecutor(cluster.id, 'volumes')
        .vars({
          ...cluster.cloudProvider?.getTerraformCredentials(),
          cluster_name: cluster.subdomainIdentifier,
          control_planes_volume_size: cluster.controlPlanesVolumeSize,
          workers_volume_size: cluster.workersVolumeSize,
          control_plane_server_ids: JSON.stringify(controlPlaneServerIds),
          worker_server_ids: JSON.stringify(workerServerIds),
          location: cluster.location
        })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as VolumesOutput

      await this.createOrUpdateVolumes(cluster.id, output)

      cluster.volumesCompletedAt = DateTime.now()
      cluster.dnsStartedAt = DateTime.now()

      await cluster.save()

    } catch (error) {
      cluster.volumesErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(_payload: ProvisionVolumesJobPayload) { }

  private buildServerIdsMap(nodes: any[]): Record<string, string> {
    const serverIds: Record<string, string> = {}

    for (const node of nodes) {
      if (node.providerId) {
        serverIds[node.id] = node.providerId
      }
    }

    return serverIds
  }

  private async createOrUpdateVolumes(
    clusterId: string,
    output: VolumesOutput
  ): Promise<void> {
    const cluster = await Cluster.query()
      .where('id', clusterId)
      .preload('nodes', (query) => {
        query.preload('storages')
      })
      .firstOrFail()

    for (const node of cluster.nodes) {
      const nodeType = node.type === 'master' ? 'control_plane' : 'worker'
      const volumeIdKey = `${nodeType}_${node.slug}_volume_id`
      const attachmentIdKey = `${nodeType}_${node.slug}_attachment_id`

      const volumeId = output[volumeIdKey]?.value as string
      const attachmentId = output[attachmentIdKey]?.value as string

      if (volumeId && attachmentId) {
        await this.createOrUpdateNodeStorage(node.id, volumeId, attachmentId)
      }
    }
  }

  private async createOrUpdateNodeStorage(
    nodeId: string,
    volumeId: string,
    attachmentId: string
  ): Promise<ClusterNodeStorage> {
    let storage = await ClusterNodeStorage.query()
      .where('cluster_node_id', nodeId)
      .first()

    if (!storage) {
      storage = new ClusterNodeStorage()
      storage.clusterNodeId = nodeId
    }

    storage.providerId = volumeId
    storage.providerMountId = attachmentId
    storage.status = 'healthy'
    await storage.save()

    return storage
  }
}