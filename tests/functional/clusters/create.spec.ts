import { test } from '@japa/runner'
import User from '#models/user'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import ClusterNode from '#models/cluster_node'
import ClusterSshKey from '#models/cluster_ssh_key'
import { randomBytes } from 'node:crypto'
import redis from '@adonisjs/redis/services/main'

async function createUserWithWorkspace() {
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

test.group('Clusters create', () => {
  test('successfully creates a cluster with all required components', async ({ client, assert, route }) => {
    const { user, workspaceId, workspaceSlug } = await createUserWithWorkspace()

    const cloudProvider = await CloudProvider.create({
      name: 'Test Hetzner Provider',
      type: 'hetzner',
      workspaceId: workspaceId,
      credentials: { token: 'test-token' },
    })

    const clusterData = {
      subdomain_identifier: 'test-cluster.kibaship.com',
      cloud_provider_id: cloudProvider.id,
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 3,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const response = await client
      .post(route('clusters.store', { workspace: workspaceSlug }))
      .form(clusterData)
      .redirects(0)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    response.assertStatus(302)
    assert.include(response.text(), route('clusters.index', { workspace: workspaceSlug }))

    const cluster = await Cluster.findBy('subdomainIdentifier', clusterData.subdomain_identifier)
    assert.isNotNull(cluster)
    assert.equal(cluster!.location, clusterData.region)
    assert.equal(cluster!.cloudProviderId, cloudProvider.id)
    assert.equal(cluster!.workspaceId, workspaceId)
    assert.equal(cluster!.status, 'provisioning')
    assert.equal(cluster!.kind, 'all_purpose')
    assert.equal(cluster!.controlPlanesVolumeSize, clusterData.control_planes_volume_size)
    assert.equal(cluster!.workersVolumeSize, clusterData.workers_volume_size)

    const sshKey = await ClusterSshKey.findBy('clusterId', cluster!.id)
    assert.isNotNull(sshKey)
    assert.include(sshKey!.publicKey, 'ssh-ed25519')
    assert.include(sshKey!.privateKey, 'BEGIN OPENSSH PRIVATE KEY')

    const controlPlaneNodes = await ClusterNode.query()
      .where('clusterId', cluster!.id)
      .where('type', 'master')
    assert.lengthOf(controlPlaneNodes, 3)
    controlPlaneNodes.forEach(node => {
      assert.equal(node.status, 'provisioning')
      assert.equal(node.type, 'master')
    })

    const workerNodes = await ClusterNode.query()
      .where('clusterId', cluster!.id)
      .where('type', 'worker')
    assert.lengthOf(workerNodes, 3)
    workerNodes.forEach(node => {
      assert.equal(node.status, 'provisioning')
      assert.equal(node.type, 'worker')
    })
  })
})
