import { test } from '@japa/runner'
import Cluster from '#models/cluster'

test.group('Cluster Model', () => {
  test('region returns US flag when cloudProvider is not loaded', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'us-east-1'

    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'US East (N. Virginia)')
  })

  test('region returns correct flag for Hetzner location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'fsn1'

    // @ts-ignore
    cluster.cloudProvider = { type: 'hetzner' }

    assert.equal(cluster.region.flag, '/flags/de.svg')
    assert.equal(cluster.region.name, 'Falkenstein, Germany')
  })

  test('region returns correct flag for Digital Ocean location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'nyc1'

    // @ts-ignore
    cluster.cloudProvider = { type: 'digital_ocean' }

    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'New York 1')
  })

  test('region returns correct flag for AWS location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'ca-central-1'

    // @ts-ignore
    cluster.cloudProvider = { type: 'aws' }

    assert.equal(cluster.region.flag, '/flags/ca.svg')
    assert.equal(cluster.region.name, 'Canada (Central)')
  })

  test('region returns US flag for unknown location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'unknown-location'

    // @ts-ignore
    cluster.cloudProvider = { type: 'hetzner' }

    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })

  test('region returns US flag for unsupported provider type', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'some-location'

    // @ts-ignore
    cluster.cloudProvider = { type: 'unsupported_provider' }

    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })

  test('region returns correct flag for Google Cloud location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'europe-west1'

    // @ts-ignore
    cluster.cloudProvider = { type: 'google_cloud' }

    assert.equal(cluster.region.flag, '/flags/be.svg')
    assert.equal(cluster.region.name, 'Belgium (europe-west1)')
  })

  test('region returns correct flag for Singapore locations', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'sgp1'

    // @ts-ignore
    cluster.cloudProvider = { type: 'digital_ocean' }

    assert.equal(cluster.region.flag, '/flags/sg.svg')
    assert.equal(cluster.region.name, 'Singapore 1')
  })

  test('region returns correct flag for BYOC cluster', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'eu-central-1'
    cluster.subdomainIdentifier = 'byoc-1234567890'

    // @ts-ignore
    cluster.cloudProvider = undefined

    assert.equal(cluster.region.flag, '/flags/de.svg')
    assert.equal(cluster.region.name, 'Frankfurt')
  })

  test('region returns correct flag for BYOC cluster in Singapore', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'ap-southeast-1'
    cluster.subdomainIdentifier = 'byoc-9876543210'

    // @ts-ignore
    cluster.cloudProvider = undefined

    assert.equal(cluster.region.flag, '/flags/sg.svg')
    assert.equal(cluster.region.name, 'Singapore')
  })

  test('region returns US flag for BYOC cluster with unknown location', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'unknown-byoc-region'
    cluster.subdomainIdentifier = 'byoc-1111111111'

    // @ts-ignore
    cluster.cloudProvider = undefined

    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })

  test('region returns US flag for non-BYOC cluster without cloudProvider', async ({ assert }) => {
    const cluster = new Cluster()
    cluster.location = 'some-location'
    cluster.subdomainIdentifier = 'regular-cluster-123'

    // @ts-ignore
    cluster.cloudProvider = undefined

    assert.equal(cluster.region.flag, '/flags/us.svg')
    assert.equal(cluster.region.name, 'Unknown')
  })
})
