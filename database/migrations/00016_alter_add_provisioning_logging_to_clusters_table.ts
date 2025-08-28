import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('networking_started_at').nullable()
      table.timestamp('networking_completed_at').nullable()
      table.text('networking_error').nullable()

      table.timestamp('ssh_keys_started_at').nullable()
      table.timestamp('ssh_keys_completed_at').nullable()
      table.text('ssh_keys_error').nullable()

      table.timestamp('load_balancers_started_at').nullable()
      table.timestamp('load_balancers_completed_at').nullable()
      table.text('load_balancers_error').nullable()

      table.timestamp('servers_started_at').nullable()
      table.timestamp('servers_completed_at').nullable()
      table.text('servers_error').nullable()

      table.timestamp('volumes_started_at').nullable()
      table.timestamp('volumes_completed_at').nullable()
      table.text('volumes_error').nullable()

      table.timestamp('kubernetes_cluster_started_at').nullable()
      table.timestamp('kubernetes_cluster_completed_at').nullable()
      table.text('kubernetes_cluster_error').nullable()

      table.timestamp('kibaship_operator_started_at').nullable()
      table.timestamp('kibaship_operator_completed_at').nullable()
      table.text('kibaship_operator_error').nullable()

      table.string('current_provisioning_step').nullable()
      table.string('overall_provisioning_status').nullable()
      table.timestamp('provisioning_started_at').nullable()
      table.timestamp('provisioning_completed_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('networking_started_at')
      table.dropColumn('networking_completed_at')
      table.dropColumn('networking_error')

      table.dropColumn('ssh_keys_started_at')
      table.dropColumn('ssh_keys_completed_at')
      table.dropColumn('ssh_keys_error')

      table.dropColumn('load_balancers_started_at')
      table.dropColumn('load_balancers_completed_at')
      table.dropColumn('load_balancers_error')

      table.dropColumn('servers_started_at')
      table.dropColumn('servers_completed_at')
      table.dropColumn('servers_error')

      table.dropColumn('volumes_started_at')
      table.dropColumn('volumes_completed_at')
      table.dropColumn('volumes_error')

      table.dropColumn('kubernetes_cluster_started_at')
      table.dropColumn('kubernetes_cluster_completed_at')
      table.dropColumn('kubernetes_cluster_error')

      table.dropColumn('kibaship_operator_started_at')
      table.dropColumn('kibaship_operator_completed_at')
      table.dropColumn('kibaship_operator_error')

      table.dropColumn('current_provisioning_step')
      table.dropColumn('overall_provisioning_status')
      table.dropColumn('provisioning_started_at')
      table.dropColumn('provisioning_completed_at')
    })
  }
}