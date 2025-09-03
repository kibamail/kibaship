import { BaseSchema } from '@adonisjs/lucid/schema'

export default class extends BaseSchema {
  protected tableName = 'deployments'

  async up() {
    this.schema.createTable(this.tableName, (table) => {
      table.uuid('id').primary()

      table.uuid('application_id').notNullable().references('id').inTable('applications').onDelete('CASCADE')
      table.uuid('environment_id').notNullable().references('id').inTable('environments').onDelete('CASCADE')

      // A snapshot of the configurations used to create this deployment. Will always be a copy of the environment configurations at the time of deployment trigger.
      // will also include some more deployment specific configurations, like commit sha for git deployments, image tag for docker image deployments, etc.
      table.json('configurations').notNullable()
      // includes meta information such as deployment time, etc.
      table.json('metadata').nullable()

      // only one deployment per environment will be promoted, and this deployment's unique url will point to the environment configured primary domain
      // by default each new deployment will be auto promoted, and the previously promoted deployment will be updated so that this field is set to null
      table.timestamp('promoted_at').nullable()


      table.timestamp('created_at')
      table.timestamp('updated_at')
    })
  }

  async down() {
    this.schema.dropTable(this.tableName)
  }
}