import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'cloud_providers'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      table.string('name').notNullable()
      table
        .enum('type', [
          'aws',
          'hetzner',
          'leaseweb',
          'google_cloud',
          'digital_ocean',
          'linode',
          'vultr',
          'ovh',
        ])
        .notNullable()
      table.uuid('workspace_id').notNullable()
      table.text('credentials').notNullable()

      table.timestamp('deleted_at').nullable()
      table.string('provider_image_arm_64').nullable()
      table.string('provider_image_amd_64').nullable()

      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}
