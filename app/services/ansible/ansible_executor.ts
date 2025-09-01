import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'
import { ClusterLogsService, ClusterLogEntry } from '#services/redis/cluster_logs_service'
import { ChildProcess } from '#utils/child_process'
import app from '@adonisjs/core/services/app'
import edge from 'edge.js'
import { join } from 'node:path'
import { mkdir, cp, writeFile, access, rm, readFile } from 'node:fs/promises'
import { constants } from 'node:fs'
import logger from '@adonisjs/core/services/logger'
import type Cluster from '#models/cluster'

export type AnsibleCommand =
  | 'clone'
  | 'checkout'
  | 'copy'
  | 'copy-script'
  | 'venv-init'
  | 'dependencies'
  | 'playbook'
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

export interface KubeSprayTemplateData {
  subdomain_identifier: string
  control_planes: Array<{
    public_ipv4_address: string
    private_ipv4_address: string
  }>
  workers: Array<{
    public_ipv4_address: string
    private_ipv4_address: string
  }>
  load_balancer_public_ipv4_address: string
  load_balancer_private_ipv4_address: string
  load_balancer_subdomain_identifier: string
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
  private clusterDir: string
  private venvPath: string
  private cluster: Cluster | null = null

  constructor(
    protected clusterId: string,
    protected stage: AnsibleStage
  ) {
    this.streamName = RedisStreamConfig.getClusterStream(clusterId)
    this.clusterDir = join(app.makePath('storage'), `ansible/clusters/${clusterId}`)
    this.venvPath = join(this.clusterDir, 'venv')
  }

  /**
   * Initialize the Ansible environment for cluster provisioning
   */
  async init(cluster: Cluster): Promise<void> {
    this.cluster = cluster
    await this.initializeStream()
    await this.setupClusterDirectory()
    await this.cloneKubespray()
    await this.checkoutKubesprayVersion()
    await this.copyKibashipInventory()
    await this.copyHardeningConfig()
    await this.copyPlaybookScript()
    await this.compileEdgeTemplates(cluster)
    await this.initializePythonEnvironment()
    await this.installDependencies()
  }

  /**
   * Execute the main Kubernetes provisioning playbook
   */
  async executePlaybook(): Promise<void> {
    const playbookScriptPath = join(this.clusterDir, 'playbook.sh')
    const inventory = 'inventory/kibaship/inventory.ini'
    const playbook = 'cluster.yml'
    const hardeningConfig = 'hardening.yaml'
    const varsConfig = 'vars.yaml'

    const args = [
      '-i',
      inventory,
      playbook,
      '-e',
      `@${hardeningConfig}`,
      '-e',
      `@${varsConfig}`,
      '-v',
    ]

    await this.executePlaybookScript(playbookScriptPath, args)
  }

  /**
   * Get the stream name for this executor
   */
  getStreamName(): string {
    return this.streamName
  }

  /**
   * Get the cluster working directory
   */
  getClusterDirectory(): string {
    return this.clusterDir
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
   * Setup cluster directory and remove existing if present
   */
  private async setupClusterDirectory(): Promise<void> {
    await this.logToStream('setup_start', 'Setting up cluster directory')

    try {
      // Check if cluster directory already exists and delete it
      try {
        await access(this.clusterDir, constants.F_OK)
        await this.logToStream('cleanup_start', 'Existing cluster directory found, removing it')
        await rm(this.clusterDir, { recursive: true, force: true })
        await this.logToStream('cleanup_success', 'Existing cluster directory removed successfully')
      } catch {
        await this.logToStream('cleanup_skip', 'No existing cluster directory found')
      }

      // Create the cluster directory
      await mkdir(this.clusterDir, { recursive: true })
      await this.logToStream('setup_success', 'Cluster directory created successfully')
    } catch (error) {
      const errorMessage = `Failed to setup cluster directory: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('setup_error', errorMessage)
      throw new Error(errorMessage)
    }
  }

  /**
   * Clone kubespray repository
   */
  private async cloneKubespray(): Promise<void> {
    await this.logToStream('clone_start', 'Cloning kubespray repository')

    const args = ['clone', 'https://github.com/kubernetes-sigs/kubespray.git', '.']

    await this.executeCommand('clone', 'git', args)
    await this.logToStream('clone_success', 'Kubespray repository cloned successfully')
  }

  /**
   * Checkout kubespray to specific version
   */
  private async checkoutKubesprayVersion(): Promise<void> {
    await this.logToStream('checkout_start', 'Checking out kubespray version v2.28.1')

    const args = ['checkout', 'v2.28.1']

    await this.executeCommand('checkout', 'git', args, undefined, this.clusterDir)
    await this.logToStream('checkout_success', 'Kubespray checked out to v2.28.1 successfully')
  }

  /**
   * Copy kibaship inventory structure
   */
  private async copyKibashipInventory(): Promise<void> {
    await this.logToStream('copy_inventory_start', 'Copying kibaship inventory structure')

    try {
      const sourceInventoryDir = join(app.makePath(), 'kubernetes/kibaship')
      const targetInventoryDir = join(this.clusterDir, 'inventory/kibaship')

      await mkdir(targetInventoryDir, { recursive: true })

      await cp(sourceInventoryDir, targetInventoryDir, {
        recursive: true,
        force: true,
      })

      await this.logToStream(
        'copy_inventory_success',
        'Kibaship inventory structure copied successfully'
      )
    } catch (error) {
      const errorMessage = `Failed to copy kibaship inventory: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('copy_inventory_error', errorMessage)
      throw new Error(errorMessage)
    }
  }

  /**
   * Copy hardening configuration
   */
  private async copyHardeningConfig(): Promise<void> {
    await this.logToStream('copy_hardening_start', 'Copying hardening configuration')

    try {
      const sourceHardeningPath = join(app.makePath(), 'kubernetes/hardening.yaml')
      const targetHardeningPath = join(this.clusterDir, 'hardening.yaml')

      await cp(sourceHardeningPath, targetHardeningPath)

      await this.logToStream(
        'copy_hardening_success',
        'Hardening configuration copied successfully'
      )
    } catch (error) {
      const errorMessage = `Failed to copy hardening config: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('copy_hardening_error', errorMessage)
      throw new Error(errorMessage)
    }
  }

  /**
   * Copy playbook script
   */
  private async copyPlaybookScript(): Promise<void> {
    await this.logToStream('copy_playbook_start', 'Copying playbook script')

    try {
      const sourcePlaybookPath = join(app.makePath(), 'kubernetes/playbook.sh')
      const targetPlaybookPath = join(this.clusterDir, 'playbook.sh')

      await cp(sourcePlaybookPath, targetPlaybookPath)

      await this.logToStream('copy_playbook_success', 'Playbook script copied successfully')
    } catch (error) {
      const errorMessage = `Failed to copy playbook script: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('copy_playbook_error', errorMessage)
      throw new Error(errorMessage)
    }
  }

  /**
   * Compile Edge templates with cluster data
   */
  private async compileEdgeTemplates(cluster: Cluster): Promise<void> {
    await this.logToStream('compile_start', 'Compiling Edge templates with cluster data')

    try {
      const clusterLoadBalancer = cluster.loadBalancers.find((lb) => lb.type === 'cluster')

      const templateData: KubeSprayTemplateData = {
        subdomain_identifier: cluster.subdomainIdentifier,
        control_planes: cluster.nodes
          .filter((node) => node.type === 'master')
          .map((node) => ({
            public_ipv4_address: node.ipv4Address as string,
            private_ipv4_address: node.privateIpv4Address as string,
          })),
        workers: cluster.nodes
          .filter((node) => node.type === 'worker')
          .map((node) => ({
            public_ipv4_address: node.ipv4Address as string,
            private_ipv4_address: node.privateIpv4Address as string,
          })),
        load_balancer_public_ipv4_address: clusterLoadBalancer?.publicIpv4Address as string,
        load_balancer_private_ipv4_address: clusterLoadBalancer?.privateIpv4Address as string,
        load_balancer_subdomain_identifier: `kube.${cluster.subdomainIdentifier}`,
      }

      await this.compileTemplate(
        'inventory/kibaship/inventory.ini.edge',
        'inventory/kibaship/inventory.ini',
        templateData
      )

      const varsTemplatePath = join(app.makePath(), 'kubernetes/vars.yaml.edge')
      const varsOutputPath = join(this.clusterDir, 'vars.yaml')
      const varsTemplateContent = await readFile(varsTemplatePath, 'utf8')
      const varsCompiledContent = await edge.renderRaw(varsTemplateContent, templateData)
      await writeFile(varsOutputPath, varsCompiledContent, 'utf8')

      await this.logToStream('compile_success', 'Edge templates compiled successfully')
    } catch (error) {
      const errorMessage = `Failed to compile templates: ${error instanceof Error ? error.message : 'Unknown error'}`
      await this.logToStream('compile_error', errorMessage)
      throw new Error(errorMessage)
    }
  }

  /**
   * Compile a single Edge template
   */
  private async compileTemplate(
    templatePath: string,
    outputPath: string,
    data: KubeSprayTemplateData
  ): Promise<void> {
    const fullTemplatePath = join(this.clusterDir, templatePath)
    const fullOutputPath = join(this.clusterDir, outputPath)

    const templateContent = await readFile(fullTemplatePath, 'utf8')
    const compiledContent = await edge.renderRaw(templateContent, data)

    await writeFile(fullOutputPath, compiledContent, 'utf8')
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

    const requirementsPath = join(this.clusterDir, 'requirements.txt')
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

    const args = ['install', '-r', requirementsPath]

    await this.executeCommand('dependencies', pipPath, args)
  }

  /**
   * Execute ansible-playbook command via bash script
   */
  private async executePlaybookScript(scriptPath: string, args: string[]): Promise<void> {
    await this.logToStream(
      'playbook_start',
      `Starting ansible-playbook via bash script with args: ${args.join(' ')}`
    )

    await this.executeCommand('playbook', scriptPath, args, this.cluster || undefined)
  }

  /**
   * Execute a command with stream logging
   */
  private async executeCommand(
    command: AnsibleCommand,
    executable: string,
    args: string[],
    cluster?: Cluster,
    workingDir?: string
  ): Promise<void> {
    const cwd = workingDir || this.clusterDir

    await new ChildProcess()
      .command(executable)
      .args(args)
      .cwd(cwd)
      .env(this.getAnsibleEnvironment(cluster))
      .onStdout(async (data) => {
        await this.logToStream(`${command}_stdout`, data)
        logger.info(`${command}_stdout: ${data.substring(0, 300)}...`)
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

      // Use kubespray ansible.cfg
      ANSIBLE_CONFIG: join(this.clusterDir, 'ansible.cfg'),

      // Python path for virtual environment
      PATH: `${join(this.venvPath, 'bin')}:${process.env.PATH || ''}`,

      // Ansible playbook binary path for bash script
      ANSIBLE_PLAYBOOK_BIN: join(this.venvPath, 'bin', 'ansible-playbook'),

      // SSH private key from cluster (for bash script)
      ...(cluster?.sshKey?.privateKey
        ? { KIBASHIP_SSH_PRIVATE_KEY: cluster.sshKey.privateKey }
        : {}),

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
