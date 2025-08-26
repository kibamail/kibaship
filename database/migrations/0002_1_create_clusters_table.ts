import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      table.string('location').notNullable()
      table.string('subdomain_identifier').notNullable().unique()
      table.string('kind').defaultTo('all_purpose')
      table.string('control_plane_endpoint').notNullable()

      table.uuid('workspace_id').nullable()
      // if no cloud provider id is set, then this is a bare metal server, and we'll need the cluster owner to provide the actual 
      table.uuid('cloud_provider_id').nullable().references('id').inTable('cloud_providers').onDelete('CASCADE')
      table.enum('status', ['provisioning', 'healthy', 'unhealthy']).notNullable().defaultTo('provisioning')

      table.string('provider_network_id').nullable()
      table.string('provider_subnet_id').nullable()

      table.string('network_ip_range').nullable()
      table.string('subnet_ip_range').nullable()

      // this is required for high availability control plane access, example: kube.staging.kibaship.com for our main cluster
      table.string('public_domain').nullable()

      // Volume sizes in GB for control plane and worker nodes
      table.integer('control_planes_volume_size').notNullable()
      table.integer('workers_volume_size').notNullable()

      table.json('progress').nullable()

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}
