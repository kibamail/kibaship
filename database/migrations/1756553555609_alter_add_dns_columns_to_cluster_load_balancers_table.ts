import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_load_balancers'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('dns_verified_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('dns_verified_at')
    })
  }
}