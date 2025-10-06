import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table
        .uuid('robot_cloud_provider_id')
        .nullable()
        .references('id')
        .inTable('cloud_providers')
        .onDelete('SET NULL')
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropForeign(['robot_cloud_provider_id'])
      table.dropColumn('robot_cloud_provider_id')
    })
  }
}