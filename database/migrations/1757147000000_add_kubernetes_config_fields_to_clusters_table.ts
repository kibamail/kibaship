import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'clusters'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.text('talos_config').nullable()
      table.text('kube_config').nullable()
      table.text('control_plane_config').nullable()
      table.text('worker_config').nullable()
      
      table.timestamp('kubernetes_config_started_at').nullable()
      table.timestamp('kubernetes_config_completed_at').nullable()
      table.timestamp('kubernetes_config_error_at').nullable()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.dropColumn('talos_config')
      table.dropColumn('kube_config')
      table.dropColumn('control_plane_config')
      table.dropColumn('worker_config')
      
      table.dropColumn('kubernetes_config_started_at')
      table.dropColumn('kubernetes_config_completed_at')
      table.dropColumn('kubernetes_config_error_at')
    })
  }
}