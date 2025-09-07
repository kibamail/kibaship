import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('kubernetes_boot_started_at').nullable()
      table.timestamp('kubernetes_boot_completed_at').nullable()
      table.timestamp('kubernetes_boot_error_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('kubernetes_boot_started_at')
      table.dropColumn('kubernetes_boot_completed_at')
      table.dropColumn('kubernetes_boot_error_at')
    })
  }
}