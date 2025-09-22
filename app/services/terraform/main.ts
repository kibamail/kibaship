import app from '@adonisjs/core/services/app'
import { TerraformExecutorContract } from '#contracts/terraform_executor'
import { TerraformStage } from './terraform_executor.js'

/**
 * Container service for creating TerraformExecutor instances
 * This provides a clean API for dependency injection while maintaining
 * the ability to mock during testing
 */
export async function createExecutor(clusterId: string, stage: TerraformStage): Promise<TerraformExecutorContract> {
  const ExecutorClass = await app.container.make('terraform.executor')
  return new ExecutorClass(clusterId, stage)
}

// Re-export types for convenience
export type { TerraformExecutorContract } from '#contracts/terraform_executor'
export type { TerraformStage, TerraformCommand, TerraformExecutionOptions, TerraformExecutionResult } from './terraform_executor.js'
export { TerraformExecutor } from './terraform_executor.js'
