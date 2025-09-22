import { test } from '@japa/runner'
import sinon from 'sinon'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionTalosImageJob from '#jobs/clusters/provision_talos_image_job'
import ProvisionSshKeysJob from '#jobs/clusters/provision_ssh_keys_job'
import ProvisionNetworkJob from '#jobs/clusters/provision_network_job'
import { TerraformExecutionOptions } from '#services/terraform/terraform_executor'
import {
  createFullyPopulatedDigitalOceanCluster,
  setupTerraformExecutorMock,
  restoreTerraformExecutor,
  MockTerraformExecutor,
  DIGITAL_OCEAN_TALOS_IMAGE_OUTPUT,
  validateTerraformPlan,
  assertTerraformPlanValid,
} from './helpers/cluster_provisioning_helpers.js'

test.group('ProvisionTalosImageJob - Digital Ocean', (group) => {
  group.setup(() => {
    setupTerraformExecutorMock()
  })

  group.teardown(() => {
    restoreTerraformExecutor()
  })

  test('should provision talos image for digital ocean and dispatch ssh keys job', async ({
    assert,
  }) => {
    const { cluster } = await createFullyPopulatedDigitalOceanCluster()

    assert.isUndefined(cluster.talosImageStartedAt)
    assert.isUndefined(cluster.talosImageCompletedAt)
    assert.isUndefined(cluster.talosImageErrorAt)
    assert.isUndefined(cluster.providerImageId)
    assert.isUndefined(cluster.networkingCompletedAt)

    class TestMockExecutor extends MockTerraformExecutor {
      async apply(options?: TerraformExecutionOptions) {
        this.setMockOutput(DIGITAL_OCEAN_TALOS_IMAGE_OUTPUT)
        return super.apply(options)
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => TestMockExecutor)

    const dispatchSpy = sinon.spy(queue, 'dispatch')

    const job = new ProvisionTalosImageJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.isNotNull(cluster.talosImageStartedAt)
    assert.isNotNull(cluster.talosImageCompletedAt)
    assert.isNull(cluster.talosImageErrorAt)
    assert.equal(cluster.providerImageId, DIGITAL_OCEAN_TALOS_IMAGE_OUTPUT.talos_image_id.value)
    assert.isNotNull(cluster.networkingCompletedAt)

    assert.isTrue(dispatchSpy.calledOnce)
    assert.isTrue(dispatchSpy.calledWith(ProvisionSshKeysJob, { clusterId: cluster.id }))

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    dispatchSpy.restore()
    app.default.container.bind('terraform.executor', () => MockTerraformExecutor)
  })

  test('should execute terraform plan and generate valid output', async ({ assert }) => {
    const { cluster } = await createFullyPopulatedDigitalOceanCluster()

    class TestMockExecutor extends MockTerraformExecutor {
      async apply(options?: TerraformExecutionOptions) {
        this.setMockOutput(DIGITAL_OCEAN_TALOS_IMAGE_OUTPUT)
        return super.apply(options)
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => TestMockExecutor)

    const job = new ProvisionTalosImageJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.isNotNull(cluster.talosImageStartedAt)
    assert.isNotNull(cluster.talosImageCompletedAt)
    assert.isNull(cluster.talosImageErrorAt)
    assert.equal(cluster.providerImageId, DIGITAL_OCEAN_TALOS_IMAGE_OUTPUT.talos_image_id.value)

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    app.default.container.bind('terraform.executor', () => MockTerraformExecutor)
  })

  test('should handle errors gracefully and set error timestamp', async ({ assert }) => {
    const { cluster } = await createFullyPopulatedDigitalOceanCluster()

    class ErrorMockExecutor extends MockTerraformExecutor {
      async apply(_options?: TerraformExecutionOptions): Promise<never> {
        await super.plan({ ..._options, storePlanOutput: true })
        throw new Error('Terraform apply failed')
      }
    }

    const app = await import('@adonisjs/core/services/app')
    app.default.container.bind('terraform.executor', () => ErrorMockExecutor)

    const job = new ProvisionTalosImageJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.isNotNull(cluster.talosImageStartedAt)
    assert.isNull(cluster.talosImageCompletedAt)
    assert.isNotNull(cluster.talosImageErrorAt)
    assert.equal(cluster.status, 'unhealthy')
    assert.isNull(cluster.providerImageId)

    const planData = validateTerraformPlan({ clusterId: cluster.id, cluster })
    assertTerraformPlanValid(planData, { clusterId: cluster.id, cluster }, assert)

    app.default.container.bind('terraform.executor', () => MockTerraformExecutor)
  })

  test('should skip terraform execution for hetzner and use pre-provisioned images', async ({
    assert,
  }) => {
    const { cluster, cloudProvider } = await createFullyPopulatedDigitalOceanCluster()

    cloudProvider.type = 'hetzner'
    cloudProvider.providerImageAmd64 = 'hetzner-amd64-image-id'
    cloudProvider.providerImageArm64 = 'hetzner-arm64-image-id'
    await cloudProvider.save()

    cluster.serverType = 'cpx11'
    await cluster.save()

    const dispatchSpy = sinon.spy(queue, 'dispatch')

    const job = new ProvisionTalosImageJob()
    await job.handle({ clusterId: cluster.id })

    await cluster.refresh()

    assert.equal(cluster.providerImageId, 'hetzner-amd64-image-id')
    assert.isNotNull(cluster.talosImageCompletedAt)

    assert.isTrue(dispatchSpy.calledOnce)
    assert.isTrue(dispatchSpy.calledWith(ProvisionNetworkJob, { clusterId: cluster.id }))

    dispatchSpy.restore()
  })
})
