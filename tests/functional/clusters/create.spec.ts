import { test } from '@japa/runner'
import Cluster from '#models/cluster'
import ClusterNode from '#models/cluster_node'
import ClusterSshKey from '#models/cluster_ssh_key'
import { createUserWithWorkspace, createTestCloudProvider, generateTestClusterData } from '#tests/helpers/test_helpers'

test.group('Clusters create', () => {
  test('successfully creates a cluster with all required components', async ({ client, assert, route }) => {
    const { user, workspaceId, workspaceSlug } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId)
    const clusterData = generateTestClusterData(cloudProvider.id)

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
    assert.equal(cluster!.controlPlanesVolumeSize, 0)
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
