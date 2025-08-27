import { test } from '@japa/runner'
import User from '#models/user'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { randomBytes } from 'node:crypto'
import redis from '@adonisjs/redis/services/main'
import { spawn } from 'node:child_process'
import { access, constants } from 'node:fs/promises'
import { join } from 'node:path'
import db from '@adonisjs/lucid/services/db'

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

async function createTestCluster(workspaceId: string, cloudProviderId: string) {
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

    // Set additional cluster fields that are required for template generation
    cluster.controlPlaneEndpoint = `https://kube.${cluster.subdomainIdentifier}`
    // Note: privateNetworkCidr and publicDomain might not be in the database schema yet
    // cluster.privateNetworkCidr = '10.0.0.0/16'
    // cluster.publicDomain = cluster.subdomainIdentifier

    // These should already be set by createWithInfrastructure
    // cluster.status = 'provisioning'
    // cluster.kind = 'all_purpose'

    await cluster.save()
    await trx.commit()

    // Load relations needed for template generation
    await cluster.load('cloudProvider')
    await cluster.load('nodes')
    await cluster.load('sshKeys')

    // Ensure nodes have slugs (they should be auto-generated)
    if (cluster.nodes && cluster.nodes.length > 0) {
      for (const node of cluster.nodes) {
        if (!node.slug) {
          // This should not happen as slug is auto-generated, but just in case
          throw new Error(`Node ${node.id} is missing slug`)
        }
      }
    }

    // Ensure SSH keys exist
    if (!cluster.sshKeys || cluster.sshKeys.length === 0) {
      throw new Error('Cluster is missing SSH keys')
    }

    return cluster
  } catch (error) {
    await trx.rollback()
    throw error
  }
}



async function runTerraformCommand(command: string, args: string[], cwd: string): Promise<void> {
  return new Promise((resolve, reject) => {
    console.log(`\n🔧 Running: terraform ${args.join(' ')} in ${cwd}`)

    const process = spawn(command, args, {
      cwd,
      stdio: ['pipe', 'pipe', 'pipe']
    })

    let stdout = ''
    let stderr = ''

    process.stdout.on('data', (data) => {
      const output = data.toString()
      stdout += output
      // Stream stdout in real-time with prefix
      output.split('\n').forEach((line: string) => {
        if (line.trim()) {
          console.log(`  📤 ${line}`)
        }
      })
    })

    process.stderr.on('data', (data) => {
      const output = data.toString()
      stderr += output
      // Stream stderr in real-time with prefix
      output.split('\n').forEach((line: string) => {
        if (line.trim()) {
          console.log(`  ❌ ${line}`)
        }
      })
    })

    process.on('close', (code) => {
      if (code === 0) {
        console.log(`  ✅ terraform ${args.join(' ')} completed successfully`)
        resolve()
      } else {
        console.log(`  💥 terraform ${args.join(' ')} failed with exit code ${code}`)
        reject(new Error(`Command failed with exit code ${code}. stderr: ${stderr}`))
      }
    })

    process.on('error', (error) => {
      console.log(`  💥 Process error: ${error.message}`)
      reject(error)
    })
  })
}

async function validateTerraformDirectory(dirPath: string): Promise<{ valid: boolean; error?: string }> {
  try {
    console.log(`\n🔍 Validating Terraform directory: ${dirPath}`)

    // Check if main.tf exists
    await access(join(dirPath, 'main.tf'), constants.F_OK)
    console.log(`  ✅ main.tf exists`)

    // Run terraform init
    await runTerraformCommand('terraform', ['init'], dirPath)

    // Run terraform validate
    await runTerraformCommand('terraform', ['validate'], dirPath)

    console.log(`  🎉 Directory ${dirPath} validation completed successfully\n`)
    return { valid: true }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    console.log(`  💥 Directory ${dirPath} validation failed: ${errorMessage}\n`)
    return {
      valid: false,
      error: errorMessage
    }
  }
}

test.group('TerraformService Integration', () => {

  test('generates valid Terraform configurations for all templates', async ({ assert }) => {
    const { workspaceId } = await createUserWithWorkspace()

    const cloudProvider = await CloudProvider.create({
      name: 'Test Hetzner Provider',
      type: 'hetzner',
      workspaceId: workspaceId,
      credentials: { token: 'test-token-for-terraform-validation' },
    })

    const cluster = await createTestCluster(workspaceId, cloudProvider.id)
    const terraformService = new TerraformService(cluster.id)

    // Verify cluster has all required data
    assert.isTrue(cluster.nodes && cluster.nodes.length > 0, 'Cluster should have nodes')
    assert.isTrue(cluster.sshKeys && cluster.sshKeys.length > 0, 'Cluster should have SSH keys')
    assert.isTrue(!!cluster.cloudProvider, 'Cluster should have cloud provider')
    assert.isTrue(!!cluster.cloudProvider.credentials?.token, 'Cloud provider should have token')

    const controlPlanes = cluster.nodes.filter(node => node.type === 'master')
    const workers = cluster.nodes.filter(node => node.type === 'worker')
    assert.lengthOf(controlPlanes, 3, 'Should have 3 control plane nodes')
    assert.lengthOf(workers, 3, 'Should have 3 worker nodes')

    // Test each template
    const templates = Object.values(TerraformTemplate)
    const expectedContent = {
      [TerraformTemplate.NETWORK]: ['hcloud_network', 'hcloud_network_subnet'],
      [TerraformTemplate.SSH_KEYS]: ['hcloud_ssh_key', 'var.public_key'],
      [TerraformTemplate.LOAD_BALANCERS]: ['hcloud_load_balancer', 'ingress', 'kube'],
      [TerraformTemplate.SERVERS]: ['hcloud_server', 'control_plane_', 'worker_'],
      [TerraformTemplate.VOLUMES]: ['hcloud_volume', 'var.control_plane_server_ids', 'var.worker_server_ids']
    }

    for (const template of templates) {
      console.log(`\n🧪 Testing template: ${template}`)

      const generatedFile = await terraformService.generate(cluster, template)

      // Basic assertions
      assert.isObject(generatedFile, `Template ${template} should return an object`)
      assert.equal(generatedFile.name, template.replace('.tf', ''), `Template ${template} should have correct name`)
      assert.isString(generatedFile.content, `Template ${template} should have content`)
      assert.isString(generatedFile.path, `Template ${template} should have path`)

      // Content-specific assertions
      const expectedStrings = expectedContent[template]
      for (const expectedString of expectedStrings) {
        assert.include(
          generatedFile.content,
          expectedString,
          `Template ${template} should contain '${expectedString}'`
        )
      }

      // Validate Terraform syntax
      const dirPath = generatedFile.path.replace('/main.tf', '')
      const result = await validateTerraformDirectory(dirPath)

      assert.isTrue(
        result.valid,
        `Template '${template}' failed validation: ${result.error || 'Unknown error'}`
      )

      console.log(`  ✅ Template ${template} generated and validated successfully`)
    }

    await terraformService.cleanup()
  }).timeout(180000)
})
