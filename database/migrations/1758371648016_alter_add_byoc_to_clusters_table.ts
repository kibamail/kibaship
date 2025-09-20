import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'add_byoc_to_clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('byoc_started_at').nullable()
      table.timestamp('byoc_completed_at').nullable()
      table.timestamp('byoc_error_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('byoc_started_at')
      table.dropColumn('byoc_completed_at')
      table.dropColumn('byoc_error_at')
    })
  }
}
