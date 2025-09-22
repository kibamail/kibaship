import { Job } from '@rlanz/bull-queue'
import Cluster from '#models/cluster'
import { createExecutor } from '#services/terraform/main'
import { TerraformService, TerraformTemplate } from '#services/terraform/terraform_service'
import { DateTime } from 'luxon'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionVolumesJob from './provision_volumes_job.js'
import drive from '@adonisjs/drive/services/main'
import yaml from 'yaml'

interface ProvisionServersJobPayload {
  clusterId: string
}

interface TerraformOutputValue {
  sensitive: boolean
  type: string
  value: string | number | object
}

interface ServersOutput {
  // Allow dynamic key access for node-specific outputs
  [key: string]: TerraformOutputValue
  
  // Summary outputs
  control_plane_server_ids: TerraformOutputValue
  worker_server_ids: TerraformOutputValue
  
  // Talos configuration outputs
  talos_config: TerraformOutputValue
  control_plane_machine_configuration: TerraformOutputValue
  worker_machine_configuration: TerraformOutputValue
  kubeconfig: TerraformOutputValue
}

export default class ProvisionServersJob extends Job {
  // This is the path to the file that is used to create the job
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionServersJobPayload) {
    const cluster = await Cluster.complete(payload.clusterId)

    if (!cluster) {
      return
    }

    cluster.serversStartedAt = DateTime.now()
    cluster.serversCompletedAt = null
    cluster.serversErrorAt = null
    await cluster.save()

    try {
      const terraform = new TerraformService(payload.clusterId)
      await terraform.generate(cluster, TerraformTemplate.SERVERS)

      const ingressLoadBalancer = cluster.loadBalancers.find(lb => lb.type === 'ingress')
      const kubeLoadBalancer = cluster.loadBalancers.find(lb => lb.type === 'cluster')

      const executor = (await createExecutor(cluster.id, 'servers'))
        .vars({
          ...cluster.cloudProvider?.getTerraformCredentials(),
          location: cluster.location,
          cluster_name: cluster.subdomainIdentifier,
          server_type: cluster.serverType,
          network_id: cluster.providerNetworkId || '',
          ssh_key_id: cluster.sshKey?.providerId || '',
          kube_load_balancer_id: kubeLoadBalancer?.providerId || '',
          ingress_load_balancer_id: ingressLoadBalancer?.providerId || '',
        })

      await executor.init()
      await executor.apply({ autoApprove: true })

      const { stdout } = await executor.output()
      const output = JSON.parse(stdout as string) as ServersOutput

      await this.updateClusterNodes(cluster.id, output)
      await this.saveTalosConfigurations(cluster.id, output)

      cluster.serversCompletedAt = DateTime.now()

      await cluster.save()

      await queue.dispatch(ProvisionVolumesJob, payload)
    } catch (error) {
      cluster.serversErrorAt = DateTime.now()

      await cluster.save()
      throw error
    }
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(_payload: ProvisionServersJobPayload) { }

  private async updateClusterNodes(
    clusterId: string,
    output: ServersOutput
  ): Promise<void> {
    const cluster = await Cluster.query()
      .where('id', clusterId)
      .preload('nodes')
      .firstOrFail()

    for (const node of cluster.nodes) {
      const serverIdKey = `${node.type === 'master' ? 'control_plane' : 'worker'}_${node.slug}_server_id`
      const publicIpKey = `${node.type === 'master' ? 'control_plane' : 'worker'}_${node.slug}_public_ip`
      const privateIpKey = `${node.type === 'master' ? 'control_plane' : 'worker'}_${node.slug}_private_ip`

      const serverId = output[serverIdKey]?.value as string
      const publicIp = output[publicIpKey]?.value as string
      const privateIp = output[privateIpKey]?.value as string

      node.ipv4Address = publicIp
      node.privateIpv4Address = privateIp
      node.status = 'healthy'
      node.providerId = serverId
      await node.save()
    }

    // Save configurations to database (encrypted)
    const talosConfigRaw = output.talos_config?.value as Cluster['talosConfig']
    const kubeConfig = output.kubeconfig?.value as string

    if (talosConfigRaw) {
      cluster.talosConfig = talosConfigRaw
    }

    if (kubeConfig) {
      const kubeconfigData = yaml.parse(kubeConfig)
      const clusterData = kubeconfigData.clusters[0].cluster
      const userData = kubeconfigData.users[0].user

      cluster.kubeconfig = {
        host: clusterData.server,
        clientCertificate: userData['client-certificate-data'],
        clientKey: userData['client-key-data'],
        clusterCaCertificate: clusterData['certificate-authority-data'],
      }
    }

    await cluster.save()
  }

  private async saveTalosConfigurations(
    clusterId: string,
    output: ServersOutput
  ): Promise<void> {
    const disk = drive.use('fs')
    const configsPath = `talos-configs/${clusterId}`
    const terraformConfigsPath = `terraform/clusters/${clusterId}/configs`
    
    const cluster = await Cluster.complete(clusterId)
    if (!cluster) return

    const kubeConfig = output.kubeconfig?.value as string
    const talosConfig = output.talos_config?.value as Cluster['talosConfig']
    const workerConfig = output.worker_machine_configuration?.value as string
    const controlPlaneConfig = output.control_plane_machine_configuration?.value as string

    if (talosConfig && cluster.nodes) {
      const controlPlaneEndpoints = cluster.nodes
        .filter(node => node.type === 'master')
        .map(node => node.ipv4Address)
        .filter(Boolean)

      const talosConfigYaml = {
        context: cluster.subdomainIdentifier,
        contexts: {
          [cluster.subdomainIdentifier]: {
            endpoints: controlPlaneEndpoints,
            ca: talosConfig.ca_certificate,
            crt: talosConfig.client_certificate,
            key: talosConfig.client_key
          }
        }
      }

      await disk.put(`${configsPath}/talosconfig`, yaml.stringify(talosConfigYaml))
    }

    // Save control plane machine configuration as YAML
    if (controlPlaneConfig) {
      await disk.put(`${configsPath}/controlplane.yaml`, controlPlaneConfig)
    }

    // Save worker machine configuration as YAML
    if (workerConfig) {
      await disk.put(`${configsPath}/worker.yaml`, workerConfig)
    }

    // Save the actual kubeconfig generated by Talos (for kubernetes-config step)
    if (kubeConfig) {
      await disk.put(`${configsPath}/kubeconfig.yaml`, kubeConfig)
      await disk.put(`${terraformConfigsPath}/kubeconfig.yaml`, kubeConfig)
    }
  }
}