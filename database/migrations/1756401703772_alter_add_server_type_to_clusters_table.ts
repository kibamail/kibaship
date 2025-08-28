import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.string('server_type').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('server_type')
    })
  }
}