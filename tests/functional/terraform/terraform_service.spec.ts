import { test } from '@japa/runner'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { access, constants } from 'node:fs/promises'
import { join } from 'node:path'
import env from '#start/env'
import { ChildProcess } from '#utils/child_process'
import { createCompleteTestSetup } from '#tests/helpers/test_helpers'


async function runTerraformCommand(command: string, args: string[], cwd: string): Promise<void> {
  console.log(`\n🔧 Running: ${command} ${args.join(' ')} in ${cwd}`)

  await new ChildProcess()
    .command(command)
    .args(args)
    .cwd(cwd)
    .env({
      AWS_ACCESS_KEY_ID: env.get('S3_ACCESS_KEY'),
      AWS_SECRET_ACCESS_KEY: env.get('S3_ACCESS_SECRET'),
    })
    .onStdout((data) => {
      data.split('\n').forEach((line: string) => {
        if (line.trim()) {
          console.log(`  📤 ${line}`)
        }
      })
    })
    .onStderr((data) => {
      data.split('\n').forEach((line: string) => {
        if (line.trim()) {
          console.log(`  ❌ ${line}`)
        }
      })
    })
    .onClose((code) => {
      if (code === 0) {
        console.log(`  ✅ ${command} ${args.join(' ')} completed successfully`)
      } else {
        console.log(`  💥 ${command} ${args.join(' ')} failed with exit code ${code}`)
      }
    })
    .onError((error) => {
      console.log(`  💥 Process error: ${error.message}`)
    })
    .execute()
}

async function validateTerraformDirectory(dirPath: string): Promise<{ valid: boolean; error?: string }> {
  try {
    console.log(`\n🔍 Validating Terraform directory: ${dirPath}`)

    await access(join(dirPath, 'main.tf'), constants.F_OK)
    console.log(`  ✅ main.tf exists`)

    await runTerraformCommand('terraform', ['init'], dirPath)

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
    const { cluster } = await createCompleteTestSetup('test-token-for-terraform-validation')
    const terraformService = new TerraformService(cluster.id)

    const templates = Object.values(TerraformTemplate)
    const expectedContent = {
      [TerraformTemplate.TALOS_IMAGE]: ['hcloud_image', 'talos'],
      [TerraformTemplate.NETWORK]: ['hcloud_network', 'hcloud_network_subnet'],
      [TerraformTemplate.SSH_KEYS]: ['hcloud_ssh_key', 'var.public_key'],
      [TerraformTemplate.LOAD_BALANCERS]: ['hcloud_load_balancer', 'ingress', 'kube'],
      [TerraformTemplate.SERVERS]: ['hcloud_server', 'control_plane_', 'worker_'],
      [TerraformTemplate.VOLUMES]: ['hcloud_volume', 'var.control_plane_server_ids', 'var.worker_server_ids'],
      [TerraformTemplate.KUBERNETES]: ['talos_machine_configuration_apply', 'talos_machine_bootstrap'],
      [TerraformTemplate.KUBERNETES_CONFIG]: ['helm_release', 'cilium', 'helm_repository'],
      [TerraformTemplate.KUBERNETES_BOOT]: ['kubernetes_manifest', 'linstor']
    }

    for (const template of templates) {
      console.log(`\n🧪 Testing template: ${template}`)

      const generatedFile = await terraformService.generate(cluster, template)

      assert.isObject(generatedFile, `Template ${template} should return an object`)
      assert.equal(generatedFile.name, template.replace('.tf', ''), `Template ${template} should have correct name`)
      assert.isString(generatedFile.content, `Template ${template} should have content`)
      assert.isString(generatedFile.path, `Template ${template} should have path`)

      const expectedStrings = expectedContent[template]
      for (const expectedString of expectedStrings) {
        assert.include(
          generatedFile.content,
          expectedString,
          `Template ${template} should contain '${expectedString}'`
        )
      }

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
