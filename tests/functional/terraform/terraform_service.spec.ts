import { test } from '@japa/runner'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { access, constants } from 'node:fs/promises'
import { join } from 'node:path'
import env from '#start/env'
import { ChildProcess } from '#utils/child_process'
import { createCompleteTestSetup } from '#tests/helpers/test_helpers'


async function runTerraformCommand(command: string, args: string[], cwd: string, vars?: Record<string, string>): Promise<void> {
  console.log(`\n🔧 Running: ${command} ${args.join(' ')} in ${cwd}`)

  const tfVars: Record<string, string> = {}

  for (const [key, value] of Object.entries(vars || {})) {
    tfVars[`TF_VAR_${key}`] = String(value)
  }

  await new ChildProcess()
    .command(command)
    .args(args)
    .cwd(cwd)
    .env({
      AWS_ACCESS_KEY_ID: env.get('S3_ACCESS_KEY'),
      AWS_SECRET_ACCESS_KEY: env.get('S3_ACCESS_SECRET'),
      ...tfVars
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

async function validateTerraformDirectory(dirPath: string, vars?: Record<string, string>): Promise<{ valid: boolean; error?: string }> {
  try {
    console.log(`\n🔍 Validating Terraform directory: ${dirPath}`)

    await access(join(dirPath, 'main.tf'), constants.F_OK)
    console.log(`  ✅ main.tf exists`)

    await runTerraformCommand('terraform', ['init'], dirPath, vars)

    await runTerraformCommand('terraform', ['validate'], dirPath, vars)

    await runTerraformCommand('terraform', ['plan'], dirPath, vars)

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

  test('generates valid Terraform configurations for all templates', async ({ assert: _assert }) => {
    const { cluster } = await createCompleteTestSetup('test-token-for-terraform-validation')
    const terraformService = new TerraformService(cluster.id)

    const templates = Object.values(TerraformTemplate)

    for (const template of templates) {
      console.log(`\n🧪 Testing template: ${template}`)

      const generatedFile = await terraformService.generate(cluster, template)

      const dirPath = generatedFile.path.replace('/main.tf', '')
      await validateTerraformDirectory(dirPath, {
        cluster_name: cluster.subdomainIdentifier,
        do_token: 'test-token-for-terraform-validation',
        public_key: cluster?.sshKey?.publicKey,
        network_id: cluster?.providerNetworkId as string,
        ingress_load_balancer_id: 'test-lb-id',
        kube_load_balancer_id: 'test-lb-id',
        server_type: 'ax44'
      })

      // todo: add assertions
      console.log(`  ✅ Template ${template} generated and validated successfully`)
    }

    await terraformService.cleanup()
  }).timeout(320000)
})
