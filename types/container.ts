import { TerraformExecutor } from '#services/terraform/terraform_executor'

declare module '@adonisjs/core/types' {
  interface ContainerBindings {
    'terraform.executor': typeof TerraformExecutor
  }
}
