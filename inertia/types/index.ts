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

export interface Project {
  id: string
  name: string
  slug: string
  workspace_id: string
  created_at: string
  updated_at: string
  environments?: Environment[]
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
}

export type CloudProviderRegionsByContinent = Record<string, CloudProviderRegion[]>

export type ClusterStatus = 'Healthy' | 'Unhealthy' | 'Pending'

export type ClusterNodeType = 'worker' | 'storage'

export interface Cluster {
  id: string
  name: string
  slug: string
  status: ClusterStatus
  workspace_id: string
  cloud_provider_id: string
  region: string
  shared_storage_worker_nodes: boolean
  vault_ssh_key_path: string | null
  created_at: string
  updated_at: string
  nodes?: ClusterNode[]
  cloud_provider?: CloudProvider
}

export interface ClusterNode {
  id: string
  cluster_id: string
  node_id: string
  type: ClusterNodeType
  status: ClusterStatus
  public_ip: string | null
  private_ip: string | null
  public_ipv6: string | null
  private_ipv6: string | null
  server_type: string | null
  is_master: boolean
  cpu_cores: number
  ram_gb: number
  disk_gb: number
  os: string
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

export type PageProps<T extends Record<string, unknown> = Record<string, unknown>> = T & {
  profile: UserProfile
  workspace: UserProfile['workspaces'][number]
}
