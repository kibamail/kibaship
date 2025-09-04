import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_nodes'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.string('private_network_interface').nullable()
      table.string('public_network_interface').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('private_network_interface')
      table.dropColumn('public_network_interface')
    })
  }
}