import { test } from '@japa/runner'
import sinon from 'sinon'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionNetworkJob from '#jobs/clusters/provision_network_job'
import ProvisionSshKeysJob from '#jobs/clusters/provision_ssh_keys_job'
import { TerraformExecutionOptions } from '#services/terraform/terraform_executor'
import { type Subprocess } from 'execa'
import {
  createFullyPopulatedDigitalOceanCluster,
  setupTerraformExecutorMock,
  restoreTerraformExecutor,
  MockTerraformExecutor,
  validateTerraformPlan,
  assertTerraformPlanValid,
  HETZNER_NETWORK_OUTPUT,
  DIGITAL_OCEAN_NETWORK_OUTPUT
} from './helpers/cluster_provisioning_helpers.js'

test.group('ProvisionNetworkJob - Hetzner', (group) => {
  group.setup(setupTerraformExecutorMock)
  group.teardown(restoreTerraformExecutor)

  test('should provision network for hetzner and dispatch ssh keys job', async ({ assert }) => {
    const { cluster, cloudProvider } = await createFullyPopulatedDigitalOceanCluster()

    // Convert to Hetzner cluster with valid 64-character token
    cloudProvider.type = 'hetzner'
    cloudProvider.credentials = {
      token: '1234567890123456789012345678901234567890123456789012345678901234'
    }
    await cloudProvider.save()

    assert.isUndefined(cluster.networkingStartedAt)
    assert.isUndefined(cluster.networkingCompletedAt)
    assert.isUndefined(cluster.networkingErrorAt)
    assert.isUndefined(cluster.providerNetworkId)
    assert.isUndefined(cluster.providerSubnetId)
    assert.equal(cluster.networkIpRange, '10.0.0.0/16') // Set by factory
    assert.equal(cluster.subnetIpRange, '10.0.1.0/24') // Set by factory

    class TestMockExecutor extends MockTerraformExecutor {
      async apply(options?: TerraformExecutionOptions) {
        this.setMockOutput(HETZNER_NETWORK_OUTPUT)
        return super.apply(options)
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => TestMockExecutor as any)

    const dispatchSpy = sinon.spy(queue, 'dispatch')

    const job = new ProvisionNetworkJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()
    await cluster.load('cloudProvider')

    assert.isNotNull(cluster.networkingStartedAt)
    assert.isNotNull(cluster.networkingCompletedAt)
    assert.isNull(cluster.networkingErrorAt)
    assert.equal(cluster.providerNetworkId, HETZNER_NETWORK_OUTPUT.network_id.value)
    assert.equal(cluster.providerSubnetId, HETZNER_NETWORK_OUTPUT.subnet_id.value)
    assert.equal(cluster.networkIpRange, HETZNER_NETWORK_OUTPUT.network_ip_range.value)
    assert.equal(cluster.subnetIpRange, HETZNER_NETWORK_OUTPUT.subnet_ip_range.value)

    assert.isTrue(dispatchSpy.calledOnce)
    assert.isTrue(dispatchSpy.calledWith(ProvisionSshKeysJob, { clusterId: cluster.id }))

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    dispatchSpy.restore()
    app.default.container.bind('terraform.executor', () => MockTerraformExecutor as any)
  })

  test('should execute terraform plan and generate valid output for hetzner', async ({ assert }) => {
    const { cluster, cloudProvider } = await createFullyPopulatedDigitalOceanCluster()

    // Convert to Hetzner cluster with valid 64-character token
    cloudProvider.type = 'hetzner'
    cloudProvider.credentials = {
      token: '1234567890123456789012345678901234567890123456789012345678901234'
    }
    await cloudProvider.save()

    class TestMockExecutor extends MockTerraformExecutor {
      async apply(options?: TerraformExecutionOptions) {
        this.setMockOutput(HETZNER_NETWORK_OUTPUT)
        return super.apply(options)
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => TestMockExecutor as any)

    const job = new ProvisionNetworkJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()
    await cluster.load('cloudProvider')

    assert.isNotNull(cluster.networkingStartedAt)
    assert.isNotNull(cluster.networkingCompletedAt)
    assert.isNull(cluster.networkingErrorAt)
    assert.equal(cluster.providerNetworkId, HETZNER_NETWORK_OUTPUT.network_id.value)

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    app.default.container.bind('terraform.executor', () => MockTerraformExecutor as any)
  })

  test('should handle errors gracefully and set error timestamp for hetzner', async ({ assert }) => {
    const { cluster, cloudProvider } = await createFullyPopulatedDigitalOceanCluster()

    // Convert to Hetzner cluster with valid 64-character token
    cloudProvider.type = 'hetzner'
    cloudProvider.credentials = {
      token: '1234567890123456789012345678901234567890123456789012345678901234'
    }
    await cloudProvider.save()

    class ErrorMockExecutor extends MockTerraformExecutor {
      async apply(_options?: TerraformExecutionOptions): Promise<Subprocess> {
        return Promise.reject(new Error('Terraform apply failed'))
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => ErrorMockExecutor as any)

    const job = new ProvisionNetworkJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()
    await cluster.load('cloudProvider')

    assert.isNotNull(cluster.networkingStartedAt)
    assert.isNull(cluster.networkingCompletedAt)
    assert.isNotNull(cluster.networkingErrorAt)
    assert.equal(cluster.status, 'unhealthy')
    assert.isNull(cluster.providerNetworkId)

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    app.default.container.bind('terraform.executor', () => MockTerraformExecutor as any)
  })
})

test.group('ProvisionNetworkJob - DigitalOcean', (group) => {
  group.setup(setupTerraformExecutorMock)
  group.teardown(restoreTerraformExecutor)

  test('should provision network for digital ocean and dispatch ssh keys job', async ({ assert }) => {
    const { cluster } = await createFullyPopulatedDigitalOceanCluster()

    assert.isUndefined(cluster.networkingStartedAt)
    assert.isUndefined(cluster.networkingCompletedAt)
    assert.isUndefined(cluster.networkingErrorAt)
    assert.isUndefined(cluster.providerNetworkId)
    assert.isUndefined(cluster.providerSubnetId)
    assert.equal(cluster.networkIpRange, '10.0.0.0/16') // Set by factory
    assert.equal(cluster.subnetIpRange, '10.0.1.0/24') // Set by factory

    class TestMockExecutor extends MockTerraformExecutor {
      async apply(options?: TerraformExecutionOptions) {
        this.setMockOutput(DIGITAL_OCEAN_NETWORK_OUTPUT)
        return super.apply(options)
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => TestMockExecutor as any)

    const dispatchSpy = sinon.spy(queue, 'dispatch')

    const job = new ProvisionNetworkJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.isNotNull(cluster.networkingStartedAt)
    assert.isNotNull(cluster.networkingCompletedAt)
    assert.isNull(cluster.networkingErrorAt)
    assert.equal(cluster.providerNetworkId, DIGITAL_OCEAN_NETWORK_OUTPUT.network_id.value)
    assert.equal(cluster.providerSubnetId, DIGITAL_OCEAN_NETWORK_OUTPUT.network_id.value) // DO uses same ID
    assert.equal(cluster.networkIpRange, DIGITAL_OCEAN_NETWORK_OUTPUT.network_ip_range.value)
    assert.equal(cluster.subnetIpRange, DIGITAL_OCEAN_NETWORK_OUTPUT.network_ip_range.value) // DO uses same range

    assert.isTrue(dispatchSpy.calledOnce)
    assert.isTrue(dispatchSpy.calledWith(ProvisionSshKeysJob, { clusterId: cluster.id }))

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    dispatchSpy.restore()
    app.default.container.bind('terraform.executor', () => MockTerraformExecutor as any)
  })

  test('should execute terraform plan and generate valid output for digital ocean', async ({ assert }) => {
    const { cluster } = await createFullyPopulatedDigitalOceanCluster()

    class TestMockExecutor extends MockTerraformExecutor {
      async apply(options?: TerraformExecutionOptions) {
        this.setMockOutput(DIGITAL_OCEAN_NETWORK_OUTPUT)
        return super.apply(options)
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => TestMockExecutor as any)

    const job = new ProvisionNetworkJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.isNotNull(cluster.networkingStartedAt)
    assert.isNotNull(cluster.networkingCompletedAt)
    assert.isNull(cluster.networkingErrorAt)
    assert.equal(cluster.providerNetworkId, DIGITAL_OCEAN_NETWORK_OUTPUT.network_id.value)

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    app.default.container.bind('terraform.executor', () => MockTerraformExecutor as any)
  })

  test('should handle errors gracefully and set error timestamp for digital ocean', async ({ assert }) => {
    const { cluster } = await createFullyPopulatedDigitalOceanCluster()

    class ErrorMockExecutor extends MockTerraformExecutor {
      async apply(_options?: TerraformExecutionOptions): Promise<Subprocess> {
        return Promise.reject(new Error('Terraform apply failed'))
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => ErrorMockExecutor as any)

    const job = new ProvisionNetworkJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.isNotNull(cluster.networkingStartedAt)
    assert.isNull(cluster.networkingCompletedAt)
    assert.isNotNull(cluster.networkingErrorAt)
    assert.equal(cluster.status, 'unhealthy')
    assert.isNull(cluster.providerNetworkId)

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    app.default.container.bind('terraform.executor', () => MockTerraformExecutor as any)
  })
})
