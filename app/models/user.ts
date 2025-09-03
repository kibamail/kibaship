import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, hasMany } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import Workspace from '#models/workspace'
import type { HasMany } from '@adonisjs/lucid/types/relations'

/**
 * User model for password-authenticated users with workspaces
 */
export default class User extends BaseModel {
  @column({ isPrimary: true })
  declare id: string


  @column()
  declare email: string

  @column()
  declare password: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime | null

  @hasMany(() => Workspace)
  declare workspaces: HasMany<typeof Workspace>

  @beforeCreate()
  public static async generateId(user: User) {
    user.id = randomUUID()
  }

}
