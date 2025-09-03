import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, hasMany, belongsTo } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import User from '#models/user'
import Project from '#models/project'
import type { HasMany, BelongsTo } from '@adonisjs/lucid/types/relations'

export default class Workspace extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare name: string

  @column()
  declare slug: string

  @column()
  declare userId: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @belongsTo(() => User)
  declare user: BelongsTo<typeof User>

  @hasMany(() => Project)
  declare projects: HasMany<typeof Project>

  @beforeCreate()
  public static async generateId(workspace: Workspace) {
    workspace.id = randomUUID()
  }
}