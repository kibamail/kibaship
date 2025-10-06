import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_node_storages'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.boolean('installation_disk').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('installation_disk')
    })
  }
}
