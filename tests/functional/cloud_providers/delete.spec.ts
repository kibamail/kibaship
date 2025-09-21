import { test } from '@japa/runner'
import CloudProvider from '#models/cloud_provider'
import { createUserWithWorkspace, createTestCloudProvider, createTestCluster } from '#tests/helpers/test_helpers'
import { DateTime } from 'luxon'

test.group('Cloud Providers delete', () => {
  test('successfully soft deletes a cloud provider', async ({ client, assert, route }) => {
    const { user, workspaceId, workspaceSlug } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId, 'test-token-123')

    assert.isTrue(cloudProvider.deletedAt === null || cloudProvider.deletedAt === undefined)
    assert.deepEqual(cloudProvider.credentials, { token: 'test-token-123' })

    const response = await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .redirects(0)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    response.assertStatus(302)
    assert.include(response.text(), route('workspace.cloud.providers', { workspace: workspaceSlug }))

    await cloudProvider.refresh()
    assert.isNotNull(cloudProvider.deletedAt)
    assert.isTrue(cloudProvider.deletedAt instanceof DateTime)
    assert.deepEqual(cloudProvider.credentials, {})
  })

  test('soft deleted cloud provider is excluded from queries', async ({ client, assert }) => {
    const { user, workspaceId } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId, 'test-token-456')

    await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    const activeProviders = await CloudProvider.query()
      .where('workspace_id', workspaceId)
      .whereNull('deleted_at')

    assert.lengthOf(activeProviders, 0)

    const allProviders = await CloudProvider.query()
      .where('workspace_id', workspaceId)

    assert.lengthOf(allProviders, 1)
    assert.isNotNull(allProviders[0].deletedAt)
  })

  test('deleting cloud provider with associated clusters preserves clusters', async ({ client, assert }) => {
    const { user, workspaceId } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId, 'test-token-789')
    const cluster = await createTestCluster(workspaceId, cloudProvider.id)

    assert.equal(cluster.cloudProviderId, cloudProvider.id)

    await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    await cloudProvider.refresh()
    assert.isNotNull(cloudProvider.deletedAt)

    await cluster.refresh()
    assert.equal(cluster.cloudProviderId, cloudProvider.id)
    assert.isNull(cluster.deletedAt)
  })

  test('returns 404 for non-existent cloud provider', async ({ client }) => {
    const { user } = await createUserWithWorkspace()
    const nonExistentId = 'non-existent-uuid-12345678-1234-1234-1234-123456789012'

    const response = await client
      .delete(`/connections/cloud-providers/${nonExistentId}`)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    response.assertStatus(404)
  })

  test('requires authentication', async ({ client, assert }) => {
    const { workspaceId } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId)

    const response = await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .withCsrfToken()
      .redirects(0)

    response.assertStatus(302)
    assert.equal(response.headers().location, '/')
  })

  test('prevents deletion of cloud provider from different workspace', async ({ client, assert }) => {
    const { workspaceId: workspace1Id } = await createUserWithWorkspace()
    const { user: user2 } = await createUserWithWorkspace()

    const cloudProvider = await createTestCloudProvider(workspace1Id)

    const response = await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user2)

    response.assertStatus(404)

    await cloudProvider.refresh()
    assert.isNull(cloudProvider.deletedAt)
  })

  test('handles multiple deletions of same cloud provider gracefully', async ({ client, assert }) => {
    const { user, workspaceId } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId)

    const response1 = await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .redirects(0)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    response1.assertStatus(302)

    await cloudProvider.refresh()
    const firstDeletedAt = cloudProvider.deletedAt
    assert.isNotNull(firstDeletedAt)

    const response2 = await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .redirects(0)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    response2.assertStatus(302)

    await cloudProvider.refresh()
    const secondDeletedAt = cloudProvider.deletedAt
    assert.isNotNull(secondDeletedAt)

    const timeDiff = Math.abs(secondDeletedAt!.toMillis() - firstDeletedAt!.toMillis())
    assert.isBelow(timeDiff, 1000)
  })

  test('clears sensitive credentials completely', async ({ client, assert }) => {
    const { user, workspaceId } = await createUserWithWorkspace()

    const cloudProvider = await CloudProvider.create({
      name: 'Test AWS Provider',
      type: 'aws',
      workspaceId: workspaceId,
      credentials: {
        access_key_id: 'AKIAIOSFODNN7EXAMPLE',
        secret_access_key: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
        token: 'some-token'
      },
    })

    assert.equal(cloudProvider.credentials.access_key_id, 'AKIAIOSFODNN7EXAMPLE')
    assert.equal(cloudProvider.credentials.secret_access_key, 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY')
    assert.equal(cloudProvider.credentials.token, 'some-token')

    await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    await cloudProvider.refresh()
    assert.deepEqual(cloudProvider.credentials, {})
    assert.isUndefined(cloudProvider.credentials.access_key_id)
    assert.isUndefined(cloudProvider.credentials.secret_access_key)
    assert.isUndefined(cloudProvider.credentials.token)
  })

  test('preserves cloud provider metadata after soft delete', async ({ client, assert }) => {
    const { user, workspaceId } = await createUserWithWorkspace()
    const cloudProvider = await createTestCloudProvider(workspaceId)

    const originalName = cloudProvider.name
    const originalType = cloudProvider.type
    const originalWorkspaceId = cloudProvider.workspaceId
    const originalCreatedAt = cloudProvider.createdAt

    await client
      .delete(`/connections/cloud-providers/${cloudProvider.id}`)
      .withCsrfToken()
      .withGuard('web')
      .loginAs(user)

    await cloudProvider.refresh()
    assert.equal(cloudProvider.name, originalName)
    assert.equal(cloudProvider.type, originalType)
    assert.equal(cloudProvider.workspaceId, originalWorkspaceId)

    const timeDiff = Math.abs(cloudProvider.createdAt.toMillis() - originalCreatedAt.toMillis())
    assert.isBelow(timeDiff, 1000)

    assert.isNotNull(cloudProvider.deletedAt)
    assert.deepEqual(cloudProvider.credentials, {})
  })
})
