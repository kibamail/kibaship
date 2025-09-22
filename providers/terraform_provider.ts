import { ApplicationService } from '@adonisjs/core/types'
import { TerraformExecutor } from '#services/terraform/terraform_executor'

/**
 * TerraformProvider registers the TerraformExecutor in the IoC container
 * to enable dependency injection and easy mocking during tests
 */
export default class TerraformProvider {
  constructor(protected app: ApplicationService) {}

  /**
   * Register method is called to register bindings to the container
   */
  register() {
    // Register TerraformExecutor class constructor as a factory binding
    this.app.container.bind('terraform.executor', () => {
      return TerraformExecutor
    })
  }
}
