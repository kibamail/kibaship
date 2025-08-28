import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.timestamp('networking_error_at').nullable()
      table.timestamp('ssh_keys_error_at').nullable()
      table.timestamp('load_balancers_error_at').nullable()
      table.timestamp('servers_error_at').nullable()
      table.timestamp('volumes_error_at').nullable()
      table.timestamp('kubernetes_cluster_error_at').nullable()
      table.timestamp('kibaship_operator_error_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('networking_error_at')
      table.dropColumn('ssh_keys_error_at')
      table.dropColumn('load_balancers_error_at')
      table.dropColumn('servers_error_at')
      table.dropColumn('volumes_error_at')
      table.dropColumn('kubernetes_cluster_error_at')
      table.dropColumn('kibaship_operator_error_at')
    })
  }
}