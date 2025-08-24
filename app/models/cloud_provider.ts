import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import encryption from '@adonisjs/core/services/encryption'

export default class CloudProvider extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare name: string

  @column()
  declare type: 'aws' | 'hetzner' | 'google_cloud' | 'digital_ocean' | 'leaseweb' | 'linode' | 'vultr' | 'ovh'

  @column()
  declare workspaceId: string

  @column({
    prepare: value => encryption.encrypt(JSON.stringify(value)),
    consume: value => JSON.parse(encryption.decrypt(value) || '{}'),
    serialize: __dirnamevalue => null,
  })
  declare credentials: Partial<{
    token: string
  }>

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  public credentialsPath() {
    return `${this.workspaceId}/${this.type}/${this.id}`
  }

  @beforeCreate()
  public static async generateId(cluster: CloudProvider) {
    cluster.id = randomUUID()
  }
}