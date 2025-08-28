import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, hasMany, belongsTo, hasOne } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import { SshKeyService } from '#services/ssh/ssh_key_service'
import Project from './project.js'
import ClusterNode from './cluster_node.js'
import ClusterSshKey from './cluster_ssh_key.js'
import ClusterLoadBalancer from './cluster_load_balancer.js'
import CloudProvider from './cloud_provider.js'
import type { HasMany, BelongsTo, HasOne } from '@adonisjs/lucid/types/relations'
import type { TransactionClientContract } from '@adonisjs/lucid/types/database'

export enum ProvisioningStepName {
  NETWORKING = 'networking',
  SSH_KEYS = 'sshKeys',
  LOAD_BALANCERS = 'loadBalancers',
  SERVERS = 'servers',
  VOLUMES = 'volumes',
  K8S = 'k8s',
  OPERATOR = 'operator'
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
  declare error: string

  @column()
  declare cloudProviderId: string | null

  @column()
  declare status: ClusterStatus

  @column()
  declare providerNetworkId: string | null

  @column()
  declare providerSubnetId: string | null

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
  declare networkingStartedAt: DateTime | null

  @column.dateTime()
  declare networkingCompletedAt: DateTime | null

  @column()
  declare networkingError: string | null

  @column.dateTime()
  declare sshKeysStartedAt: DateTime | null

  @column.dateTime()
  declare sshKeysCompletedAt: DateTime | null

  @column()
  declare sshKeysError: string | null

  @column.dateTime()
  declare loadBalancersStartedAt: DateTime | null

  @column.dateTime()
  declare loadBalancersCompletedAt: DateTime | null

  @column()
  declare loadBalancersError: string | null

  @column.dateTime()
  declare serversStartedAt: DateTime | null

  @column.dateTime()
  declare serversCompletedAt: DateTime | null

  @column()
  declare serversError: string | null

  @column.dateTime()
  declare volumesStartedAt: DateTime | null

  @column.dateTime()
  declare volumesCompletedAt: DateTime | null

  @column()
  declare volumesError: string | null

  @column.dateTime()
  declare kubernetesClusterStartedAt: DateTime | null

  @column.dateTime()
  declare kubernetesClusterCompletedAt: DateTime | null

  @column()
  declare kubernetesClusterError: string | null

  @column.dateTime()
  declare kibashipOperatorStartedAt: DateTime | null

  @column.dateTime()
  declare kibashipOperatorCompletedAt: DateTime | null

  @column()
  declare kibashipOperatorError: string | null

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

  @beforeCreate()
  public static async generateId(cluster: Cluster) {
    cluster.id = randomUUID()
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
    cluster.controlPlanesVolumeSize = data.control_planes_volume_size
    cluster.workersVolumeSize = data.workers_volume_size
    cluster.useTransaction(trx)

    await cluster.save()

    await cluster.createSshKey(trx)
    await cluster.createNodes(data.control_plane_nodes_count, data.worker_nodes_count, trx)

    return cluster
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
    trx: TransactionClientContract
  ): Promise<void> {
    const nodes: ClusterNode[] = []

    for (let i = 0; i < controlPlaneCount; i++) {
      const controlPlaneNode = new ClusterNode()
      controlPlaneNode.clusterId = this.id
      controlPlaneNode.type = 'master'
      controlPlaneNode.status = 'provisioning'
      controlPlaneNode.useTransaction(trx)
      nodes.push(controlPlaneNode)
    }

    for (let i = 0; i < workerCount; i++) {
      const workerNode = new ClusterNode()
      workerNode.clusterId = this.id
      workerNode.type = 'worker'
      workerNode.status = 'provisioning'
      workerNode.useTransaction(trx)
      nodes.push(workerNode)
    }

    await Promise.all(nodes.map(node => node.save()))
  }

  public static complete(clusterId: string) {
    return Cluster.query()
      .where('id', clusterId)
      .preload('cloudProvider')
      .preload('nodes')
      .preload('sshKey')
      .preload('nodes')
      .first()
  }
}
