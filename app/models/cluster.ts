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
import encryption from '@adonisjs/core/services/encryption'
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
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'

export enum ProvisioningStepName {
  TALOS_IMAGE = 'talosImage',
  NETWORKING = 'networking',
  SSH_KEYS = 'sshKeys',
  LOAD_BALANCERS = 'loadBalancers',
  SERVERS = 'servers',
  VOLUMES = 'volumes',
  K8S = 'k8s',
  K8S_CONFIG = 'k8sConfig',
  OPERATOR = 'operator',
}

export type ProvisioningStepStatus = 'pending' | 'in_progress' | 'ready' | 'failed' | 'deleting'

export interface ProvisioningStep {
  status: ProvisioningStepStatus
  startedAt?: string
  completedAt?: string
  errorMessage?: string
  terraformState?: string
}

export interface ClusterProvisioningProgress {
  step: ProvisioningStepName
  status: 'pending' | 'in_progress' | 'ready' | 'failed'
  startedAt?: string
  completedAt?: string
  steps: {
    sshKeys: ProvisioningStep
    networking: ProvisioningStep
    loadBalancers: ProvisioningStep
    servers: ProvisioningStep
    volumes: ProvisioningStep
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

  @column()
  declare vlanId: number | null

  @column()
  declare vswitchId: number | null

  @column()
  declare vswitchSubnetIpRange: string | null

  @column()
  declare robotCloudProviderId: string | null

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
  declare kubernetesConfigStartedAt: DateTime | null

  @column.dateTime()
  declare kubernetesConfigCompletedAt: DateTime | null

  @column.dateTime()
  declare kubernetesConfigErrorAt: DateTime | null

  @column.dateTime()
  declare kubernetesBootStartedAt: DateTime | null

  @column.dateTime()
  declare kubernetesBootCompletedAt: DateTime | null

  @column.dateTime()
  declare kubernetesBootErrorAt: DateTime | null

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

  @column.dateTime()
  declare byocStartedAt: DateTime | null

  @column.dateTime()
  declare byocCompletedAt: DateTime | null

  @column.dateTime()
  declare byocErrorAt: DateTime | null

  @column.dateTime()
  declare bareMetalNetworkingStartedAt: DateTime | null

  @column.dateTime()
  declare bareMetalNetworkingCompletedAt: DateTime | null

  @column.dateTime()
  declare bareMetalNetworkingErrorAt: DateTime | null

  @column({
    prepare: (value) => (value ? encryption.encrypt(JSON.stringify(value)) : null),
    consume: (value) => (value ? JSON.parse(encryption.decrypt(value) || '{}') : null),
    serializeAs: null,
  })
  declare kubeconfig: {
    host: string
    clientCertificate: string
    clientKey: string
    clusterCaCertificate: string
  } | null

  @column({
    prepare: (value) => (value ? encryption.encrypt(JSON.stringify(value)) : null),
    consume: (value) => (value ? JSON.parse(encryption.decrypt(value) || '{}') : null),
    serializeAs: null,
  })
  declare talosConfig: {
    ca_certificate: string
    client_certificate: string
    client_key: string
  } | null

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

  @belongsTo(() => CloudProvider, {
    foreignKey: 'robotCloudProviderId',
  })
  declare robotCloudProvider: BelongsTo<typeof CloudProvider>

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

    throw new Error(
      'No available IP ranges. All ranges from 10.219.0.0/16 to 10.223.0.0/16 are in use for this user.'
    )
  }

  public static async createWithInfrastructure(
    data: {
      subdomain_identifier: string
      cloud_provider_id: string
      region: string
      control_plane_nodes_count: number
      worker_nodes_count: number
      server_type: string
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
    cluster.controlPlanesVolumeSize = 0
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
      0, // todo: remove logic for control plane volumes
      data.workers_volume_size
    )

    return cluster
  }

  public static async createWithHetznerRobotServers(
    data: {
      subdomain_identifier: string
      cloud_provider_id: string
      robot_cloud_provider_id: string
      region: string
      robot_server_numbers: number[]
      servers: Array<{
        server_number: number
        server_ip: string
        server_name: string
      }>
      vlan_id: number | null
      vswitch_id: number | null
      vlan_name: string | null
    },
    workspaceId: string,
    trx: TransactionClientContract
  ): Promise<Cluster> {
    const cluster = new Cluster()
    cluster.location = data.region
    cluster.cloudProviderId = data.cloud_provider_id
    cluster.robotCloudProviderId = data.robot_cloud_provider_id
    cluster.workspaceId = workspaceId
    cluster.status = 'provisioning'
    cluster.kind = 'all_purpose'
    cluster.subdomainIdentifier = data.subdomain_identifier
    cluster.controlPlaneEndpoint = ''
    cluster.serverType = 'bare-metal'
    cluster.controlPlanesVolumeSize = 0
    cluster.workersVolumeSize = 0
    cluster.talosVersion = talosVersion
    cluster.vlanId = data.vlan_id
    cluster.vswitchId = data.vswitch_id
    const workspace = await Workspace.findOrFail(workspaceId)
    cluster.networkIpRange = await Cluster.getNextAvailableIpRange(workspace.userId)
    // For bare metal, we need two subnets within the network /16 range:
    // 1. vSwitch subnet (x.x.1.0/24) - for server private IPs
    // 2. Load balancer subnet (x.x.2.0/24) - for the ingress load balancer
    const networkParts = cluster.networkIpRange.split('/')
    const networkBase = networkParts[0].split('.')
    cluster.vswitchSubnetIpRange = `${networkBase[0]}.${networkBase[1]}.1.0/24`
    cluster.subnetIpRange = `${networkBase[0]}.${networkBase[1]}.2.0/24`
    cluster.useTransaction(trx)

    await cluster.save()

    await cluster.createSshKey(trx)

    // Create nodes based on server count
    // If only 3 servers, all are masters
    // If more than 3, first 3 are masters, rest are workers
    const totalServers = data.robot_server_numbers.length
    const masterCount = totalServers === 3 ? 3 : 3
    const workerCount = totalServers === 3 ? 0 : totalServers - 3

    // Generate private IPs from vSwitch subnet range
    const vswitchSubnetBase = cluster.vswitchSubnetIpRange!.split('/')[0].split('.')
    const generatePrivateIp = (index: number) => {
      // Start at .10, then .20, .30, etc.
      const lastOctet = (index + 1) * 10
      return `${vswitchSubnetBase[0]}.${vswitchSubnetBase[1]}.${vswitchSubnetBase[2]}.${lastOctet}`
    }

    const nodes: ClusterNode[] = []

    // Create master nodes
    for (let i = 0; i < masterCount; i++) {
      const server = data.servers.find((s) => s.server_number === data.robot_server_numbers[i])
      const masterNode = new ClusterNode()
      masterNode.clusterId = cluster.id
      masterNode.type = 'master'
      masterNode.status = 'provisioning'
      masterNode.serverType = 'bare-metal'
      masterNode.providerId = data.robot_server_numbers[i].toString()
      masterNode.ipv4Address = server?.server_ip || null
      masterNode.privateIpv4Address = generatePrivateIp(i)
      masterNode.useTransaction(trx)
      nodes.push(masterNode)
    }

    // Create worker nodes
    for (let i = 0; i < workerCount; i++) {
      const server = data.servers.find(
        (s) => s.server_number === data.robot_server_numbers[masterCount + i]
      )
      const workerNode = new ClusterNode()
      workerNode.clusterId = cluster.id
      workerNode.type = 'worker'
      workerNode.status = 'provisioning'
      workerNode.serverType = 'bare-metal'
      workerNode.providerId = data.robot_server_numbers[masterCount + i].toString()
      workerNode.ipv4Address = server?.server_ip || null
      workerNode.privateIpv4Address = generatePrivateIp(masterCount + i)
      workerNode.useTransaction(trx)
      nodes.push(workerNode)
    }

    await Promise.all(nodes.map((node) => node.save()))

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
      const size = node.type === 'master' ? controlPlanesVolumeSize : workersVolumeSize

      if (size > 0) {
        const storage = new ClusterNodeStorage()
        storage.clusterNodeId = node.id
        storage.status = 'provisioning'
        storage.size = size
        storage.useTransaction(trx)
        storages.push(storage)
      }
    }

    await Promise.all(storages.map((storage) => storage.save()))
  }

  public static complete(clusterId: string) {
    return Cluster.query()
      .where('id', clusterId)
      .preload('cloudProvider')
      .preload('robotCloudProvider')
      .preload('loadBalancers')
      .preload('nodes')
      .preload('sshKey')
      .preload('workspace', (query) => {
        query.preload('user')
      })
      .preload('nodes', (nodesQuery) => nodesQuery.preload('storages'))
      .first()
  }

  public getStepStatus(stage: TerraformStage): ProvisioningStepStatus {
    switch (stage) {
      case 'talos-image':
        if (this.talosImageCompletedAt) return 'ready'
        if (this.talosImageErrorAt) return 'failed'
        if (this.talosImageStartedAt) return 'in_progress'
        return 'pending'

      case 'network':
        if (this.networkingCompletedAt) return 'ready'
        if (this.networkingErrorAt) return 'failed'
        if (this.networkingStartedAt) return 'in_progress'
        return 'pending'

      case 'ssh-keys':
        if (this.sshKeysCompletedAt) return 'ready'
        if (this.sshKeysErrorAt) return 'failed'
        if (this.sshKeysStartedAt) return 'in_progress'
        return 'pending'

      case 'load-balancers':
        if (this.loadBalancersCompletedAt) return 'ready'
        if (this.loadBalancersErrorAt) return 'failed'
        if (this.loadBalancersStartedAt) return 'in_progress'
        return 'pending'

      case 'servers':
        if (this.serversCompletedAt) return 'ready'
        if (this.serversErrorAt) return 'failed'
        if (this.serversStartedAt) return 'in_progress'
        return 'pending'

      case 'volumes':
        if (this.volumesCompletedAt) return 'ready'
        if (this.volumesErrorAt) return 'failed'
        if (this.volumesStartedAt) return 'in_progress'
        return 'pending'

      case 'kubernetes-config':
        if (this.kubernetesConfigCompletedAt) return 'ready'
        if (this.kubernetesConfigErrorAt) return 'failed'
        if (this.kubernetesConfigStartedAt) return 'in_progress'
        return 'pending'

      case 'kubernetes-boot':
        if (this.kubernetesBootCompletedAt) return 'ready'
        if (this.kubernetesBootErrorAt) return 'failed'
        if (this.kubernetesBootStartedAt) return 'in_progress'
        return 'pending'

      case 'kibaship-operator':
        if (this.kibashipOperatorCompletedAt) return 'ready'
        if (this.kibashipOperatorErrorAt) return 'failed'
        if (this.kibashipOperatorStartedAt) return 'in_progress'
        return 'pending'

      case 'dns':
        if (this.dnsCompletedAt) return 'ready'
        if (this.dnsErrorAt) return 'failed'
        if (this.dnsStartedAt) return 'in_progress'
        return 'pending'

      default:
        return 'pending'
    }
  }

  public getBareMetalNetworkingStatus(): ProvisioningStepStatus {
    if (this.bareMetalNetworkingCompletedAt) return 'ready'
    if (this.bareMetalNetworkingErrorAt) return 'failed'
    if (this.bareMetalNetworkingStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalCloudLoadBalancerStatus(): ProvisioningStepStatus {
    if (this.networkingCompletedAt) return 'ready'
    if (this.networkingErrorAt) return 'failed'
    if (this.networkingStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalTalosImageStatus(): ProvisioningStepStatus {
    if (this.talosImageCompletedAt) return 'ready'
    if (this.talosImageErrorAt) return 'failed'
    if (this.talosImageStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalServersBootstrapStatus(): ProvisioningStepStatus {
    if (this.serversCompletedAt) return 'ready'
    if (this.serversErrorAt) return 'failed'
    if (this.serversStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalKubernetesConfigStatus(): ProvisioningStepStatus {
    if (this.kubernetesConfigCompletedAt) return 'ready'
    if (this.kubernetesConfigErrorAt) return 'failed'
    if (this.kubernetesConfigStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalKubernetesBootStatus(): ProvisioningStepStatus {
    if (this.kubernetesBootCompletedAt) return 'ready'
    if (this.kubernetesBootErrorAt) return 'failed'
    if (this.kubernetesBootStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalKibashipOperatorStatus(): ProvisioningStepStatus {
    if (this.kibashipOperatorCompletedAt) return 'ready'
    if (this.kibashipOperatorErrorAt) return 'failed'
    if (this.kibashipOperatorStartedAt) return 'in_progress'
    return 'pending'
  }

  public getBareMetalDnsStatus(): ProvisioningStepStatus {
    if (this.dnsCompletedAt) return 'ready'
    if (this.dnsErrorAt) return 'failed'
    if (this.dnsStartedAt) return 'in_progress'
    return 'pending'
  }

  @computed()
  public get bareMetalProgress() {
    return {
      'bare-metal-networking': this.getBareMetalNetworkingStatus(),
      'bare-metal-cloud-load-balancer': this.getBareMetalCloudLoadBalancerStatus(),
      'bare-metal-talos-image': this.getBareMetalTalosImageStatus(),
      'bare-metal-servers-bootstrap': this.getBareMetalServersBootstrapStatus(),
      'kubernetes-config': this.getBareMetalKubernetesConfigStatus(),
      'kubernetes-boot': this.getBareMetalKubernetesBootStatus(),
      'dns': this.getBareMetalDnsStatus(),
      'kibaship-operator': this.getBareMetalKibashipOperatorStatus(),
    }
  }

  @computed()
  public get progress() {
    // If this is a bare metal cluster, return bare metal specific progress
    if (this.robotCloudProviderId) {
      return this.bareMetalProgress
    }

    const stages: TerraformStage[] = [
      'talos-image',
      'network',
      'ssh-keys',
      'load-balancers',
      'servers',
      'volumes',
      'kubernetes-config',
      'kubernetes-boot',
      'dns',
      'kibaship-operator',
    ]

    return stages.reduce(
      (acc, stage) => {
        acc[stage] = this.getStepStatus(stage)
        return acc
      },
      {} as Record<TerraformStage, ProvisioningStepStatus>
    )
  }

  @computed()
  public get provisioningStatus(): ProvisioningStepStatus {
    if (this.deletedAt) {
      return 'deleting'
    }

    // If this is a bare metal cluster, use bare metal specific status
    if (this.robotCloudProviderId) {
      const bareMetalStatuses = Object.values(this.bareMetalProgress)

      if (bareMetalStatuses.some((status) => status === 'failed')) {
        return 'failed'
      }

      if (bareMetalStatuses.some((status) => status === 'in_progress')) {
        return 'in_progress'
      }

      if (bareMetalStatuses.every((status) => status === 'ready')) {
        return 'ready'
      }

      return 'pending'
    }

    let stages: TerraformStage[] = [
      'talos-image',
      'network',
      'ssh-keys',
      'load-balancers',
      'servers',
      'volumes',
      'kubernetes-config',
      'kubernetes-boot',
      'kibaship-operator',
      'dns',
    ]

    const statuses = stages.map((stage) => this.getStepStatus(stage))

    if (statuses.some((status) => status === 'failed')) {
      return 'failed'
    }

    if (statuses.some((status) => status === 'in_progress')) {
      return 'in_progress'
    }

    if (statuses.every((status) => status === 'ready')) {
      return 'ready'
    }

    return 'pending'
  }

  @computed()
  public get firstFailedStage(): TerraformStage | null {
    // Handle Hetzner Robot (bare metal) clusters separately
    if (this.robotCloudProviderId) {
      if (this.getBareMetalNetworkingStatus() === 'failed') {
        return 'bare-metal-networking' as TerraformStage
      }

      if (this.getBareMetalCloudLoadBalancerStatus() === 'failed') {
        return 'bare-metal-cloud-load-balancer' as TerraformStage
      }

      if (this.getBareMetalTalosImageStatus() === 'failed') {
        return 'bare-metal-talos-image' as TerraformStage
      }

      if (this.getBareMetalServersBootstrapStatus() === 'failed') {
        return 'bare-metal-servers-bootstrap' as TerraformStage
      }

      if (this.getBareMetalKubernetesConfigStatus() === 'failed') {
        return 'kubernetes-config' as TerraformStage
      }

      if (this.getBareMetalKubernetesBootStatus() === 'failed') {
        return 'kubernetes-boot' as TerraformStage
      }

      if (this.getBareMetalDnsStatus() === 'failed') {
        return 'dns' as TerraformStage
      }

      if (this.getBareMetalKibashipOperatorStatus() === 'failed') {
        return 'kibaship-operator' as TerraformStage
      }

      return null
    }

    // Handle standard cloud provider stages
    const stages: TerraformStage[] = [
      'talos-image',
      'network',
      'ssh-keys',
      'load-balancers',
      'servers',
      'volumes',
      'kubernetes-config',
      'kubernetes-boot',
      'dns',
      'kibaship-operator',
    ]

    for (const stage of stages) {
      const status = this.getStepStatus(stage)
      if (status === 'failed') {
        return stage
      }
    }

    return null
  }

  /**
   * Get the flag for this cluster based on its cloud provider and location.
   * Returns the flag path for the region where this cluster is deployed.
   * If cloudProvider is not loaded, checks if it's a BYOC cluster and uses BYOC regions.
   * Otherwise returns the US flag as default.
   */
  @computed()
  public get region(): { flag: string; name: string } {
    try {
      if (!this.cloudProvider || this.cloudProvider.type === 'byoc') {
        const regionsByContinent = CloudProviderDefinitions.regions('byoc')
        const allRegions = Object.values(regionsByContinent).flat()
        const region = allRegions.find((region) => region.slug === this.location)

        return (
          region || {
            name: 'Unknown',
            flag: '/flags/us.svg',
          }
        )
      }

      if (!this.cloudProvider) {
        return {
          name: 'Unknown',
          flag: '/flags/us.svg',
        }
      }

      const regionsByContinent = CloudProviderDefinitions.regions(this.cloudProvider.type)

      const allRegions = Object.values(regionsByContinent).flat()

      const region = allRegions.find((region) => region.slug === this.location)

      return (
        region || {
          name: 'Unknown',
          flag: '/flags/us.svg',
        }
      )
    } catch (error) {
      return {
        name: 'Unknown',
        flag: '/flags/us.svg',
      }
    }
  }
}
