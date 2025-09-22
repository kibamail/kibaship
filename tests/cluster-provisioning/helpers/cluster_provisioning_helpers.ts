import { DateTime } from 'luxon'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import db from '@adonisjs/lucid/services/db'
import app from '@adonisjs/core/services/app'
import env from '#start/env'
import { talosVersion as configTalosVersion, talosFactoryHash } from '#config/app'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import { randomBytes } from 'node:crypto'
import User from '#models/user'
import Workspace from '#models/workspace'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import ClusterNode from '#models/cluster_node'
import ClusterNodeStorage from '#models/cluster_node_storage'
import ClusterSshKey from '#models/cluster_ssh_key'
import ClusterLoadBalancer from '#models/cluster_load_balancer'
import {
  TerraformExecutor,
  TerraformExecutionOptions,
  TerraformStage,
  TerraformPlanData,
  DigitalOceanCustomImageResource,
  DigitalOceanImagesFilter,
} from '#services/terraform/terraform_executor'
import { Assert } from '@japa/assert'

/**
 * Creates a fully populated cluster for Digital Ocean with all required infrastructure
 * This includes nodes, storages, SSH keys, load balancers, etc.
 */
export async function createFullyPopulatedDigitalOceanCluster() {
  const trx = await db.transaction()

  try {
    // Create user and workspace
    const email = `test_${randomBytes(4).toString('hex')}@example.com`
    const user = new User()
    user.email = email
    user.password = 'testpassword123'
    user.useTransaction(trx)
    await user.save()

    const [username] = email.split('@')
    const workspace = new Workspace()
    workspace.name = `${username}'s Workspace`
    workspace.slug = username.toLowerCase().replace(/[^a-z0-9]/g, '-')
    workspace.userId = user.id
    workspace.useTransaction(trx)
    await workspace.save()

    // Create Digital Ocean cloud provider
    const cloudProvider = new CloudProvider()
    cloudProvider.name = 'Test Digital Ocean Provider'
    cloudProvider.type = 'digital_ocean'
    cloudProvider.workspaceId = workspace.id
    cloudProvider.credentials = {
      token: env.get('DIGITAL_OCEAN_API_TESTING', 'test-do-token-12345'),
    }
    cloudProvider.providerImageProvisioningCompletedAt = DateTime.now()
    cloudProvider.useTransaction(trx)
    await cloudProvider.save()

    // Create cluster
    const cluster = new Cluster()
    cluster.location = 'nyc3'
    cluster.controlPlaneEndpoint = `https://kube.test-cluster-${randomBytes(4).toString('hex')}.kibaship.com`
    cluster.subdomainIdentifier = `test-cluster-${randomBytes(4).toString('hex')}.kibaship.com`
    cluster.kind = 'all_purpose'
    cluster.workspaceId = workspace.id
    cluster.cloudProviderId = cloudProvider.id
    cluster.serverType = 's-2vcpu-2gb'
    cluster.status = 'provisioning'
    cluster.talosVersion = configTalosVersion
    cluster.networkIpRange = '10.0.0.0/16'
    cluster.subnetIpRange = '10.0.1.0/24'
    cluster.controlPlanesVolumeSize = 50
    cluster.workersVolumeSize = 100
    cluster.useTransaction(trx)
    await cluster.save()

    // Create SSH key
    const sshKey = new ClusterSshKey()
    sshKey.clusterId = cluster.id
    sshKey.publicKey = 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... test-key'
    sshKey.privateKey =
      '-----BEGIN OPENSSH PRIVATE KEY-----\ntest-private-key\n-----END OPENSSH PRIVATE KEY-----'
    sshKey.useTransaction(trx)
    await sshKey.save()

    // Create load balancers
    const kubeLoadBalancer = new ClusterLoadBalancer()
    kubeLoadBalancer.clusterId = cluster.id
    kubeLoadBalancer.type = 'cluster'
    kubeLoadBalancer.publicIpv4Address = '192.168.1.100'
    kubeLoadBalancer.privateIpv4Address = '10.0.1.100'
    kubeLoadBalancer.useTransaction(trx)
    await kubeLoadBalancer.save()

    const ingressLoadBalancer = new ClusterLoadBalancer()
    ingressLoadBalancer.clusterId = cluster.id
    ingressLoadBalancer.type = 'ingress'
    ingressLoadBalancer.publicIpv4Address = '192.168.1.101'
    ingressLoadBalancer.privateIpv4Address = '10.0.1.101'
    ingressLoadBalancer.useTransaction(trx)
    await ingressLoadBalancer.save()

    // Create control plane nodes
    const controlPlaneNodes = []
    for (let i = 0; i < 3; i++) {
      const node = new ClusterNode()
      node.clusterId = cluster.id
      node.type = 'master'
      node.status = 'provisioning'
      node.serverType = 's-2vcpu-2gb'
      node.ipv4Address = `192.168.1.${10 + i}`
      node.privateIpv4Address = `10.0.1.${10 + i}`
      node.useTransaction(trx)
      await node.save()
      controlPlaneNodes.push(node)

      // Create storage for control plane node
      const storage = new ClusterNodeStorage()
      storage.clusterNodeId = node.id
      storage.status = 'provisioning'
      storage.size = 50
      storage.useTransaction(trx)
      await storage.save()
    }

    // Create worker nodes
    const workerNodes = []
    for (let i = 0; i < 3; i++) {
      const node = new ClusterNode()
      node.clusterId = cluster.id
      node.type = 'worker'
      node.status = 'provisioning'
      node.serverType = 's-2vcpu-2gb'
      node.ipv4Address = `192.168.1.${20 + i}`
      node.privateIpv4Address = `10.0.1.${20 + i}`
      node.useTransaction(trx)
      await node.save()
      workerNodes.push(node)

      // Create storage for worker node
      const storage = new ClusterNodeStorage()
      storage.clusterNodeId = node.id
      storage.status = 'provisioning'
      storage.size = 100
      storage.useTransaction(trx)
      await storage.save()
    }

    await trx.commit()

    // Load all relationships
    await cluster.load('cloudProvider')
    await cluster.load('nodes', (query) => query.preload('storages'))
    await cluster.load('sshKey')
    await cluster.load('loadBalancers')
    await cluster.load('workspace', (query) => query.preload('user'))

    return {
      user,
      workspace,
      cloudProvider,
      cluster,
      controlPlaneNodes,
      workerNodes,
      sshKey,
      loadBalancers: [kubeLoadBalancer, ingressLoadBalancer],
    }
  } catch (error) {
    await trx.rollback()
    throw error
  }
}

export interface TerraformOutput {
  sensitive: boolean
  type: string
  value: string | number | boolean
}

export interface MockTerraformOutputData {
  [key: string]: TerraformOutput
}

export class MockTerraformExecutor extends TerraformExecutor {
  private mockOutputData: MockTerraformOutputData | null = null

  constructor(clusterId: string, stage: TerraformStage) {
    super(clusterId, stage)
  }

  setMockOutput(outputData: MockTerraformOutputData): this {
    this.mockOutputData = outputData
    return this
  }

  async apply(_options?: TerraformExecutionOptions): Promise<any> {
    await super.plan({ ..._options, storePlanOutput: true })
    return Promise.resolve({})
  }

  async output(): Promise<any> {
    if (!this.mockOutputData) {
      throw new Error('Mock output data not set. Use setMockOutput() to set the expected output.')
    }
    return Promise.resolve({
      stdout: JSON.stringify(this.mockOutputData),
    })
  }
}

/**
 * Digital Ocean Talos Image terraform output for testing
 */
export const DIGITAL_OCEAN_TALOS_IMAGE_OUTPUT = {
  talos_image_id: {
    sensitive: false,
    type: 'string',
    value: '123456789',
  },
}

export function setupTerraformExecutorMock() {
  app.container.bind('terraform.executor', () => MockTerraformExecutor)
}

export function restoreTerraformExecutor() {
  app.container.bind('terraform.executor', () => TerraformExecutor)
}

export interface TerraformPlanValidationOptions {
  clusterId: string
  cluster: Cluster
}

export function validateTerraformPlan(options: TerraformPlanValidationOptions): TerraformPlanData {
  const planOutputFile = join(
    process.cwd(),
    'storage',
    'terraform',
    'clusters',
    options.clusterId,
    'talos-image',
    'terraform-plan-output.json'
  )

  const planContent = readFileSync(planOutputFile, 'utf-8')
  const planData: TerraformPlanData = JSON.parse(planContent)

  return planData
}

export function assertTerraformPlanValid(
  planData: TerraformPlanData,
  options: TerraformPlanValidationOptions,
  assert: Assert
): void {
  const { cluster } = options

  const expectedArchitecture =
    cluster.cloudProvider.type === 'digital_ocean'
      ? 'amd64'
      : (() => {
          const serverTypes = CloudProviderDefinitions.serverTypes(cluster.cloudProvider.type)
          const serverTypeInfo = serverTypes[cluster.serverType]
          return serverTypeInfo.arch === 'arm' ? 'arm64' : 'amd64'
        })()

  const expectedFactoryHash = talosFactoryHash
  const expectedTalosVersion = cluster.talosVersion
  const expectedRegion = cluster.location

  assert.equal(planData.format_version, '1.2')
  assert.isString(planData.terraform_version)
  assert.isTrue(planData.applyable)
  assert.isTrue(planData.complete)
  assert.isFalse(planData.errored)

  assert.isUndefined(planData.variables, 'Variables section should be removed for security')

  const customImageResource = planData.planned_values?.root_module?.resources?.find(
    (r) => r.type === 'digitalocean_custom_image' && r.name === 'talos_image'
  )
  assert.isDefined(
    customImageResource,
    'digitalocean_custom_image.talos_image resource should be planned'
  )

  const imageValues = customImageResource!.values as unknown as DigitalOceanCustomImageResource

  const expectedImageName = `kibaship-talos-linux-${expectedTalosVersion}-${expectedRegion}`
  assert.equal(imageValues.name, expectedImageName)

  assert.deepEqual(imageValues.regions, [expectedRegion])

  assert.equal(imageValues.distribution, 'Unknown OS')

  const expectedUrlPattern = new RegExp(
    `^https://factory\\.talos\\.dev/image/${expectedFactoryHash}/${expectedTalosVersion}/digital-ocean-${expectedArchitecture}\\.raw\\.gz$`
  )
  assert.match(
    imageValues.url,
    expectedUrlPattern,
    'Talos factory URL should match expected pattern'
  )

  const resourceChange = planData.resource_changes?.find(
    (r) => r.type === 'digitalocean_custom_image' && r.name === 'talos_image'
  )
  assert.isDefined(resourceChange, 'digitalocean_custom_image resource change should exist')
  assert.deepEqual(resourceChange!.change.actions, ['create'])

  assert.property(planData.output_changes, 'talos_image_id')
  const outputChange = planData.output_changes!['talos_image_id'] as { actions: string[] }
  assert.deepEqual(outputChange.actions, ['create'])

  const existingImageDataSource = planData.prior_state?.values?.root_module?.resources?.find(
    (r) => r.type === 'digitalocean_images' && r.name === 'existing_talos_image'
  )
  assert.isDefined(existingImageDataSource, 'existing_talos_image data source should exist')

  const filters = (existingImageDataSource!.values as { filter: DigitalOceanImagesFilter[] }).filter
  assert.lengthOf(filters, 2, 'Should have exactly 2 filters')

  const nameFilter = filters.find((f) => f.key === 'name')
  const typeFilter = filters.find((f) => f.key === 'type')

  assert.isDefined(nameFilter, 'Name filter should exist')
  assert.isDefined(typeFilter, 'Type filter should exist')
  assert.deepEqual(nameFilter!.values, [expectedImageName])
  assert.deepEqual(typeFilter!.values, ['custom'])

  assert.property(planData.configuration?.provider_config, 'digitalocean')
  assert.property(planData.configuration?.provider_config, 'talos')

  const doProvider = planData.configuration!.provider_config!.digitalocean
  const talosProvider = planData.configuration!.provider_config!.talos

  assert.equal(doProvider.full_name, 'registry.terraform.io/digitalocean/digitalocean')
  assert.equal(doProvider.version_constraint, '~> 2.66')
  assert.equal(talosProvider.full_name, 'registry.terraform.io/siderolabs/talos')
  assert.equal(talosProvider.version_constraint, '0.9.0')
}
