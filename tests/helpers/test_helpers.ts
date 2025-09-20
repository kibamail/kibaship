import User from '#models/user'
import Workspace from '#models/workspace'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import ClusterLoadBalancer from '#models/cluster_load_balancer'

import { randomBytes } from 'node:crypto'
import db from '@adonisjs/lucid/services/db'

export interface UserWithWorkspace {
  user: User
  workspaceId: string
  workspaceSlug: string
}

export interface TestClusterData {
  subdomain_identifier: string
  cloud_provider_id: string
  region: string
  control_plane_nodes_count: number
  worker_nodes_count: number
  server_type: string
  control_planes_volume_size: number
  workers_volume_size: number
}

/**
 * Creates a test user with password authentication and a real workspace
 */
export async function createUserWithWorkspace(): Promise<UserWithWorkspace> {
  const email = `test_${randomBytes(4).toString('hex')}@example.com`
  const password = 'testpassword123'

  const { user, workspace } = await db.transaction(async (trx) => {
    const user = new User()
    user.email = email
    user.password = password
    user.useTransaction(trx)
    await user.save()

    const [username] = email.split('@')
    const workspaceName = `${username}'s Workspace`
    const slug = username.toLowerCase().replace(/[^a-z0-9]/g, '-')

    const workspace = new Workspace()
    workspace.name = workspaceName
    workspace.slug = slug
    workspace.userId = user.id
    workspace.useTransaction(trx)
    await workspace.save()

    return { user, workspace }
  })

  return {
    user,
    workspaceId: workspace.id,
    workspaceSlug: workspace.slug
  }
}

/**
 * Creates a test user with password authentication (legacy OAuth support removed)
 */
export async function createTestUser(email?: string, password?: string): Promise<User> {
  const testEmail = email || `test_${randomBytes(4).toString('hex')}@example.com`
  const testPassword = password || 'testpassword123'

  return User.create({
    email: testEmail,
    password: testPassword,
  })
}

/**
 * Creates a registered user with workspace for testing protected routes
 */
export async function createRegisteredUser(email?: string, password?: string): Promise<{ user: User; workspace: Workspace }> {
  const testEmail = email || `registered_${randomBytes(4).toString('hex')}@example.com`
  const testPassword = password || 'testpassword123'

  return db.transaction(async (trx) => {
    const user = new User()
    user.email = testEmail
    user.password = testPassword
    user.useTransaction(trx)
    await user.save()

    const [username] = testEmail.split('@')
    const workspaceName = `${username}'s Workspace`
    const slug = `${username.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${randomBytes(2).toString('hex')}`

    const workspace = new Workspace()
    workspace.name = workspaceName
    workspace.slug = slug
    workspace.userId = user.id
    workspace.useTransaction(trx)
    await workspace.save()

    return { user, workspace }
  })
}

/**
 * Creates a test cloud provider for Hetzner
 */
export async function createTestCloudProvider(workspaceId: string, token: string = 'test-token'): Promise<CloudProvider> {
  return CloudProvider.create({
    name: 'Test digital ocean provider',
    type: 'digital_ocean',
    workspaceId: workspaceId,
    credentials: { token },
  })
}

/**
 * Creates a test cluster with all required infrastructure components
 */
export async function createTestCluster(workspaceId: string, cloudProviderId: string): Promise<Cluster> {
  const trx = await db.transaction()

  try {
    const cluster = await Cluster.createWithInfrastructure(
      {
        subdomain_identifier: `test-cluster-${randomBytes(4).toString('hex')}.kibaship.com`,
        cloud_provider_id: cloudProviderId,
        region: 'nbg1',
        control_plane_nodes_count: 3,
        worker_nodes_count: 3,
        server_type: 'cx11',
        workers_volume_size: 100,
      },
      workspaceId,
      trx
    )

    cluster.controlPlaneEndpoint = `https://kube.${cluster.subdomainIdentifier}`
    const lb = new ClusterLoadBalancer()
    lb.clusterId = cluster.id
    lb.type = 'cluster'
    lb.publicIpv4Address = null
    lb.privateIpv4Address = null


    lb.useTransaction(trx)
    await lb.save()


    await cluster.save()
    await trx.commit()

    await cluster.load('cloudProvider')
    await cluster.load('nodes', (query) => { query.preload('storages') })
    await cluster.load('sshKey')
    await cluster.load('loadBalancers')


    if (cluster.nodes && cluster.nodes.length > 0) {
      for (const node of cluster.nodes) {
        if (!node.slug) {
          throw new Error(`Node ${node.id} is missing slug`)
        }
      }
    }

    if (!cluster.sshKey) {
      throw new Error('Cluster is missing SSH keys')
    }

    return cluster
  } catch (error) {
    await trx.rollback()
    throw error
  }
}

/**
 * Creates a complete test setup with user, workspace, cloud provider, and cluster
 */
export async function createCompleteTestSetup(cloudProviderToken: string = 'test-token') {
  const { user, workspaceId, workspaceSlug } = await createUserWithWorkspace()
  const cloudProvider = await createTestCloudProvider(workspaceId, cloudProviderToken)
  const cluster = await createTestCluster(workspaceId, cloudProvider.id)

  return {
    user,
    workspaceId,
    workspaceSlug,
    cloudProvider,
    cluster
  }
}

/**
 * Generates test cluster data for API requests
 */
export function generateTestClusterData(cloudProviderId: string): TestClusterData {
  return {
    subdomain_identifier: `test-cluster-${randomBytes(4).toString('hex')}.kibaship.com`,
    cloud_provider_id: cloudProviderId,
    region: 'eu-central',
    control_plane_nodes_count: 3,
    worker_nodes_count: 3,
    server_type: 'cx11',
    control_planes_volume_size: 50,
    workers_volume_size: 100,
  }
}

/**
 * Generates a random workspace ID for testing
 */
export function generateWorkspaceId(): string {
  return `workspace_${randomBytes(8).toString('hex')}`
}

/**
 * Generates a random OAuth ID for testing
 */
export function generateOauthId(): string {
  return `oauth_${randomBytes(8).toString('hex')}`
}

/**
 * Generates a random test email
 */
export function generateTestEmail(): string {
  return `test_${randomBytes(4).toString('hex')}@example.com`
}
