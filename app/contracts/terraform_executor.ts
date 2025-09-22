import { TerraformExecutionOptions, TerraformStage, TerraformOutputResult } from '#services/terraform/terraform_executor'
import { type Subprocess } from 'execa'

/**
 * Abstract contract for TerraformExecutor to enable dependency injection and mocking
 */
export abstract class TerraformExecutorContract {
  constructor(protected clusterId: string, protected stage: TerraformStage) {}

  /**
   * Set Terraform variables
   */
  abstract vars(variables: Record<string, string | number | boolean>): this

  /**
   * Initialize the Terraform stream
   */
  abstract initializeStream(): Promise<void>

  /**
   * Execute terraform init
   */
  abstract init(options?: TerraformExecutionOptions): Promise<void>

  /**
   * Execute terraform plan
   */
  abstract plan(options?: TerraformExecutionOptions): Promise<void>

  /**
   * Execute terraform apply
   */
  abstract apply(options?: TerraformExecutionOptions): Promise<Subprocess>

  /**
   * Execute terraform destroy
   */
  abstract destroy(options?: TerraformExecutionOptions): Promise<Subprocess>

  /**
   * Get the stream name for this executor
   */
  abstract getStreamName(): string

  /**
   * Get the Terraform working directory
   */
  abstract getTerraformDirectory(): string

  /**
   * Get stream length (number of log entries)
   */
  abstract getLogCount(): Promise<number>

  /**
   * Read historical logs from the stream
   */
  abstract readLogs(fromId?: string, count?: number): Promise<unknown[]>

  /**
   * Get terraform output
   */
  abstract output(): Promise<TerraformOutputResult>
}
