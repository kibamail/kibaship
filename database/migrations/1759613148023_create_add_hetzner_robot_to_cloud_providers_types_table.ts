import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cloud_providers'

  async up() {
    this.schema.alterTable(this.tableName, (table) => {
      table.enum('type', [
        'aws',
        'hetzner',
        'hetzner_robot',
        'leaseweb',
        'google_cloud',
        'digital_ocean',
        'linode',
        'vultr',
        'ovh',
      ]).notNullable().alter()
    })
  }

  async down() {
    this.schema.alterTable(this.tableName, (table) => {
      table.enum('type', [
        'aws',
        'hetzner',
        'leaseweb',
        'google_cloud',
        'digital_ocean',
        'linode',
        'vultr',
        'ovh',
      ]).notNullable().alter()
    })
  }
}
