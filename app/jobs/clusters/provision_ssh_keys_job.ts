import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { TerraformExecutor } from '#services/terraform/terraform_executor'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionLoadBalancersJob from './provision_load_balancers_job.js'

interface ProvisionSshKeysJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface SshKeysOutput {
  ssh_key_id: TerraformOutputValue
  ssh_key_name: TerraformOutputValue
  ssh_key_fingerprint: TerraformOutputValue
  ssh_key_labels: TerraformOutputValue
}

export default class ProvisionSshKeysJob extends Job {
  // This is the path to the file that is used to create the job
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionSshKeysJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster || !cluster?.sshKey) {
      return
    }

    cluster.sshKeysStartedAt = DateTime.now()
    cluster.sshKeysCompletedAt = null
    cluster.sshKeysErrorAt = null
    await cluster.save()

    try {

      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.SSH_KEYS)

      const executor = new TerraformExecutor(cluster.id, 'ssh-keys')
        .vars({
          ...cluster.cloudProvider?.getTerraformCredentials(),
          cluster_name: cluster.subdomainIdentifier,
          public_key: cluster.sshKey.publicKey
        })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as SshKeysOutput

      cluster.sshKey.providerId = output.ssh_key_id.value as string
      cluster.sshKeysCompletedAt = DateTime.now()

      await cluster.sshKey.save()

      await cluster.save()

      await queue.dispatch(ProvisionLoadBalancersJob, payload)
    } catch (error) {
      cluster.sshKeysErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(_payload: ProvisionSshKeysJobPayload) {
  }
}
