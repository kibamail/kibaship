import { DateTime } from 'luxon'
import { customAlphabet } from 'nanoid'
import { BaseModel, beforeCreate, column, belongsTo, hasMany } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import Cluster from './cluster.js'
import ClusterNodeStorage from './cluster_node_storage.js'
import type { BelongsTo, HasMany } from '@adonisjs/lucid/types/relations'

export type ClusterNodeType = 'master' | 'worker'
export type ClusterNodeStatus = 'provisioning' | 'healthy' | 'unhealthy'

export default class ClusterNode extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare type: ClusterNodeType

  @column()
  declare ipv4Address: string | null

  @column()
  declare ipv6Address: string | null

  @column()
  declare privateIpv4Address: string | null

  @column()
  declare clusterId: string

  @column()
  declare slug: string

  @column()
  declare status: ClusterNodeStatus

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @belongsTo(() => Cluster)
  declare cluster: BelongsTo<typeof Cluster>

  @hasMany(() => ClusterNodeStorage)
  declare storages: HasMany<typeof ClusterNodeStorage>

  @beforeCreate()
  public static async generateId(clusterNode: ClusterNode) {
    clusterNode.id = randomUUID()
    clusterNode.slug = customAlphabet('ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789')(10)
  }
}
