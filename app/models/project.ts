import { DateTime } from 'luxon'
import { afterCreate, BaseModel, beforeCreate, belongsTo, column } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import Environment from './environment.js'
import Cluster from './cluster.js'
import type { BelongsTo } from '@adonisjs/lucid/types/relations'

/**
 * Project model for organizing applications and resources
 */
export default class Project extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare workspaceId: string

  @column()
  declare clusterId: string

  @column()
  declare name: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  @beforeCreate()
  public static async generateId(project: Project) {
    project.id = randomUUID()
  }

  @belongsTo(() => Cluster)
  declare cluster: BelongsTo<typeof Cluster>

  @beforeCreate()
  public static async pickCluster(project: Project) {
    // In future, the cluster will be picked intelligently based on resource utilisation, customer plan, customer preferred location, and other factors.
    const cluster = await Cluster.query().first()

    if (!cluster) {
      throw new Error('No cluster configured.')
    }

    project.clusterId = cluster.id
  }

  @afterCreate()
  public static async createEnvironment(project: Project) {
    await Environment.create({
      name: 'production',
      projectId: project.id,
    })
  }
}
