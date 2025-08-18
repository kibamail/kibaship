import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'

export default class SourceCodeRepository extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare sourceCodeProviderId: string

  @column()
  declare repository: string

  @column()
  declare visibility: 'public' | 'private'

  @column.dateTime()
  declare lastUpdatedAt: DateTime | null

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @beforeCreate()
  public static async generateId(sourceCodeRepository: SourceCodeRepository) {
    sourceCodeRepository.id = randomUUID()
  }
}
