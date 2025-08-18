import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'source_code_providers'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()
      table.uuid('workspace_id').notNullable()
      table.string('avatar').nullable()
      table.enum('provider', ['github', 'gitlab', 'bitbucket']).notNullable()
      table.enum('type', ['organization', 'user']).notNullable()
      table.string('name').notNullable()
      table.string('provider_id').unique().notNullable()
      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}
