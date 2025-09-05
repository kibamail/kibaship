import { DateTime } from 'luxon'
import {
  BaseModel,
  beforeCreate,
  column,
  hasMany,
  belongsTo,
  hasOne,
  afterUpdate,
  computed,
} from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import { SshKeyService } from '#services/ssh/ssh_key_service'
import { TerraformStage } from '#services/terraform/terraform_executor'
import { talosVersion } from '#config/app'
import Project from './project.js'
import ClusterNode from './cluster_node.js'
import ClusterSshKey from './cluster_ssh_key.js'
import ClusterLoadBalancer from './cluster_load_balancer.js'
import ClusterNodeStorage from './cluster_node_storage.js'
import CloudProvider from './cloud_provider.js'
import Workspace from './workspace.js'
import type { HasMany, BelongsTo, HasOne } from '@adonisjs/lucid/types/relations'
import type { TransactionClientContract } from '@adonisjs/lucid/types/database'
import redis from '@adonisjs/redis/services/main'
import logger from '@adonisjs/core/services/logger'

export enum ProvisioningStepName {
  TALOS_IMAGE = 'talosImage',
  NETWORKING = 'networking',
  SSH_KEYS = 'sshKeys',
  LOAD_BALANCERS = 'loadBalancers',
  SERVERS = 'servers',
  VOLUMES = 'volumes',
  K8S = 'k8s',
  OPERATOR = 'operator',
}

export type ProvisioningStepStatus = 'pending' | 'in_progress' | 'completed' | 'failed'

export interface ProvisioningStep {
  status: ProvisioningStepStatus
  startedAt?: string
  completedAt?: string
  errorMessage?: string
  terraformState?: string
}

export interface ClusterProvisioningProgress {
  step: ProvisioningStepName
  status: 'pending' | 'in_progress' | 'completed' | 'failed'
  startedAt?: string
  completedAt?: string
  steps: {
    sshKeys: ProvisioningStep
    networking: ProvisioningStep
    loadBalancers: ProvisioningStep
    servers: ProvisioningStep
    volumes: ProvisioningStep
    kubernetesCluster: ProvisioningStep
    kibashipOperator: ProvisioningStep
  }
}

export type ClusterStatus = 'provisioning' | 'healthy' | 'unhealthy'
export type ClusterKind = 'all_purpose' | 'volume_storage' | 'pipelines'

export default class Cluster extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare location: string

  @column()
  declare controlPlaneEndpoint: string

  @column()
  declare subdomainIdentifier: string

  @column()
  declare kind: ClusterKind

  @column()
  declare workspaceId: string | null


  @column()
  declare cloudProviderId: string | null

  @column()
  declare serverType: string

  @column()
  declare status: ClusterStatus

  @column()
  declare providerNetworkId: string | null

  @column()
  declare providerSubnetId: string | null

  @column()
  declare providerImageId: string | null

  @column()
  declare talosVersion: string

  @column()
  declare networkIpRange: string | null

  @column()
  declare subnetIpRange: string | null

  @column()
  declare publicDomain: string | null

  @column()
  declare controlPlanesVolumeSize: number

  @column()
  declare workersVolumeSize: number

  @column.dateTime()
  declare deletedAt: DateTime | null

  @column.dateTime()
  declare dnsStartedAt: DateTime | null

  @column.dateTime()
  declare dnsCompletedAt: DateTime | null

  @column.dateTime()
  declare dnsLastCheckedAt: DateTime | null

  @column.dateTime()
  declare dnsErrorAt: DateTime | null

  @column.dateTime()
  declare talosImageStartedAt: DateTime | null

  @column.dateTime()
  declare talosImageCompletedAt: DateTime | null

  @column.dateTime()
  declare talosImageErrorAt: DateTime | null

  @column.dateTime()
  declare networkingStartedAt: DateTime | null

  @column.dateTime()
  declare networkingCompletedAt: DateTime | null

  @column.dateTime()
  declare networkingErrorAt: DateTime | null

  @column.dateTime()
  declare sshKeysStartedAt: DateTime | null

  @column.dateTime()
  declare sshKeysCompletedAt: DateTime | null

  @column.dateTime()
  declare sshKeysErrorAt: DateTime | null

  @column.dateTime()
  declare loadBalancersStartedAt: DateTime | null

  @column.dateTime()
  declare loadBalancersCompletedAt: DateTime | null

  @column.dateTime()
  declare loadBalancersErrorAt: DateTime | null

  @column.dateTime()
  declare serversStartedAt: DateTime | null

  @column.dateTime()
  declare serversCompletedAt: DateTime | null

  @column.dateTime()
  declare serversErrorAt: DateTime | null

  @column.dateTime()
  declare volumesStartedAt: DateTime | null

  @column.dateTime()
  declare volumesCompletedAt: DateTime | null

  @column.dateTime()
  declare volumesErrorAt: DateTime | null

  @column.dateTime()
  declare kubernetesClusterStartedAt: DateTime | null

  @column.dateTime()
  declare kubernetesClusterCompletedAt: DateTime | null

  @column.dateTime()
  declare kubernetesClusterErrorAt: DateTime | null

  @column.dateTime()
  declare kibashipOperatorStartedAt: DateTime | null

  @column.dateTime()
  declare kibashipOperatorCompletedAt: DateTime | null

  @column.dateTime()
  declare kibashipOperatorErrorAt: DateTime | null

  @column()
  declare currentProvisioningStep: string | null

  @column()
  declare overallProvisioningStatus: string | null

  @column.dateTime()
  declare provisioningStartedAt: DateTime | null

  @column.dateTime()
  declare provisioningCompletedAt: DateTime | null

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @hasMany(() => Project)
  declare projects: HasMany<typeof Project>

  @hasMany(() => ClusterNode)
  declare nodes: HasMany<typeof ClusterNode>

  @hasOne(() => ClusterSshKey)
  declare sshKey: HasOne<typeof ClusterSshKey>

  @hasMany(() => ClusterLoadBalancer)
  declare loadBalancers: HasMany<typeof ClusterLoadBalancer>

  @belongsTo(() => CloudProvider)
  declare cloudProvider: BelongsTo<typeof CloudProvider>

  @belongsTo(() => Workspace)
  declare workspace: BelongsTo<typeof Workspace>

  @beforeCreate()
  public static async generateId(cluster: Cluster) {
    cluster.id = randomUUID()
  }

  @afterUpdate()
  public static async publishUpdate(cluster: Cluster) {
    const pub = await redis.publish(
      'cluster:updated',
      JSON.stringify({
        id: cluster.id,
      })
    )

    logger.info(`Published cluster update for ${cluster.id}: ${pub}`)
  }

  public static async getNextAvailableIpRange(userId: string): Promise<string> {
    const maxRange = 16
    
    for (let i = 0; i <= maxRange; i++) {
      const ipRange = `10.${219 + i}.0.0/16`
      const existingCluster = await Cluster.query()
        .where('networkIpRange', ipRange)
        .whereHas('workspace', (workspaceQuery) => {
          workspaceQuery.where('userId', userId)
        })
        .whereNull('deletedAt')
        .first()
      
      if (!existingCluster) {
        return ipRange
      }
    }
    
    throw new Error('No available IP ranges. All ranges from 10.219.0.0/16 to 10.223.0.0/16 are in use for this user.')
  }

  public static async createWithInfrastructure(
    data: {
      subdomain_identifier: string
      cloud_provider_id: string
      region: string
      control_plane_nodes_count: number
      worker_nodes_count: number
      server_type: string
      control_planes_volume_size: number
      workers_volume_size: number
    },
    workspaceId: string,
    trx: TransactionClientContract
  ): Promise<Cluster> {
    const cluster = new Cluster()
    cluster.location = data.region
    cluster.cloudProviderId = data.cloud_provider_id
    cluster.workspaceId = workspaceId
    cluster.status = 'provisioning'
    cluster.kind = 'all_purpose'
    cluster.subdomainIdentifier = data.subdomain_identifier
    cluster.controlPlaneEndpoint = ''
    cluster.serverType = data.server_type
    cluster.controlPlanesVolumeSize = data.control_planes_volume_size
    cluster.workersVolumeSize = data.workers_volume_size
    cluster.talosVersion = talosVersion
    const workspace = await Workspace.findOrFail(workspaceId)
    cluster.networkIpRange = await Cluster.getNextAvailableIpRange(workspace.userId)
    cluster.subnetIpRange = cluster.networkIpRange
    cluster.useTransaction(trx)

    await cluster.save()

    await cluster.createSshKey(trx)
    await cluster.createNodes(
      data.control_plane_nodes_count,
      data.worker_nodes_count,
      data.server_type,
      trx,
      data.control_planes_volume_size,
      data.workers_volume_size
    )

    return cluster
  }

  public async resetProvisionProgress() {
    this.status = 'provisioning'

    await this.save()
  }

  public async createSshKey(trx: TransactionClientContract): Promise<ClusterSshKey> {
    const sshKeyPair = await SshKeyService.generateEd25519KeyPair()

    if (!sshKeyPair.publicKey || !sshKeyPair.privateKey) {
      throw new Error('Failed to generate SSH key pair')
    }

    const clusterSshKey = new ClusterSshKey()
    clusterSshKey.clusterId = this.id
    clusterSshKey.publicKey = sshKeyPair.publicKey
    clusterSshKey.privateKey = sshKeyPair.privateKey
    clusterSshKey.useTransaction(trx)
    await clusterSshKey.save()

    return clusterSshKey
  }

  public async createNodes(
    controlPlaneCount: number,
    workerCount: number,
    serverType: string,
    trx: TransactionClientContract,
    controlPlanesVolumeSize: number,
    workersVolumeSize: number
  ): Promise<void> {
    const nodes: ClusterNode[] = []
    const storages: ClusterNodeStorage[] = []

    for (let i = 0; i < controlPlaneCount; i++) {
      const controlPlaneNode = new ClusterNode()
      controlPlaneNode.clusterId = this.id
      controlPlaneNode.type = 'master'
      controlPlaneNode.status = 'provisioning'
      controlPlaneNode.serverType = serverType
      controlPlaneNode.useTransaction(trx)
      nodes.push(controlPlaneNode)
    }

    for (let i = 0; i < workerCount; i++) {
      const workerNode = new ClusterNode()
      workerNode.clusterId = this.id
      workerNode.type = 'worker'
      workerNode.status = 'provisioning'
      workerNode.serverType = serverType
      workerNode.useTransaction(trx)
      nodes.push(workerNode)
    }

    // Save nodes first
    await Promise.all(nodes.map((node) => node.save()))

    // Create storage for each node
    for (const node of nodes) {
      const storage = new ClusterNodeStorage()
      storage.clusterNodeId = node.id
      storage.status = 'provisioning'
      storage.size = node.type === 'master' ? controlPlanesVolumeSize : workersVolumeSize
      storage.useTransaction(trx)
      storages.push(storage)
    }

    await Promise.all(storages.map((storage) => storage.save()))
  }

  public static complete(clusterId: string) {
    return Cluster.query()
      .where('id', clusterId)
      .preload('cloudProvider')
      .preload('loadBalancers')
      .preload('nodes')
      .preload('sshKey')
      .preload('nodes', (nodesQuery) => nodesQuery.preload('storages'))
      .first()
  }

  public getStepStatus(stage: TerraformStage): ProvisioningStepStatus {
    switch (stage) {
      case 'talos-image':
        if (this.talosImageCompletedAt) return 'completed'
        if (this.talosImageErrorAt) return 'failed'
        if (this.talosImageStartedAt) return 'in_progress'
        return 'pending'

      case 'network':
        if (this.networkingCompletedAt) return 'completed'
        if (this.networkingErrorAt) return 'failed'
        if (this.networkingStartedAt) return 'in_progress'
        return 'pending'

      case 'ssh-keys':
        if (this.sshKeysCompletedAt) return 'completed'
        if (this.sshKeysErrorAt) return 'failed'
        if (this.sshKeysStartedAt) return 'in_progress'
        return 'pending'

      case 'load-balancers':
        if (this.loadBalancersCompletedAt) return 'completed'
        if (this.loadBalancersErrorAt) return 'failed'
        if (this.loadBalancersStartedAt) return 'in_progress'
        return 'pending'

      case 'servers':
        if (this.serversCompletedAt) return 'completed'
        if (this.serversErrorAt) return 'failed'
        if (this.serversStartedAt) return 'in_progress'
        return 'pending'

      case 'volumes':
        if (this.volumesCompletedAt) return 'completed'
        if (this.volumesErrorAt) return 'failed'
        if (this.volumesStartedAt) return 'in_progress'
        return 'pending'
      
      case 'dns':
        if (this.dnsCompletedAt) return 'completed'
        if (this.dnsErrorAt) return 'failed'
        if (this.dnsStartedAt) return 'in_progress'
        return 'pending'

      case 'kubernetes':
        if (this.kubernetesClusterCompletedAt) return 'completed'
        if (this.kubernetesClusterErrorAt) return 'failed'
        if (this.kubernetesClusterStartedAt) return 'in_progress'
        return 'pending'

      default:
        return 'pending'
    }
  }

  @computed()
  public get progress() {
    const stages: TerraformStage[] = [
      'talos-image',
      'network',
      'ssh-keys',
      'load-balancers',
      'servers',
      'volumes',
      'dns',
      'kubernetes',
    ]

    return stages.reduce((acc, stage) => {
      acc[stage] = this.getStepStatus(stage)
      return acc
    }, {} as Record<TerraformStage, ProvisioningStepStatus>)
  }

  @computed()
  public get provisioningStatus(): ProvisioningStepStatus {
    
    const stages: TerraformStage[] = [
      'talos-image',
      'network',
      'ssh-keys',
      'load-balancers',
      'servers',
      'volumes',
      'dns',
      'kubernetes',
    ]

    const statuses = stages.map(stage => this.getStepStatus(stage))

    if (statuses.some(status => status === 'failed')) {
      return 'failed'
    }

    if (statuses.some(status => status === 'in_progress')) {
      return 'in_progress'
    }

    if (statuses.every(status => status === 'completed')) {
      return 'completed'
    }

    return 'pending'
  }

  @computed()
  public get firstFailedStage(): TerraformStage | null {
    const stages: TerraformStage[] = [
      'talos-image',
      'network',
      'ssh-keys',
      'load-balancers',
      'servers',
      'volumes',
      'kubernetes',
    ]

    for (const stage of stages) {
      const status = this.getStepStatus(stage)
      if (status === 'failed') {
        return stage
      }
    }

    return null
  }
}
