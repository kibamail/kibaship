import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_node_storages'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.integer('size').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('size')
    })
  }
}