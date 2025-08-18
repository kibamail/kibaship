import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'

export default class SourceCodeProvider extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare provider: 'github' | 'gitlab' | 'bitbucket'

  @column()
  declare type: 'organization' | 'user'

  @column()
  declare name: string

  @column()
  declare avatar: string

  @column()
  declare workspaceId: string

  @column()
  declare providerId: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @beforeCreate()
  public static async generateId(sourceCodeProvider: SourceCodeProvider) {
    sourceCodeProvider.id = randomUUID()
  }
}
