import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column, belongsTo } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import Cluster from './cluster.js'
import type { BelongsTo } from '@adonisjs/lucid/types/relations'

export type ClusterLoadBalancerType = 'cluster' | 'ingress' | 'tcp' | 'udp'

export default class ClusterLoadBalancer extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare clusterId: string

  @column()
  declare type: ClusterLoadBalancerType

  @column({
    columnName: 'public_ipv4_address'
  })
  declare publicIpv4Address: string | null

  @column({
    columnName: 'private_ipv4_address'
  })
  declare privateIpv4Address: string | null

  @column()
  declare providerId: string | null

  @column.dateTime()
  declare dnsVerifiedAt: DateTime

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @belongsTo(() => Cluster)
  declare cluster: BelongsTo<typeof Cluster>

  @beforeCreate()
  public static async generateId(clusterLoadBalancer: ClusterLoadBalancer) {
    clusterLoadBalancer.id = randomUUID()
  }
}
