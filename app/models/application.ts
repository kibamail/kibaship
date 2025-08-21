import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'

type ApplicationConfiguration = {
  gitConfiguration: {
    sourceCodeRepositoryId: string
  }
  dockerImageConfiguration: {
    image: string
  }
}

/**
 * Application model for messaging endpoints with unique identification
 */
export default class Application extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare projectId: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column()
  declare name: string

  @column()
  declare type: 'mysql' | 'postgres' | 'git' | 'docker_image' | 'template'

  @column()
  declare configurations: Partial<ApplicationConfiguration>

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @beforeCreate()
  public static async generateId(application: Application) {
    application.id = randomUUID()
  }
}
