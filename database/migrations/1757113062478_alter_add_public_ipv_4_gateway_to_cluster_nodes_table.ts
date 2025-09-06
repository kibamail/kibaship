import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_nodes'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.string('public_ipv4_gateway').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('public_ipv4_gateway')
    })
  }
}