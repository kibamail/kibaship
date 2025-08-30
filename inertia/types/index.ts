import { TerraformStage } from '#services/terraform/terraform_executor'
import { UserProfile } from '@kibamail/auth-sdk'

export interface User {
  id: string
  name: string
  email: string
  email_verified_at?: string
}

export interface Workspace {
  id: string
  name: string
  slug: string
  user_id: string
  created_at: string
  updated_at: string
  projects?: Project[]
}

export type ClusterKind = 'all_purpose' | 'volume_storage' | 'pipelines'

export interface Project {
  id: string
  name: string
  slug: string
  workspace_id: string
  created_at: string
  updated_at: string
  cluster: Cluster
  environments?: Environment[]
  applications: Application[]
}

export interface Environment {
  id: string
  slug: string
  project_id: string
  created_at: string
  updated_at: string
}

export interface WorkspaceMembership {
  id: string
  workspace_id: string
  user_id: string | null
  email: string
  role: 'developer' | 'admin'
  created_at: string
  updated_at: string
  user?: User
  workspace?: Workspace
  projects?: Project[]
}

export interface WorkspaceMembershipProject {
  id: string
  workspace_membership_id: string
  project_id: string
  created_at: string
  updated_at: string
}

export type CloudProviderType =
  | 'aws'
  | 'hetzner'
  | 'leaseweb'
  | 'google_cloud'
  | 'digital_ocean'
  | 'linode'
  | 'vultr'
  | 'ovh'

export interface CloudProvider {
  id: string
  name: string
  type: CloudProviderType
  workspace_id: string
  created_at: string
  updated_at: string
}

export interface CloudProviderRegion {
  name: string
  slug: string
  flag: string
  availableServerTypes: Record<string, boolean>
}

export type CloudProviderRegionsByContinent = Record<string, CloudProviderRegion[]>

export type ClusterStatus = 'provisioning' | 'healthy' | 'unhealthy'

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

export type ClusterNodeType = 'master' | 'worker'
export type ClusterNodeStatus = 'provisioning' | 'healthy' | 'unhealthy'
export type ClusterNodeStorageStatus = 'provisioning' | 'healthy' | 'unhealthy'
export type ClusterLoadBalancerType = 'cluster' | 'ingress' | 'tcp' | 'udp'

export interface Cluster {
  id: string
  location: string
  controlPlaneEndpoint: string
  subdomainIdentifier: string
  kind: ClusterKind
  workspaceId: string | null
  error: string
  cloudProviderId: string | null
  status: ClusterStatus
  providerNetworkId: string | null
  providerSubnetId: string | null
  networkIpRange: string | null
  subnetIpRange: string | null
  publicDomain: string | null
  controlPlanesVolumeSize: number
  workersVolumeSize: number
  networkingStartedAt: string | null
  networkingCompletedAt: string | null
  networkingError: string | null
  networkingErrorAt: string | null
  sshKeysStartedAt: string | null
  sshKeysCompletedAt: string | null
  sshKeysError: string | null
  sshKeysErrorAt: string | null
  loadBalancersStartedAt: string | null
  loadBalancersCompletedAt: string | null
  loadBalancersError: string | null
  loadBalancersErrorAt: string | null
  serversStartedAt: string | null
  serversCompletedAt: string | null
  serversError: string | null
  serversErrorAt: string | null
  volumesStartedAt: string | null
  volumesCompletedAt: string | null
  volumesError: string | null
  volumesErrorAt: string | null
  kubernetesClusterStartedAt: string | null
  kubernetesClusterCompletedAt: string | null
  kubernetesClusterError: string | null
  kubernetesClusterErrorAt: string | null
  kibashipOperatorStartedAt: string | null
  kibashipOperatorCompletedAt: string | null
  kibashipOperatorError: string | null
  kibashipOperatorErrorAt: string | null
  currentProvisioningStep: string | null
  overallProvisioningStatus: string | null
  provisioningStartedAt: string | null
  provisioningCompletedAt: string | null
  createdAt: string
  updatedAt: string
  projects?: Project[]
  nodes?: ClusterNode[]
  sshKey?: ClusterSshKey
  loadBalancers?: ClusterLoadBalancer[]
  cloudProvider?: CloudProvider
}

export interface ProvisioningStepInfo {
  stage: TerraformStage
  title: string
  description: string
  icon: React.ReactNode
}


export interface ClusterNode {
  id: string
  cluster_id: string
  type: ClusterNodeType
  status: ClusterNodeStatus
  ipv4_address: string | null
  ipv6_address: string | null
  private_ipv4_address: string | null
  created_at: string
  updated_at: string
  cluster?: Cluster
  storages?: ClusterNodeStorage[]
}

export interface ClusterNodeStorage {
  id: string
  cluster_node_id: string
  provider_id: string | null
  provider_mount_id: string | null
  status: ClusterNodeStorageStatus
  created_at: string
  updated_at: string
  cluster_node?: ClusterNode
}

export interface ClusterSshKey {
  id: string
  cluster_id: string
  public_key: string
  private_key: string
  provider_id: string | null
  created_at: string
  updated_at: string
  cluster?: Cluster
}

export interface ClusterLoadBalancer {
  id: string
  cluster_id: string
  type: ClusterLoadBalancerType
  public_ipv4_address: string | null
  private_ipv4_address: string | null
  provider_id: string | null
  created_at: string
  updated_at: string
  cluster?: Cluster
}

export interface CloudProviderCredentialField {
  name: string
  label: string
  type: 'text' | 'password' | 'textarea'
  placeholder: string
  required: boolean
}

export interface CloudProviderServerType {
  cpu: number
  ram: number
  disk: number
  name: string
}

export interface CloudProviderInfo {
  type: CloudProviderType
  name: string
  implemented: boolean
  description: string
  documentationLink: string
  credentialFields: CloudProviderCredentialField[]
}

export interface SourceCodeRepository {
  id: string
  repository: string
  visibility: 'public' | 'private'
  lastUpdatedAt: string
}

export interface SourceCodeProvider {
  id: string
  name: string
  avatar: string
  provider: 'github' | 'gitlab' | 'bitbucket'
  type: 'user' | 'organization'
}


export interface Application {
  id: string
  name: string
  type: 'mysql' | 'postgres' | 'git' | 'docker_image'
  project_id: string
  created_at: string
  updated_at: string
}

export interface Deployment {
  id: string
}

export type PageProps<T extends Record<string, unknown> = Record<string, unknown>> = T & {
  profile: UserProfile
  project: Project
  projects: Project[]
  workspace: UserProfile['workspaces'][number]
  cloudProviderRegions: Record<CloudProviderType, CloudProviderRegionsByContinent>
}
