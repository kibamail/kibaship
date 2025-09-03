import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_nodes'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      table.string('slug').notNullable().unique()
      table.enum('type', ['master', 'worker']).notNullable()

      table.string('provider_id').nullable()
      table.string('ipv4_address').nullable()
      table.string('ipv6_address').nullable()
      table.string('private_ipv4_address').nullable()

      table.uuid('cluster_id').notNullable().references('id').inTable('clusters').onDelete('CASCADE')
      table.enum('status', ['provisioning', 'healthy', 'unhealthy']).notNullable().defaultTo('provisioning')
      table.string('server_type').nullable()

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}
