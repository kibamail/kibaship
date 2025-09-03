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

      // Server type for cloud provider
      table.string('server_type').nullable()

      // Talos image fields
      table.string('provider_image_id').nullable()
      table.string('talos_version').notNullable()
      table.timestamp('talos_image_started_at').nullable()
      table.timestamp('talos_image_completed_at').nullable()
      table.timestamp('talos_image_error_at').nullable()

      // Provisioning logging fields
      table.timestamp('networking_started_at').nullable()
      table.timestamp('networking_completed_at').nullable()
      table.timestamp('networking_error_at').nullable()

      table.timestamp('ssh_keys_started_at').nullable()
      table.timestamp('ssh_keys_completed_at').nullable()
      table.timestamp('ssh_keys_error_at').nullable()

      table.timestamp('load_balancers_started_at').nullable()
      table.timestamp('load_balancers_completed_at').nullable()
      table.timestamp('load_balancers_error_at').nullable()

      table.timestamp('servers_started_at').nullable()
      table.timestamp('servers_completed_at').nullable()
      table.timestamp('servers_error_at').nullable()

      table.timestamp('volumes_started_at').nullable()
      table.timestamp('volumes_completed_at').nullable()
      table.timestamp('volumes_error_at').nullable()

      table.timestamp('kubernetes_cluster_started_at').nullable()
      table.timestamp('kubernetes_cluster_completed_at').nullable()
      table.timestamp('kubernetes_cluster_error_at').nullable()

      table.timestamp('kibaship_operator_started_at').nullable()
      table.timestamp('kibaship_operator_completed_at').nullable()
      table.timestamp('kibaship_operator_error_at').nullable()

      table.string('current_provisioning_step').nullable()
      table.string('overall_provisioning_status').nullable()
      table.timestamp('provisioning_started_at').nullable()
      table.timestamp('provisioning_completed_at').nullable()

      // DNS fields  
      table.timestamp('dns_started_at').nullable()
      table.timestamp('dns_completed_at').nullable()
      table.timestamp('dns_last_checked_at').nullable()
      table.timestamp('dns_error_at').nullable()

      // Soft delete
      table.timestamp('deleted_at').nullable()

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}
