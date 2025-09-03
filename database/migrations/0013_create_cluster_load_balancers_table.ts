import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_load_balancers'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      table.uuid('cluster_id').notNullable().references('id').inTable('clusters').onDelete('CASCADE')

      table.enum('type', ['cluster', 'ingress', 'tcp', 'udp']).notNullable()
      table.string('public_ipv4_address').nullable()
      table.string('private_ipv4_address').nullable()
      table.string('provider_id').nullable()

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}