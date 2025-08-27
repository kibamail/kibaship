import User from '#models/user'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import { randomBytes } from 'node:crypto'
import redis from '@adonisjs/redis/services/main'
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
 * Creates a test user with a mock workspace profile stored in Redis
 */
export async function createUserWithWorkspace(): Promise<UserWithWorkspace> {
  const user = await User.create({
    email: `test_${randomBytes(4).toString('hex')}@example.com`,
    oauthId: `oauth_${randomBytes(8).toString('hex')}`,
  })

  const workspaceId = `workspace_${randomBytes(8).toString('hex')}`
  const workspaceSlug = `test-workspace-${randomBytes(4).toString('hex')}`
  const mockProfile = {
    id: user.oauthId,
    email: user.email,
    workspaces: [
      {
        id: workspaceId,
        slug: workspaceSlug,
        name: 'Test Workspace',
      },
    ],
  }

  await redis.set(`users:${user.id}`, JSON.stringify(mockProfile))

  return { user, workspaceId, workspaceSlug }
}

/**
 * Creates a test cloud provider for Hetzner
 */
export async function createTestCloudProvider(workspaceId: string, token: string = 'test-token'): Promise<CloudProvider> {
  return CloudProvider.create({
    name: 'Test Hetzner Provider',
    type: 'hetzner',
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
        control_planes_volume_size: 50,
        workers_volume_size: 100,
      },
      workspaceId,
      trx
    )

    cluster.controlPlaneEndpoint = `https://kube.${cluster.subdomainIdentifier}`

    await cluster.save()
    await trx.commit()

    await cluster.load('cloudProvider')
    await cluster.load('nodes')
    await cluster.load('sshKeys')

    if (cluster.nodes && cluster.nodes.length > 0) {
      for (const node of cluster.nodes) {
        if (!node.slug) {
          throw new Error(`Node ${node.id} is missing slug`)
        }
      }
    }

    if (!cluster.sshKeys || cluster.sshKeys.length === 0) {
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
