import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import Cluster from '#models/cluster'
import ClusterNodeStorage from '#models/cluster_node_storage'
import { createExecutor } from '#services/terraform/main'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { TalosDetectionService } from '#services/talos/talos_detection_service'
import { DateTime } from 'luxon'
import ProvisionKubernetesConfigJob from './provision_kubernetes_config_job.js'

interface ProvisionVolumesJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string | string[]
  value: string | number | object
}

interface TalosDisk {
  bus_path: string
  cdrom: boolean
  dev_path: string
  io_size: number
  modalias: string
  model: string
  pretty_size: string
  readonly: boolean
  rotational: boolean
  secondary_disks: string[]
  sector_size: number
  serial: string
  size: number
  sub_system: string
  symlinks: string[]
  transport: string
  uuid: string
  wwid: string
}

interface VolumesOutput {
  // Individual volume outputs for each volume slug
  [volumeOutputKey: string]: TerraformOutputValue

  // Summary output with all volume IDs mapped by slug
  volume_ids: TerraformOutputValue & {
    value: Record<string, string>
  }

  // Talos disk discovery outputs mapped by node slug
  all_worker_disks: TerraformOutputValue & {
    value: Record<string, TalosDisk[]>
  }
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

      const executor = (await createExecutor(cluster.id, 'volumes')).vars({
        ...cluster.cloudProvider?.getTerraformCredentials(),
        cluster_name: cluster.subdomainIdentifier,
        location: cluster.location,
      })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as VolumesOutput

      await this.createOrUpdateVolumes(cluster.id, output)

      // After volumes are created, match disks with volume outputs
      const nodes = await cluster.related('nodes').query().preload('storages')
      const talosDetectionService = new TalosDetectionService(cluster.id)

      for (const node of nodes) {
        const result = await talosDetectionService.matchAndUpdateDisks(node, output)
        if (!result.success) {
          throw new Error(`Failed to match disks for ${node.slug}: ${result.error}`)
        }
      }

      cluster.volumesCompletedAt = DateTime.now()

      await cluster.save()

      await queue.dispatch(ProvisionKubernetesConfigJob, payload)
    } catch (error) {
      console.error(error)
      cluster.volumesErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(_payload: ProvisionVolumesJobPayload) {}

  private async createOrUpdateVolumes(clusterId: string, output: VolumesOutput): Promise<void> {
    const cluster = await Cluster.query()
      .where('id', clusterId)
      .preload('nodes', (query) => {
        query.preload('storages')
      })
      .firstOrFail()

    for (const node of cluster.nodes) {
      for (const storage of node.storages) {
        const volumeIdKey = `volume_${storage.slug}_id`
        const attachmentIdKey = `volume_${storage.slug}_attachment_id`

        const volumeId = output[volumeIdKey]?.value as string
        const attachmentId = output[attachmentIdKey]?.value as string

        if (volumeId && attachmentId) {
          await this.updateNodeStorage(storage, volumeId, attachmentId)
        }
      }
    }
  }

  private async updateNodeStorage(
    storage: ClusterNodeStorage,
    volumeId: string,
    attachmentId: string
  ): Promise<ClusterNodeStorage> {
    storage.providerId = volumeId
    storage.providerMountId = attachmentId
    storage.status = 'healthy'
    await storage.save()

    return storage
  }
}
