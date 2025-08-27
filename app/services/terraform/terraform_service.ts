import edge from 'edge.js'
import app from '@adonisjs/core/services/app'
import { mkdir, writeFile, readdir, stat, rm } from 'node:fs/promises'
import { join } from 'node:path'
import { tmpdir } from 'node:os'
import Cluster from '#models/cluster'
import { ModelObject } from '@adonisjs/lucid/types/model'

export enum TerraformTemplate {
  NETWORK = 'network.tf',
  SSH_KEYS = 'ssh-keys.tf',
  LOAD_BALANCERS = 'load-balancers.tf',
  SERVERS = 'servers.tf',
  VOLUMES = 'volumes.tf'
}

export interface TemplateContext {
  cluster_name: string
  network_zone: string
  location: string
  hcloud_token: string
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
}

export interface TerraformFile {
  name: string
  content: string
  path: string
}

/**
 * Terraform template service that handles Edge template rendering and file generation.
 *
 * This service manages Terraform template generation for cluster infrastructure:
 * - Renders Edge templates by name for any cloud provider
 * - Initializes cluster directories and generates all templates
 * - Provides cleanup functionality for temporary files
 *
 * All methods follow the $trycatch pattern and return [result, error] tuples
 * for consistent error handling throughout the application.
 */
export class TerraformService {
  private edge: typeof edge
  private baseTempDir: string
  private clusterDirectory: string

  constructor(clusterId: string) {
    this.edge = edge
    this.baseTempDir = join(tmpdir(), 'kibaship-terraform')
    this.clusterDirectory = join(this.baseTempDir, clusterId)
  }

  /**
   * Initializes the service by setting up Edge views and creating cluster directory
   */
  async init() {
    await this.cleanup()

    const terraformViewsPath = app.makePath('resources/views/clusters/terraform')

    this.edge.mount(terraformViewsPath)

    await mkdir(this.clusterDirectory, { recursive: true })
  }

  /**
   * Writes a single Terraform file to a subdirectory with main.tf
   */
  async writeTerraformFile(file: Omit<TerraformFile, 'path'>) {
    const templateName = file.name.replace('.tf', '')
    const templateDir = join(this.clusterDirectory, templateName)

    await mkdir(templateDir, { recursive: true })

    const filePath = join(templateDir, 'main.tf')
    await writeFile(filePath, file.content, 'utf8')

    return filePath
  }

  /**
   * Writes multiple Terraform files to subdirectories with main.tf
   */
  async writeTerraformFiles(files: Omit<TerraformFile, 'path'>[]) {
    const filePaths: TerraformFile[] = []

    for (const file of files) {
      const filePath = await this.writeTerraformFile(file)
      const templateName = file.name.replace('.tf', '')

      filePaths.push({
        name: templateName,
        content: file.content,
        path: filePath
      })
    }

    return filePaths
  }

  /**
   * Checks if a directory exists and is accessible
   */
  async directoryExists(dirPath: string): Promise<boolean> {
    try {
      const stats = await stat(dirPath)
      return stats.isDirectory()
    } catch {
      return false
    }
  }

  /**
   * Lists all files in a directory
   */
  async listFiles(dirPath: string): Promise<string[]> {
    return readdir(dirPath)
  }

  /**
   * Removes the cluster directory and all its contents
   */
  async cleanup() {
    const exists = await this.directoryExists(this.clusterDirectory)

    if (exists) {
      await rm(this.clusterDirectory, { recursive: true, force: true })
    }
  }

  /**
   * Generates a single Terraform template for a cluster and writes it to a file
   * @param cluster - The cluster to generate template for
   * @param templateName - The specific template name to generate
   */
  async generate(cluster: Cluster, templateName: TerraformTemplate): Promise<TerraformFile> {
    await this.init()

    const context = this.buildTemplateContext(cluster)
    const cloudProviderType = cluster.cloudProvider.type

    const content = await this.edge.render(`${cloudProviderType}/${templateName}`, context)
    const terraformFile: Omit<TerraformFile, 'path'> = {
      name: templateName,
      content
    }

    const filePath = await this.writeTerraformFile(terraformFile)
    return {
      name: templateName.replace('.tf', ''),
      content,
      path: filePath
    }
  }



  /**
   * Builds comprehensive template context for all infrastructure templates
   */
  private buildTemplateContext(cluster: Cluster): TemplateContext {
    const hcloudToken = cluster.cloudProvider?.credentials?.token
    if (!hcloudToken) {
      throw new Error('Cloud provider token is required')
    }

    const publicKey = cluster.sshKeys?.[0]?.publicKey
    if (!publicKey) {
      throw new Error('SSH public key is required')
    }

    const controlPlanes = cluster.nodes?.filter(node => node.type === 'master')?.map(node => {
      const nodeData = node.toJSON()
      return {
        id: nodeData.id,
        slug: nodeData.slug,
        type: nodeData.type,
        ...nodeData
      }
    }) || []

    const workers = cluster.nodes?.filter(node => node.type === 'worker')?.map(node => {
      const nodeData = node.toJSON()
      return {
        id: nodeData.id,
        slug: nodeData.slug,
        type: nodeData.type,
        ...nodeData
      }
    }) || []

    return {
      cluster_name: cluster.subdomainIdentifier,
      network_zone: this.getNetworkZoneFromLocation(cluster.location),
      location: cluster.location,
      hcloud_token: hcloudToken,
      control_planes: controlPlanes,
      workers: workers,
      public_key: publicKey,
      control_planes_volume_size: cluster.controlPlanesVolumeSize,
      workers_volume_size: cluster.workersVolumeSize
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
