import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_node_storages'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.string('slug').notNullable().unique()
      table.string('disk_name').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('slug')
      table.dropColumn('disk_name')
    })
  }
}