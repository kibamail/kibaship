import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cluster_node_storages'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      // this will map to the attached volume id for the volume, but on bare metal, provider_id will be null and provider_mount_id will refer to where the bare metal disk is mounted
      table.string('provider_id').nullable()
      table.string('provider_mount_id').nullable()
      table.enum('status', ['provisioning', 'healthy', 'unhealthy']).notNullable().defaultTo('provisioning')
      table.uuid('cluster_node_id').notNullable().references('id').inTable('cluster_nodes').onDelete('CASCADE')

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}