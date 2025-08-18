import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, hasMany } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import Project from './project.js'
import type { HasMany } from '@adonisjs/lucid/types/relations'

export default class Cluster extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare location: string

  @column()
  declare controlPlaneEndpoint: string

  @column()
  declare subdomainIdentifier: string

  @column()
  declare kind: 'all_purpose' | 'volume_storage' | 'pipelines'

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @hasMany(() => Project)
  declare projects: HasMany<typeof Project>

  @beforeCreate()
  public static async generateId(cluster: Cluster) {
    cluster.id = randomUUID()
  }
}
