import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('bare_metal_networking_started_at').nullable()
      table.timestamp('bare_metal_networking_completed_at').nullable()
      table.timestamp('bare_metal_networking_error_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('bare_metal_networking_started_at')
      table.dropColumn('bare_metal_networking_completed_at')
      table.dropColumn('bare_metal_networking_error_at')
    })
  }
}