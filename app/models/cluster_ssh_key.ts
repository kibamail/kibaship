import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, belongsTo } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import encryption from '@adonisjs/core/services/encryption'
import Cluster from './cluster.js'
import type { BelongsTo } from '@adonisjs/lucid/types/relations'

export default class ClusterSshKey extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare clusterId: string

  @column()
  declare publicKey: string

  @column({
    prepare: value => encryption.encrypt(value),
    consume: value => encryption.decrypt(value) || '',
    serializeAs: null
  })
  declare privateKey: string

  @column()
  declare providerId: string | null

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @belongsTo(() => Cluster)
  declare cluster: BelongsTo<typeof Cluster>

  @beforeCreate()
  public static async generateId(clusterSshKey: ClusterSshKey) {
    clusterSshKey.id = randomUUID()
  }
}