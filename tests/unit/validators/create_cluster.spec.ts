import { test } from '@japa/runner'
import { createClusterValidator } from '#validators/create_cluster'
import CloudProvider from '#models/cloud_provider'
import { randomBytes } from 'node:crypto'

test.group('Create cluster validator', () => {
  test('validates valid cluster data successfully', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const validData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: cloudProvider.id,
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const [error, data] = await createClusterValidator.tryValidate(validData)

    assert.isNull(error)
    assert.isNotNull(data)
    if (data) {
      assert.equal(data.subdomain_identifier, 'test.kibaship.com')
      assert.equal(data.cloud_provider_id, cloudProvider.id)
      assert.equal(data.region, 'eu-central')
      assert.equal(data.control_plane_nodes_count, 3)
      assert.equal(data.worker_nodes_count, 5)
      assert.equal(data.server_type, 'cx11')
      assert.equal(data.control_planes_volume_size, 50)
      assert.equal(data.workers_volume_size, 100)
    }
  })

  test('fails validation with invalid subdomain format', async ({ assert }) => {
    const invalidData = {
      subdomain_identifier: 'invalid_domain_format',
      cloud_provider_id: '550e8400-e29b-41d4-a716-446655440000',
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })

  test('fails validation with invalid control plane nodes count', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const invalidData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: cloudProvider.id,
      region: 'eu-central',
      control_plane_nodes_count: 2,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })

  test('fails validation with insufficient worker nodes', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const invalidData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: cloudProvider.id,
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 1,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })

  test('fails validation with non-existent cloud provider', async ({ assert }) => {
    const invalidData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: '550e8400-e29b-41d4-a716-446655440000',
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })

  test('fails validation with invalid UUID format for cloud provider', async ({ assert }) => {
    const invalidData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: 'not-a-uuid',
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 50,
      workers_volume_size: 100,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })

  test('validates all allowed control plane node counts', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const allowedCounts = [1, 3, 5]

    for (const count of allowedCounts) {
      const validData = {
        subdomain_identifier: `test-${count}.kibaship.com`,
        cloud_provider_id: cloudProvider.id,
        region: 'eu-central',
        control_plane_nodes_count: count,
        worker_nodes_count: 5,
        server_type: 'cx11',
        control_planes_volume_size: 50,
        workers_volume_size: 100,
      }

      const [error, data] = await createClusterValidator.tryValidate(validData)

      assert.isNull(error, `Control plane count ${count} should be valid`)
      assert.isNotNull(data, `Control plane count ${count} should return data`)
    }
  })

  test('validates various valid domain formats', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const validDomains = [
      'simple.com',
      'sub.domain.com',
      'api.staging.example.org',
      'my-app.domain.co.uk',
      'test123.example.xyz',
      'a.b.c.d.example.com',
    ]

    for (const domain of validDomains) {
      const validData = {
        subdomain_identifier: domain,
        cloud_provider_id: cloudProvider.id,
        region: 'eu-central',
        control_plane_nodes_count: 3,
        worker_nodes_count: 5,
        server_type: 'cx11',
        control_planes_volume_size: 50,
        workers_volume_size: 100,
      }

      const [error, data] = await createClusterValidator.tryValidate(validData)

      assert.isNull(error, `Domain ${domain} should be valid`)
      assert.isNotNull(data, `Domain ${domain} should return data`)
    }
  })

  test('rejects invalid domain formats', async ({ assert }) => {
    const invalidDomains = [
      'invalid_domain',
      '-invalid.com',
      'invalid-.com',
      'invalid..com',
      'invalid.c',
      'just-text',
      'domain.',
      '.domain.com',
    ]

    for (const domain of invalidDomains) {
      const invalidData = {
        subdomain_identifier: domain,
        cloud_provider_id: '550e8400-e29b-41d4-a716-446655440000',
        region: 'eu-central',
        control_plane_nodes_count: 3,
        worker_nodes_count: 5,
        server_type: 'cx11',
        control_planes_volume_size: 50,
        workers_volume_size: 100,
      }

      const [error, data] = await createClusterValidator.tryValidate(invalidData)

      assert.isNotNull(error, `Domain ${domain} should be invalid`)
      assert.isNull(data, `Domain ${domain} should not return data`)
      if (error) {
        assert.property(error, 'messages')
      }
    }
  })

  test('fails validation with missing required fields', async ({ assert }) => {
    const incompleteData = {
      subdomain_identifier: 'test.kibaship.com',
    }

    const [error, data] = await createClusterValidator.tryValidate(incompleteData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
      assert.isAtLeast(error.messages.length, 7)
    }
  })

  test('fails validation with volume size too small', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const invalidData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: cloudProvider.id,
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 5,
      workers_volume_size: 5,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })

  test('fails validation with volume size too large', async ({ assert }) => {
    const cloudProvider = await CloudProvider.create({
      name: 'Test Provider',
      type: 'hetzner',
      workspaceId: `workspace_${randomBytes(8).toString('hex')}`,
      credentials: { token: 'test-token' },
    })

    const invalidData = {
      subdomain_identifier: 'test.kibaship.com',
      cloud_provider_id: cloudProvider.id,
      region: 'eu-central',
      control_plane_nodes_count: 3,
      worker_nodes_count: 5,
      server_type: 'cx11',
      control_planes_volume_size: 600,
      workers_volume_size: 600,
    }

    const [error, data] = await createClusterValidator.tryValidate(invalidData)

    assert.isNotNull(error)
    assert.isNull(data)
    if (error) {
      assert.property(error, 'messages')
      assert.isArray(error.messages)
    }
  })
})
