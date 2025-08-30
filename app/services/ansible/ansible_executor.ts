import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'
import { ClusterLogsService, ClusterLogEntry } from '#services/redis/cluster_logs_service'
import { ChildProcess } from '#utils/child_process'
import app from '@adonisjs/core/services/app'
import edge from 'edge.js'
import { join } from 'node:path'
import { mkdir, cp, writeFile, access, rm } from 'node:fs/promises'
import { constants } from 'node:fs'
import logger from '@adonisjs/core/services/logger'
import type Cluster from '#models/cluster'

export type AnsibleCommand = 'venv-init' | 'dependencies' | 'playbook'
export type AnsibleStage = 'kubernetes'

export interface AnsibleExecutionOptions {
  playbook?: string
  inventory?: string
  additionalArgs?: string[]
  verbose?: boolean
}

export interface AnsibleExecutionResult {
  success: boolean
  exitCode: number
  streamName: string
  error?: string
}

export interface AnsibleInventoryData {
  name: string
  sshUser: string
  sshKeyPath: string
  controlPlanes: Array<{
    name: string
    publicIP: string
    privateIP: string
  }>
  workers: Array<{
    name: string
    publicIP: string
    privateIP: string
  }>
  loadBalancers: {
    kube: {
      domain: string
      port: number
      publicIP: string
      privateIP?: string
    }
    ingress?: {
      domain: string
      publicIP: string
      privateIP?: string
    }
  }
  network: {
    serviceSubnet: string
    podSubnet: string
    dnsDomain: string
  }
  cloudProvider?: {
    name: string
    region: string
    projectId: string
  }
  provisionedAt: string
}

/**
 * Handles Ansible playbook execution with Redis stream logging
 *
 * Usage:
 * ```typescript
 * const executor = new AnsibleExecutor('cluster-123', 'kubernetes')
 *
 * await executor
 *   .init(cluster)
 *   .then(() => executor.executePlaybook())
 * ```
 */
export class AnsibleExecutor {
  private streamName: string
  private ansibleDir: string
  private sourceAnsibleDir: string
  private venvPath: string
  private inventoryPath: string
  private cluster: Cluster | null = null

  constructor(
    protected clusterId: string,
    protected stage: AnsibleStage
  ) {
    this.streamName = RedisStreamConfig.getClusterStream(clusterId)
    this.ansibleDir = join(app.makePath('storage'), `ansible/clusters/${clusterId}/${stage}`)
    this.sourceAnsibleDir = join(app.makePath(), 'ansible')
    this.venvPath = join(this.ansibleDir, 'venv')
    this.inventoryPath = join(this.ansibleDir, 'inventory/kibaship/inventory.ini')
  }

  /**
   * Initialize the Ansible environment for cluster provisioning
   */
  async init(cluster: Cluster): Promise<void> {
    this.cluster = cluster
    await this.initializeStream()
    await this.copyAnsibleFiles()
    await this.generateInventory(cluster)
    await this.initializePythonEnvironment()
    await this.installDependencies()
  }

  /**
   * Execute the main Kubernetes provisioning playbook
   */
  async executePlaybook(): Promise<void> {
    const playbook = 'kubernetes.yml'
    const inventory = 'inventory/kibaship/inventory.ini'

    const args = [
      '-i',
      inventory,
      playbook,
      '-v', // Always use verbose for provisioning visibility
    ]

    await this.executeAnsiblePlaybook('playbook', args)
  }

  /**
   * Get the stream name for this executor
   */
  getStreamName(): string {
    return this.streamName
  }

  /**
   * Get the Ansible working directory
   */
  getAnsibleDirectory(): string {
    return this.ansibleDir
  }

  /**
   * Get stream length (number of log entries)
   */
  async getLogCount(): Promise<number> {
    try {
      return await new RedisStream().stream(this.streamName).length()
    } catch (error) {
      return 0
    }
  }

  /**
   * Read historical logs from the stream
   */
  async readLogs(fromId: string = '0', count?: number): Promise<ClusterLogEntry[]> {
    try {
      const messages = await new RedisStream()
        .stream(this.streamName)
        .from(fromId)
        .count(count || 100)
        .read()

      return ClusterLogsService.parseStreamMessages(messages, this.clusterId)
    } catch (error) {
      logger.error('Failed to read logs from stream:', error)
      return []
    }
  }

  /**
   * Initialize the Ansible stream
   */
  private async initializeStream(): Promise<void> {
    await new RedisStream()
      .stream(this.streamName)
      .fields({
        event: 'ansible_stream_initialized',
        cluster_id: this.clusterId,
        stage: this.stage,
        timestamp: new Date().toISOString(),
        message: `Ansible execution stream initialized for ${this.stage} stage`,
      })
      .onError((error) => {
        logger.error('Failed to initialize ansible stream:', error)
      })
      .add()
  }

  /**
   * Copy the entire ansible directory to cluster storage
   */
  private async copyAnsibleFiles(): Promise<void> {
    await this.logToStream('copy_start', 'Copying Ansible files to cluster directory')

    try {
      // Check if ansible storage directory already exists for cluster and delete it
      try {
        await access(this.ansibleDir, constants.F_OK)
        await this.logToStream('cleanup_start', 'Existing Ansible directory found, removing it')
        await rm(this.ansibleDir, { recursive: true, force: true })
        await this.logToStream('cleanup_success', 'Existing Ansible directory removed successfully')
      } catch {
        // Directory doesn't exist, which is fine
        await this.logToStream(
          'cleanup_skip',
          'No existing Ansible directory found, proceeding with copy'
        )
      }

      // Create the cluster ansible directory
      await mkdir(this.ansibleDir, { recursive: true })

      // Copy entire ansible directory
      await cp(this.sourceAnsibleDir, this.ansibleDir, {
        recursive: true,
        force: true,
      })

      await this.logToStream('copy_success', 'Ansible files copied successfully')

      // After copying, check if venv directory exists inside copied directory and delete it
      const copiedVenvPath = join(this.ansibleDir, 'venv')
      try {
        await access(copiedVenvPath, constants.F_OK)
        await this.logToStream(
          'venv_cleanup_start',
          'Found venv directory in copied files, removing it'
        )
        await rm(copiedVenvPath, { recursive: true, force: true })
        await this.logToStream('venv_cleanup_success', 'Venv directory removed from copied files')
      } catch {
        // venv directory doesn't exist in copied files, which is fine
        await this.logToStream('venv_cleanup_skip', 'No venv directory found in copied files')
      }
    } catch (error) {
      const errorMessage = `Failed to copy Ansible files: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('copy_error', errorMessage)
      throw new Error(errorMessage)
    }
  }


  /**
   * Generate inventory.ini from the Edge template
   */
  private async generateInventory(cluster: Cluster): Promise<void> {
    await this.logToStream(
      'inventory_start',
      'Generating Ansible inventory from cluster configuration'
    )

    try {
      // Cluster is already loaded with all relationships

      // Prepare inventory data
      const inventoryData: AnsibleInventoryData = {
        name: cluster.subdomainIdentifier,
        sshUser: 'root', // Fixed user for Kibaship clusters
        sshKeyPath: 'ENV_SSH_KEY', // SSH key will be passed via environment variable
        controlPlanes: cluster.nodes
          .filter((node) => node.type === 'master')
          .map((node, index) => ({
            name: `k8s-cp-${index + 1}`,
            publicIP: node.ipv4Address!,
            privateIP: node.privateIpv4Address!,
          })),
        workers: cluster.nodes
          .filter((node) => node.type === 'worker')
          .map((node, index) => ({
            name: `k8s-worker-${index + 1}`,
            publicIP: node.ipv4Address!,
            privateIP: node.privateIpv4Address!,
          })),
        loadBalancers: {
          kube: {
            domain: `kube.${cluster.subdomainIdentifier}`,
            port: 6443,
            publicIP: cluster.loadBalancers.find((lb) => lb.type === 'cluster')?.publicIpv4Address!,
            privateIP:
              cluster.loadBalancers.find((lb) => lb.type === 'cluster')?.privateIpv4Address ||
              undefined,
          },
          ingress: cluster.loadBalancers.find((lb) => lb.type === 'ingress')
            ? {
                domain: cluster.subdomainIdentifier,
                publicIP: cluster.loadBalancers.find((lb) => lb.type === 'ingress')
                  ?.publicIpv4Address!,
                privateIP:
                  cluster.loadBalancers.find((lb) => lb.type === 'ingress')?.privateIpv4Address ||
                  undefined,
              }
            : undefined,
        },
        network: {
          serviceSubnet: '10.96.0.0/12',
          podSubnet: '10.244.0.0/16',
          dnsDomain: 'cluster.local',
        },
        cloudProvider: cluster.cloudProvider
          ? {
              name: cluster.cloudProvider.type,
              region: cluster.location,
              projectId: cluster.cloudProvider.id,
            }
          : undefined,
        provisionedAt: new Date().toISOString(),
      }

      // Load and render the Edge template
      const templatePath = join(this.ansibleDir, 'inventory/kibaship/inventory.ini.edge')
      const inventoryContent = await edge.render(templatePath, {
        cluster: inventoryData,
        kibashipVersion: '1.0.0',
      })

      // Write the generated inventory
      await writeFile(this.inventoryPath, inventoryContent, 'utf8')

      await this.logToStream(
        'inventory_success',
        `Ansible inventory generated successfully: ${cluster.nodes.length} nodes, ${cluster.loadBalancers.length} load balancers`
      )
    } catch (error) {
      const errorMessage = `Failed to generate inventory: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('inventory_error', errorMessage)
      throw new Error(errorMessage)
    }
  }

  /**
   * Initialize Python virtual environment
   */
  private async initializePythonEnvironment(): Promise<void> {
    await this.logToStream('venv_start', 'Initializing Python virtual environment')

    // Check if venv already exists
    try {
      await access(this.venvPath, constants.F_OK)
      await this.logToStream(
        'venv_exists',
        'Python virtual environment already exists, skipping creation'
      )
      return
    } catch {
      // venv doesn't exist, create it
    }

    const args = ['-m', 'venv', 'venv']

    await this.executeCommand('venv-init', 'python3', args)
  }

  /**
   * Install Ansible and Python dependencies
   */
  private async installDependencies(): Promise<void> {
    await this.logToStream('deps_start', 'Installing Ansible dependencies')

    const requirementsPath = join(this.ansibleDir, 'requirements.txt')
    const pipPath = join(this.venvPath, 'bin', 'pip')

    // Check if requirements.txt exists
    try {
      await access(requirementsPath, constants.F_OK)
    } catch {
      await this.logToStream(
        'deps_warning',
        'No requirements.txt found, skipping dependency installation'
      )
      return
    }

    const args = ['install', '-r', 'requirements.txt']

    await this.executeCommand('dependencies', pipPath, args)
  }

  /**
   * Execute ansible-playbook command
   */
  private async executeAnsiblePlaybook(command: AnsibleCommand, args: string[]): Promise<void> {
    const ansiblePlaybookPath = join(this.venvPath, 'bin', 'ansible-playbook')

    await this.logToStream(
      'playbook_start',
      `Starting ansible-playbook with args: ${args.join(' ')}`
    )

    await this.executeCommand(command, ansiblePlaybookPath, args, this.cluster || undefined)
  }

  /**
   * Execute a command with stream logging
   */
  private async executeCommand(
    command: AnsibleCommand,
    executable: string,
    args: string[],
    cluster?: Cluster
  ): Promise<void> {
    await new ChildProcess()
      .command(executable)
      .args(args)
      .cwd(this.ansibleDir)
      .env(this.getAnsibleEnvironment(cluster))
      .onStdout(async (data) => {
        await this.logToStream(`${command}_stdout`, data)
        logger.info(`${command}_stdout: ${data.substring(0, 100)}...`)
      })
      .onStderr(async (data) => {
        await this.logToStream(`${command}_stderr`, data)
        logger.error(`${command}_stderr: ${data.substring(0, 100)}...`)
      })
      .onClose(async (code) => {
        const message = `${executable} ${command} exited with code: ${code}`
        await this.logToStream(`${command}_close`, message)
        logger.info(`${command}_close: ${message}`)
      })
      .onError(async (error) => {
        await this.logToStream(`${command}_error`, error.message)
        logger.error(`${command}_error: ${error.message}`)
      })
      .execute()
  }

  /**
   * Get environment variables for Ansible execution
   */
  private getAnsibleEnvironment(cluster?: Cluster): Record<string, string> {
    const ansibleEnv: Record<string, string> = {
      // Disable host key checking for automated provisioning
      ANSIBLE_HOST_KEY_CHECKING: 'false',

      // Use our ansible.cfg
      ANSIBLE_CONFIG: join(this.ansibleDir, 'ansible.cfg'),

      // Python path for virtual environment
      PATH: `${join(this.venvPath, 'bin')}:${process.env.PATH || ''}`,

      // SSH private key from cluster
      ...(cluster?.sshKey?.privateKey ? { KIBASHIP_SSH_PRIVATE_KEY: cluster.sshKey.privateKey } : {}),

      // Inherit system environment
      ...process.env,
    }

    return ansibleEnv
  }

  /**
   * Log a message to the Redis stream
   */
  private async logToStream(logType: string, message: string): Promise<void> {
    try {
      await new RedisStream()
        .stream(this.streamName)
        .fields({
          type: logType,
          message: message.trim(),
          timestamp: new Date().toISOString(),
          cluster_id: this.clusterId,
          stage: this.stage,
        })
        .add()
    } catch (error) {
      console.error('Failed to log to stream:', error)
    }
  }
}
