import { RedisStream } from '#utils/redis_stream'
import { ChildProcess } from '#utils/child_process'
import app from '@adonisjs/core/services/app'
import env from '#start/env'
import { join } from 'node:path'
import logger from '@adonisjs/core/services/logger'

export type TerraformCommand = 'init' | 'apply' | 'plan' | 'destroy'
export type TerraformStage = 'network' | 'servers' | 'volumes' | 'kubernetes'

export interface TerraformExecutionOptions {
  autoApprove?: boolean
  jsonOutput?: boolean
  additionalArgs?: string[]
}

export interface TerraformExecutionResult {
  success: boolean
  exitCode: number
  streamName: string
  error?: string
}

export type TerraformLogCallback = (logType: string, message: string, timestamp: string) => void
export type TerraformOutputCallback = (outputs: Record<string, any>) => void

/**
 * Handles Terraform command execution with Redis stream logging
 *
 * Usage:
 * ```typescript
 * const executor = new TerraformExecutor('cluster-123', 'network')
 *
 * await executor
 *   .onLog((type, message) => console.log(`${type}: ${message}`))
 *   .onComplete((result) => console.log('Done:', result))
 *   .onError((error) => console.error('Failed:', error))
 *   .onOutput((outputs) => console.log('Terraform outputs:', outputs))
 *   .init()
 *
 * await executor.apply({ autoApprove: true })
 * ```
 */
export class TerraformExecutor {
  private streamName = 'terraform:clusters'
  private terraformDir: string
  private _vars: Record<string, string | number | boolean> = {}

  constructor(protected clusterId: string, protected stage: TerraformStage) {
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
        message: `Terraform execution stream initialized for ${this.stage} stage`
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

    args.push(...options.additionalArgs || [])

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

    if (options.additionalArgs) {
      args.push(...options.additionalArgs)
    }

    await this.executeCommand('plan', args)
  }

  /**
   * Execute terraform apply
   */
  async apply(options: TerraformExecutionOptions = {}) {
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

    await this.executeCommand('apply', args)

    return this.getTerraformOutput()
  }

  /**
   * Execute terraform destroy
   */
  async destroy(options: TerraformExecutionOptions = {}) {
    const args = ['destroy']

    if (options.autoApprove) {
      args.push('-auto-approve')
    }

    if (options.additionalArgs) {
      args.push(...options.additionalArgs)
    }

    return this.executeCommand('destroy', args)
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
      return await new RedisStream()
        .stream(this.streamName)
        .length()
    } catch (error) {
      return 0
    }
  }

  /**
   * Read historical logs from the stream
   */
  async readLogs(fromId: string = '0', count?: number): Promise<Array<{
    id: string
    type: string
    message: string
    timestamp: string
    cluster_id: string
  }>> {
    try {
      const messages = await new RedisStream()
        .stream(this.streamName)
        .from(fromId)
        .count(count || 100)
        .read()

      return messages.flatMap(msg =>
        msg.entries.map(entry => ({
          id: entry.id,
          type: entry.fields.type || 'unknown',
          message: entry.fields.message || '',
          timestamp: entry.fields.timestamp || '',
          cluster_id: entry.fields.cluster_id || this.clusterId
        }))
      )
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
  private async getTerraformOutput() {
    return new ChildProcess()
      .command('terraform')
      .args(['output', '-json'])
      .cwd(this.terraformDir)
      .env(this.getTerraformEnvironment())
      .onStdout(async (data) => {
        await this.logToStream('output_stdout', data)
      })
      .onStderr(async (data) => {
        await this.logToStream('output_stderr', data)
      })
      .onClose(async (code) => {
        const message = `Terraform output exited with code: ${code}`
        await this.logToStream('output_close', message)
      })
      .onError(async (error) => {
        await this.logToStream('output_error', error.message)
      })
      .execute()
  }

  /**
   * Execute a terraform command with stream logging
   */
  private async executeCommand(command: TerraformCommand, args: string[]) {
    await this.logToStream('command_start', `Starting terraform ${command} with args: ${args.join(' ')}`)

    return new ChildProcess()
      .command('terraform')
      .args(args)
      .cwd(this.terraformDir)
      .env(this.getTerraformEnvironment())
      .onStdout(async (data) => {
        await this.logToStream(`${command}_stdout`, data)

        logger.info(`${command}_stdout: ${data.substring(0, 100)}...`)
      })
      .onStderr(async (data) => {
        await this.logToStream(`${command}_stderr`, data)

        logger.error(`${command}_stderr: ${data.substring(0, 100)}...`)
      })
      .onClose(async (code) => {
        const message = `Terraform ${command} exited with code: ${code}`

        await this.logToStream(`${command}_close`, message)

        logger.info(`${command}_close: ${message.substring(0, 100)}...`)
      })
      .onError(async (error) => {
        await this.logToStream(`${command}_error`, error.message)

        logger.error(`${command}_error: ${error.message}...`)
      })
      .execute()
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
          stage: this.stage
        })
        .add()
    } catch (error) {
      console.error('Failed to log to stream:', error)
    }
  }
}
