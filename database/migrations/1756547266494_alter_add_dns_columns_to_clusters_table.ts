import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('dns_started_at').nullable()
      table.timestamp('dns_completed_at').nullable()
      table.timestamp('dns_last_checked_at').nullable()
      table.timestamp('dns_error_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('dns_started_at')
      table.dropColumn('dns_completed_at')
      table.dropColumn('dns_last_checked_at')
      table.dropColumn('dns_error_at')
    })
  }
}
