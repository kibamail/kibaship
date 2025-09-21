import { test } from '@japa/runner'
import Cluster from '#models/cluster'

test.group('Cluster Model', () => {
  test('region returns US flag when cloudProvider is not loaded', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'us-east-1'

    // cloudProvider is not loaded (undefined)
    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'US East (N. Virginia)')
  })

  test('region returns correct flag for Hetzner location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'fsn1'

    // Mock cloudProvider by setting the property directly
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'hetzner' }

    // fsn1 is Falkenstein, Germany - should return German flag
    assert.equal(cluster.region.flag, '/flags/de.svg')
    assert.equal(cluster.region.name, 'Falkenstein, Germany')
  })

  test('region returns correct flag for Digital Ocean location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'nyc1'

    // Mock cloudProvider by setting the property directly
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'digital_ocean' }

    // nyc1 is New York - should return US flag
    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'New York 1')
  })

  test('region returns correct flag for AWS location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'ca-central-1'

    // Mock cloudProvider by setting the property directly
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'aws' }

    // ca-central-1 is Canada Central - should return Canadian flag
    assert.equal(cluster.region.flag, '/flags/ca.svg')
    assert.equal(cluster.region.name, 'Canada (Central)')
  })

  test('region returns US flag for unknown location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'unknown-location'

    // Mock cloudProvider by setting the property directly
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'hetzner' }

    // Unknown location should default to US flag
    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })

  test('region returns US flag for unsupported provider type', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'some-location'

    // Mock cloudProvider with unsupported type
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'unsupported_provider' }

    // Unsupported provider should default to US flag
    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })

  test('region returns correct flag for Google Cloud location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'europe-west1'

    // Mock cloudProvider by setting the property directly
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'google_cloud' }

    // europe-west1 is Belgium - should return Belgian flag
    assert.equal(cluster.region.flag, '/flags/be.svg')
    assert.equal(cluster.region.name, 'Belgium (europe-west1)')
  })

  test('region returns correct flag for Singapore locations', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'sgp1'

    // Mock cloudProvider by setting the property directly
    // @ts-ignore - Mocking the relationship for testing
    cluster.cloudProvider = { type: 'digital_ocean' }

    // sgp1 is Singapore - should return Singapore flag
    assert.equal(cluster.region.flag, '/flags/sg.svg')
    assert.equal(cluster.region.name, 'Singapore 1')
  })

  test('region returns correct flag for BYOC cluster', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'eu-central-1'
    cluster.subdomainIdentifier = 'byoc-1234567890'

    // No cloudProvider for BYOC clusters
    // @ts-ignore - Setting to undefined for testing BYOC clusters
    cluster.cloudProvider = undefined

    // eu-central-1 is Frankfurt, Germany - should return German flag
    assert.equal(cluster.region.flag, '/flags/de.svg')
    assert.equal(cluster.region.name, 'Frankfurt')
  })

  test('region returns correct flag for BYOC cluster in Singapore', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'ap-southeast-1'
    cluster.subdomainIdentifier = 'byoc-9876543210'

    // No cloudProvider for BYOC clusters
    // @ts-ignore - Setting to undefined for testing BYOC clusters
    cluster.cloudProvider = undefined

    // ap-southeast-1 is Singapore - should return Singapore flag
    assert.equal(cluster.region.flag, '/flags/sg.svg')
    assert.equal(cluster.region.name, 'Singapore')
  })

  test('region returns US flag for BYOC cluster with unknown location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'unknown-byoc-region'
    cluster.subdomainIdentifier = 'byoc-1111111111'

    // No cloudProvider for BYOC clusters
    // @ts-ignore - Setting to undefined for testing BYOC clusters
    cluster.cloudProvider = undefined

    // Unknown BYOC location should default to US flag
    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })

  test('region returns US flag for non-BYOC cluster without cloudProvider', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'some-location'
    cluster.subdomainIdentifier = 'regular-cluster-123'

    // No cloudProvider and not a BYOC cluster
    // @ts-ignore - Setting to undefined for testing non-BYOC clusters without provider
    cluster.cloudProvider = undefined

    // Should default to US flag
    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })
})
