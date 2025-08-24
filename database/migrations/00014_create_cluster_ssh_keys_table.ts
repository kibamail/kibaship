import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_ssh_keys'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      table.uuid('cluster_id').notNullable().references('id').inTable('clusters').onDelete('CASCADE')
      table.text('public_key').notNullable()
      table.text('private_key').notNullable() // encrypted
      table.string('provider_id').nullable()

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}