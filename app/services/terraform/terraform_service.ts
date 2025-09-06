import edge from 'edge.js'
import app from '@adonisjs/core/services/app'
import drive from '@adonisjs/drive/services/main'
import env from '#start/env'
import { join } from 'node:path'
import { talosFactoryHash, talosVersion } from '#config/app'
import Cluster from '#models/cluster'
import { ModelObject } from '@adonisjs/lucid/types/model'

export enum TerraformTemplate {
  TALOS_IMAGE = 'talos-image.tf',
  NETWORK = 'network.tf',
  SSH_KEYS = 'ssh-keys.tf',
  LOAD_BALANCERS = 'load-balancers.tf',
  SERVERS = 'servers.tf',
  VOLUMES = 'volumes.tf',
  KUBERNETES = 'kubernetes.tf',
  KUBERNETES_CONFIG = 'kubernetes-config.tf'
}

export interface TemplateContext {
  cluster_id: string
  cluster_name: string
  cluster_talos_version: string
  cluster_talos_factory_hash: string
  cluster_region: string
  cluster_network_id: string
  cluster_network_ip_range: string
  cluster_private_subnet: string
  cluster_pod_subnet: string
  cluster_service_subnet: string
  cluster_subnet_mask: string
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
    provider_id: string
    ipv4_address: string | null
    primary_disk_name: string | null
    private_network_interface: string | null
    public_network_interface: string | null
  }>
  workers: Array<ModelObject & {
    id: string
    slug: string
    type: string
    provider_id: string
    ipv4_address: string | null
    primary_disk_name: string | null
    private_network_interface: string | null
    public_network_interface: string | null
  }>
  public_key: string
  control_planes_volume_size: number
  workers_volume_size: number
  volumes: Array<{
    id: string
    slug: string
    size: number
    node_provider_id: string
  }>

  cluster_load_balancer_private_ipv4_address: string | null
  cluster_load_balancer_public_ipv4_address: string | null
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

    const controlPlanes = (cluster.nodes?.filter(node => node.type === 'master')?.map(node => ({
      id: node.id,
      slug: node.slug,
      type: node.type,
      provider_id: node.providerId as string,
      ipv4_address: node.ipv4Address,
      private_ipv4_address: node.privateIpv4Address,
      primary_disk_name: node.storages?.[0]?.diskName,
      private_network_interface: node.privateNetworkInterface,
      public_network_interface: node.publicNetworkInterface,
      public_ipv4_gateway: node.publicIpv4Gateway
    })) || [])

    const workers = (cluster.nodes?.filter(node => node.type === 'worker')?.map(node => ({
      id: node.id,
      slug: node.slug,
      type: node.type,
      provider_id: node.providerId as string,
      ipv4_address: node.ipv4Address,
      primary_disk_name: node.storages?.[0]?.diskName,
      private_ipv4_address: node.privateIpv4Address,
      private_network_interface: node.privateNetworkInterface,
      public_network_interface: node.publicNetworkInterface,
            public_ipv4_gateway: node.publicIpv4Gateway
    })) || [])


    const loadBalancer = cluster.loadBalancers.find(lb => lb.type === 'cluster')

    return {
      // =============================================================================
      // GLOBAL VARIABLES - Used by ALL Terraform templates
      // =============================================================================
      
      /** Used by all templates for S3 backend state storage */
      s3_region: env.get('S3_REGION'),         // Backend state in all templates
      s3_bucket: env.get('S3_BUCKET'),         // Backend state in all templates
      
      /** Core cluster identifiers used across multiple templates */
      cluster_id: cluster.id,                  // Used by: servers.tf.edge, load-balancers.tf.edge, tagging
      cluster_name: cluster.subdomainIdentifier, // Used by: all templates for resource naming
      cluster_region: cluster.location,        // Used by: servers.tf.edge, talos-image.tf.edge, volumes.tf.edge
      location: cluster.location,              // Used by: volumes.tf.edge (legacy alias for cluster_region)
      
      // =============================================================================
      // INFRASTRUCTURE VARIABLES - Used by multiple infrastructure templates
      // =============================================================================
      
      /** Network configuration used by servers and load balancers */
      cluster_network_id: cluster.providerNetworkId as string, // Used by: servers.tf.edge
      cluster_network_ip_range: cluster.networkIpRange as string, // Used by: network.tf.edge for VPC IP range
      cluster_private_subnet: cluster.subnetIpRange as string, // Used by: kubernetes.tf.edge for validSubnets
      cluster_pod_subnet: '10.244.0.0/16', // Used by: kubernetes.tf.edge for pod CIDR
      cluster_service_subnet: '10.96.0.0/12', // Used by: kubernetes.tf.edge for service CIDR
      cluster_subnet_mask: '/16',
      network_zone: this.getNetworkZoneFromLocation(cluster.location), // Used by: network.tf.edge
      
      /** SSH and security configuration */
      cluster_ssh_key_id: cluster.sshKey?.providerId as string, // Used by: servers.tf.edge
      public_key: publicKey,                   // Used by: ssh-keys.tf.edge
      
      /** Talos OS configuration */
      cluster_talos_version: cluster.talosVersion || talosVersion, // Used by: talos-image.tf.edge
      cluster_talos_image: cluster.providerImageId as string, // Used by: servers.tf.edge
      cluster_talos_factory_hash: talosFactoryHash, // Used by: talos-image.tf.edge

      // =============================================================================
      // NODE ARRAYS - Used by servers.tf.edge and kubernetes.tf.edge
      // =============================================================================
      
      /** Control plane nodes array with computed properties */
      control_planes: controlPlanes,           // Used by: servers.tf.edge, kubernetes.tf.edge
      
      /** Worker nodes array with computed properties */  
      workers: workers,                        // Used by: servers.tf.edge, kubernetes.tf.edge
      
      // =============================================================================
      // LOAD BALANCER VARIABLES - Used by load-balancers.tf.edge and kubernetes.tf.edge
      // =============================================================================
      
      /** Load balancer domain for Kubernetes API access */
      cluster_load_balancer_private_ipv4_address: loadBalancer?.privateIpv4Address as string,
      cluster_load_balancer_public_ipv4_address: loadBalancer?.publicIpv4Address as string,
      
      // =============================================================================
      // VOLUME-SPECIFIC VARIABLES - Used ONLY by volumes.tf.edge
      // =============================================================================
      
      /** Volume sizes for different node types */
      control_planes_volume_size: cluster.controlPlanesVolumeSize, // Used by: volumes.tf.edge (deprecated)
      workers_volume_size: cluster.workersVolumeSize,            // Used by: volumes.tf.edge (deprecated)
      
      /** Dynamic volumes array with storage metadata */
      volumes: cluster.nodes?.map(node =>     // Used by: volumes.tf.edge ONLY
        node.storages.map(storage => ({
          id: storage.id,                      // Storage record ID
          slug: storage.slug,                  // Unique storage identifier for resource naming
          size: node.type === 'master' ? cluster.controlPlanesVolumeSize : cluster.workersVolumeSize,
          node_provider_id: node.providerId as string // Droplet ID for volume attachment
        }))
      ).flat()
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
