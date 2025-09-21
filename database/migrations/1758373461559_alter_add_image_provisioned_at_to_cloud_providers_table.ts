import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cloud_providers'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('provider_image_provisioning_started_at').nullable()
      table.timestamp('provider_image_provisioning_error_at').nullable()
      table.timestamp('provider_image_provisioning_completed_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('provider_image_provisioning_started_at')
      table.dropColumn('provider_image_provisioning_error_at')
      table.dropColumn('provider_image_provisioning_completed_at')
    })
  }
}