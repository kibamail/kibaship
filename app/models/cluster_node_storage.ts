import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, belongsTo } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import ClusterNode from './cluster_node.js'
import type { BelongsTo } from '@adonisjs/lucid/types/relations'

export type ClusterNodeStorageStatus = 'provisioning' | 'healthy' | 'unhealthy'

export default class ClusterNodeStorage extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare providerId: string | null

  @column()
  declare providerMountId: string | null

  @column()
  declare status: ClusterNodeStorageStatus

  @column()
  declare clusterNodeId: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @belongsTo(() => ClusterNode)
  declare clusterNode: BelongsTo<typeof ClusterNode>

  @beforeCreate()
  public static async generateId(clusterNodeStorage: ClusterNodeStorage) {
    clusterNodeStorage.id = randomUUID()
  }
}