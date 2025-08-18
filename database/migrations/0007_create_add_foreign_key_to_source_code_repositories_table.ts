import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'source_code_repositories'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table
        .foreign('source_code_provider_id')
        .references('id')
        .inTable('source_code_providers')
        .onDelete('CASCADE')
    })
  }

  async down() {}
}
