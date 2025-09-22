import { RedisStream } from '#utils/redis_stream'
import { RedisStreamConfig } from '#services/redis/redis_stream_config'
import { ClusterLogsService, ClusterLogEntry } from '#services/redis/cluster_logs_service'
import { ChildProcess } from '#utils/child_process'
import app from '@adonisjs/core/services/app'
import env from '#start/env'
import { join } from 'node:path'
import logger from '@adonisjs/core/services/logger'
import { TerraformExecutorContract } from '#contracts/terraform_executor'
import { writeFile } from 'node:fs/promises'
import { type Subprocess } from 'execa'

export type TerraformCommand = 'init' | 'apply' | 'plan' | 'destroy'
export type TerraformStage =
  | 'network'
  | 'ssh-keys'
  | 'load-balancers'
  | 'servers'
  | 'volumes'
  | 'talos-image'
  | 'dns'
  | 'talos'
  | 'kubernetes-config'
  | 'kubernetes-boot'
  | 'kubernetes-byoc'

export interface TerraformExecutionOptions {
  autoApprove?: boolean
  jsonOutput?: boolean
  additionalArgs?: string[]
  storePlanOutput?: boolean
}

export interface TerraformExecutionResult {
  success: boolean
  exitCode: number
  streamName: string
  error?: string
}

/**
 * Interface for terraform output result with proper typing
 */
export interface TerraformOutputResult {
  stdout: string
  stderr: string
}

export type TerraformLogCallback = (logType: string, message: string, timestamp: string) => void
export type TerraformOutputCallback = (outputs: Record<string, unknown>) => void

export interface TerraformPlanData {
  format_version: string
  terraform_version: string
  variables?: Record<string, unknown>
  planned_values?: {
    outputs?: Record<string, { sensitive: boolean }>
    root_module?: {
      resources?: TerraformPlanResource[]
    }
  }
  resource_changes?: TerraformResourceChange[]
  output_changes?: Record<string, unknown>
  prior_state?: {
    values?: {
      root_module?: {
        resources?: TerraformPlanResource[]
      }
    }
  }
  configuration?: {
    provider_config?: Record<string, TerraformProviderConfig>
    root_module?: {
      resources?: TerraformConfigResource[]
      variables?: Record<string, { description?: string; sensitive?: boolean }>
    }
  }
  timestamp?: string
  applyable?: boolean
  complete?: boolean
  errored?: boolean
}

export interface TerraformPlanResource {
  address: string
  mode: 'managed' | 'data'
  type: string
  name: string
  index?: number
  provider_name: string
  schema_version: number
  values: Record<string, unknown>
  sensitive_values?: Record<string, unknown>
}

export interface TerraformResourceChange {
  address: string
  mode: 'managed' | 'data'
  type: string
  name: string
  index?: number
  provider_name: string
  change: {
    actions: string[]
    before: unknown
    after: Record<string, unknown>
    after_unknown?: Record<string, unknown>
    before_sensitive?: boolean | Record<string, unknown>
    after_sensitive?: boolean | Record<string, unknown>
  }
}

export interface TerraformProviderConfig {
  name: string
  full_name: string
  version_constraint?: string
  expressions?: Record<string, unknown>
}

export interface TerraformConfigResource {
  address: string
  mode: 'managed' | 'data'
  type: string
  name: string
  provider_config_key: string
  expressions: Record<string, { constant_value?: unknown; references?: string[] }>
  schema_version: number
  count_expression?: { references: string[] }
}

export interface DigitalOceanCustomImageResource {
  distribution: string
  name: string
  regions: string[]
  url: string
  description?: string | null
  tags?: string[] | null
  timeouts?: unknown | null
}

export interface DigitalOceanImagesFilter {
  key: string
  values: string[]
  match_by?: string
  all?: boolean
}

/**
 * Handles Terraform command execution with Redis stream logging
 *
 * Usage:
 * ```typescript
 * const executor = new TerraformExecutor('cluster-123', 'network')
 *
 * await executor
 *   .onLog((type, message) => report(`${type}: ${message}`))
 *   .onComplete((result) => report('Done:', result))
 *   .onError((error) => report('Failed:', error))
 *   .onOutput((outputs) => report('Terraform outputs:', outputs))
 *   .init()
 *
 * await executor.apply({ autoApprove: true })
 * ```
 */
export class TerraformExecutor extends TerraformExecutorContract {
  private streamName: string
  private terraformDir: string
  private _vars: Record<string, string | number | boolean> = {}

  constructor(clusterId: string, stage: TerraformStage) {
    super(clusterId, stage)
    this.streamName = RedisStreamConfig.getClusterStream(clusterId)
    this.terraformDir = join(app.makePath('storage'), `terraform/clusters/${clusterId}/${stage}`)
  }

  /**
   * Set Terraform variables
   */
  vars(variables: Record<string, string | number | boolean>): this {
    this._vars = { ...this._vars, ...variables }

    return this
  }

  /**
   * Initialize the Terraform stream
   */
  async initializeStream(): Promise<void> {
    await new RedisStream()
      .stream(this.streamName)
      .fields({
        event: 'terraform_stream_initialized',
        cluster_id: this.clusterId,
        stage: this.stage,
        timestamp: new Date().toISOString(),
        message: `Terraform execution stream initialized for ${this.stage} stage`,
      })
      .onError((error) => {
        logger.error('Failed to initialize terraform stream:', error)
      })
      .add()
  }

  /**
   * Execute terraform init
   */
  async init(options: TerraformExecutionOptions = {}): Promise<void> {
    await this.initializeStream()

    const args = ['init']

    args.push(...(options.additionalArgs || []))

    await this.executeCommand('init', args)
  }

  /**
   * Execute terraform plan
   */
  async plan(options: TerraformExecutionOptions = {}): Promise<void> {
    const args = ['plan']

    if (options.jsonOutput) {
      args.push('-json')
    }

    // Save plan to file if we need to store output
    if (options.storePlanOutput) {
      args.push('-out=terraform-plan.tfplan')
    }

    if (options.additionalArgs) {
      args.push(...options.additionalArgs)
    }

    await this.executeCommand('plan', args)

    // Store plan output if requested
    if (options.storePlanOutput) {
      await this.storePlanOutput()
    }
  }

  /**
   * Execute terraform apply
   */
  async apply(options: TerraformExecutionOptions = {}): Promise<Subprocess> {
    const args = ['apply']

    if (options.autoApprove) {
      args.push('-auto-approve')
    }

    if (options.jsonOutput) {
      args.push('-json')
    }

    if (options.additionalArgs) {
      args.push(...options.additionalArgs)
    }

    return Promise.resolve(this.executeCommand('apply', args))
  }

  /**
   * Execute terraform destroy
   */
  async destroy(options: TerraformExecutionOptions = {}): Promise<Subprocess> {
    const args = ['destroy']

    if (options.autoApprove) {
      args.push('-auto-approve')
    }

    if (options.additionalArgs) {
      args.push(...options.additionalArgs)
    }

    return Promise.resolve(this.executeCommand('destroy', args))
  }

  /**
   * Get the stream name for this executor
   */
  getStreamName(): string {
    return this.streamName
  }

  /**
   * Get the Terraform working directory
   */
  getTerraformDirectory(): string {
    return this.terraformDir
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
   * Get environment variables for Terraform execution
   */
  private getTerraformEnvironment(): Record<string, string> {
    const terraformEnv: Record<string, string> = {}

    terraformEnv.AWS_ACCESS_KEY_ID = env.get('S3_ACCESS_KEY') as string
    terraformEnv.AWS_SECRET_ACCESS_KEY = env.get('S3_ACCESS_SECRET') as string

    for (const [key, value] of Object.entries(this._vars)) {
      terraformEnv[`TF_VAR_${key}`] = String(value)
    }

    return terraformEnv
  }

  /**
   * Get terraform output and trigger callback
   */
  public async output(): Promise<TerraformOutputResult> {
    const [result, error] = await new ChildProcess()
      .command('terraform')
      .args(['output', '-json'])
      .cwd(this.terraformDir)
      .env(this.getTerraformEnvironment())
      .executeAsync()

    if (error || !result) {
      logger.error('Failed to get terraform output:', error?.message)
      throw error || new Error('Failed to get terraform output')
    }

    logger.info('Terraform output fetched successfully')
    return {
      stdout: result.stdout,
      stderr: result.stderr,
    }
  }

  /**
   * Execute a terraform command with stream logging
   */
  private executeCommand(command: TerraformCommand, args: string[]): Subprocess {
    this.logToStream(
      'command_start',
      `Starting terraform ${command} with args: ${args.join(' ')}`
    )

    return new ChildProcess()
      .command('terraform')
      .args(args)
      .cwd(this.terraformDir)
      .env(this.getTerraformEnvironment())
      .onStdout(async (data) => {
        await this.logToStream(`${command}_stdout`, data)

        logger.info(`${command}_stdout: ${data.substring(0, 500)}...`)
      })
      .onStderr(async (data) => {
        await this.logToStream(`${command}_stderr`, data)

        logger.error(`${command}_stderr: ${data.substring(0, 1000)}...`)
      })
      .onClose(async (code) => {
        const message = `Terraform ${command} exited with code: ${code}`

        await this.logToStream(`${command}_close`, message)

        logger.info(`${command}_close: ${message.substring(0, 500)}...`)
      })
      .onError(async (error) => {
        await this.logToStream(`${command}_error`, error.message)

        logger.error(`${command}_error: ${error.message}...`)
      })
      .execute()
  }

  /**
   * Store the terraform plan output to a file in the terraform directory
   */
  private async storePlanOutput(): Promise<void> {
    try {
      const [result, error] = await new ChildProcess()
        .command('terraform')
        .args(['show', '-json', 'terraform-plan.tfplan'])
        .cwd(this.terraformDir)
        .env(this.getTerraformEnvironment())
        .executeAsync()

      if (error || !result) {
        throw error || new Error('Failed to execute terraform show command')
      }

      if (!result.stdout || typeof result.stdout !== 'string') {
        throw new Error('No stdout received from terraform show command')
      }

      const planData = JSON.parse(result.stdout)
      const sanitizedPlan = this.sanitizePlanOutput(planData)

      const planOutputFile = join(this.terraformDir, 'terraform-plan-output.json')
      await writeFile(planOutputFile, JSON.stringify(sanitizedPlan, null, 2))

      logger.info(`Plan output stored to: ${planOutputFile}`)
    } catch (error) {
      logger.error('Failed to store plan output:', error)
    }
  }

  /**
   * Remove sensitive data from terraform plan output
   */
  private sanitizePlanOutput(planData: TerraformPlanData): TerraformPlanData {
    const sanitized = { ...planData }

    if (sanitized.variables) {
      delete sanitized.variables
    }

    return sanitized
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
      logger.error('Failed to log to stream:', error)
    }
  }
}
