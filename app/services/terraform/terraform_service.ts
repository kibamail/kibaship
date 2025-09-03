import edge from 'edge.js'
import app from '@adonisjs/core/services/app'
import drive from '@adonisjs/drive/services/main'
import env from '#start/env'
import { join } from 'node:path'
import { talosVersion } from '#config/app'
import Cluster from '#models/cluster'
import { ModelObject } from '@adonisjs/lucid/types/model'

export enum TerraformTemplate {
  TALOS_IMAGE = 'talos-image.tf',
  NETWORK = 'network.tf',
  SSH_KEYS = 'ssh-keys.tf',
  LOAD_BALANCERS = 'load-balancers.tf',
  SERVERS = 'servers.tf',
  VOLUMES = 'volumes.tf'
}

export interface TemplateContext {
  cluster_id: string
  cluster_name: string
  cluster_talos_version: string
  cluster_region: string
  cluster_network_id: string
  network_zone: string
  location: string
  s3_region: string
  s3_bucket: string
  cluster_talos_image: string
  cluster_ssh_key_id: string
  control_planes: Array<ModelObject & {
    id: string
    slug: string
    type: string
  }>
  workers: Array<ModelObject & {
    id: string
    slug: string
    type: string
  }>
  public_key: string
  control_planes_volume_size: number
  workers_volume_size: number
  cluster_load_balancer_domain: string
}

export interface TerraformFile {
  name: string
  content: string
  key: string
  path: string
}

/**
 * Terraform template service that handles Edge template rendering and file generation.
 *
 * This service manages Terraform template generation for cluster infrastructure:
 * - Renders Edge templates by name for any cloud provider
 * - Stores cluster Terraform files using AdonisJS Drive
 * - Provides cleanup functionality for cluster files
 *
 * All methods follow the $trycatch pattern and return [result, error] tuples
 * for consistent error handling throughout the application.
 */
export class TerraformService {
  private edge: typeof edge
  private disk: ReturnType<typeof drive.use>
  private clusterBasePath: string

  constructor(clusterId: string) {
    this.edge = edge
    this.disk = drive.use('fs')

    this.clusterBasePath = `terraform/clusters/${clusterId}`
  }

  /**
   * Converts a Drive key to a file system path
   */
  private keyToPath(key: string): string {
    return join(app.makePath('storage'), key)
  }

  /**
   * Initializes the service by setting up Edge views
   */
  async init() {
    const terraformViewsPath = app.makePath('resources/views/clusters/terraform')
    this.edge.mount(terraformViewsPath)
  }

  /**
   * Writes a single Terraform file using Drive
   */
  async writeTerraformFile(file: Omit<TerraformFile, 'key' | 'path'>) {
    const templateName = file.name.replace('.tf', '')
    const fileKey = `${this.clusterBasePath}/${templateName}/main.tf`

    await this.disk.put(fileKey, file.content)

    return fileKey
  }

  /**
   * Writes multiple Terraform files using Drive
   */
  async writeTerraformFiles(files: Omit<TerraformFile, 'key' | 'path'>[]) {
    const terraformFiles: TerraformFile[] = []

    for (const file of files) {
      const fileKey = await this.writeTerraformFile(file)
      const templateName = file.name.replace('.tf', '')

      terraformFiles.push({
        name: templateName,
        content: file.content,
        key: fileKey,
        path: this.keyToPath(fileKey)
      })
    }

    return terraformFiles
  }

  /**
   * Checks if a file exists in Drive
   */
  async fileExists(key: string): Promise<boolean> {
    try {
      await this.disk.get(key)
      return true
    } catch {
      return false
    }
  }

  /**
   * Gets a Terraform file content from Drive
   */
  async getTerraformFile(templateName: string): Promise<string | null> {
    const fileKey = `${this.clusterBasePath}/${templateName}/main.tf`
    return this.disk.get(fileKey)
  }

  /**
   * Removes all cluster Terraform files from Drive
   */
  async cleanup() {
    try {
      await this.disk.deleteAll(this.clusterBasePath)
    } catch {
      // Ignore errors if path doesn't exist
    }
  }

  /**
   * Generates a single Terraform template for a cluster and stores it using Drive
   * @param cluster - The cluster to generate template for
   * @param templateName - The specific template name to generate
   */
  async generate(cluster: Cluster, templateName: TerraformTemplate): Promise<TerraformFile> {
    await this.init()

    const context = this.buildTemplateContext(cluster)
    const cloudProviderType = cluster.cloudProvider.type

    const content = await this.edge.render(`${cloudProviderType}/${templateName}`, context)
    const terraformFile: Omit<TerraformFile, 'key' | 'path'> = {
      name: templateName,
      content
    }

    const fileKey = await this.writeTerraformFile(terraformFile)

    return {
      name: templateName.replace('.tf', ''),
      content,
      key: fileKey,
      path: this.keyToPath(fileKey)
    }
  }



  /**
   * Builds comprehensive template context for all infrastructure templates
   */
  private buildTemplateContext(cluster: Cluster): TemplateContext {
    const publicKey = cluster.sshKey?.publicKey

    const controlPlanes = (cluster.nodes?.filter(node => node.type === 'master')?.map(node => node.toJSON()) || []) as TemplateContext['control_planes']

    const workers = (cluster.nodes?.filter(node => node.type === 'worker')?.map(node => node.toJSON()) || []) as TemplateContext['workers']

    return {
      cluster_id: cluster.id,
      cluster_name: cluster.subdomainIdentifier,
      cluster_talos_version: cluster.talosVersion || talosVersion,
      cluster_region: cluster.location,
      network_zone: this.getNetworkZoneFromLocation(cluster.location),
      location: cluster.location,
      s3_region: env.get('S3_REGION'),
      s3_bucket: env.get('S3_BUCKET'),
      control_planes: controlPlanes,
      cluster_talos_image: cluster.providerImageId as string,
      cluster_ssh_key_id: cluster.sshKey?.providerId as string,
      workers: workers,
      cluster_network_id: cluster.providerNetworkId as string,
      public_key: publicKey,
      control_planes_volume_size: cluster.controlPlanesVolumeSize,
      workers_volume_size: cluster.workersVolumeSize,
      cluster_load_balancer_domain: `kube.${cluster.subdomainIdentifier}`
    }
  }

  /**
   * Maps cluster location to appropriate network zone for cloud provider
   */
  private getNetworkZoneFromLocation(location: string): string {
    const locationToZoneMap: Record<string, string> = {
      'nbg1': 'eu-central',
      'fsn1': 'eu-central',
      'hel1': 'eu-central',
      'ash': 'us-east',
      'hil': 'us-west'
    }

    return locationToZoneMap[location] || 'eu-central'
  }
}
