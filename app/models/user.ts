import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { UserProfile } from '@kibamail/auth-sdk'
import redis from '@adonisjs/redis/services/main'
import encryption from '@adonisjs/core/services/encryption'
import app from '@adonisjs/core/services/app'
import { randomUUID } from 'node:crypto'

export default class User extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare oauthId: string

  @column()
  declare email: string

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime | null

  @beforeCreate()
  public static async generateId(user: User) {
    user.id = randomUUID()
  }

  /**
   * Store the user object alongside teams and workspaces in redis
   *
   * @param user {UserProfile}
   * @param accessToken {string}
   */
  public async cache(user: UserProfile, accessToken: string) {
    await redis.set(`users:${this.id}`, JSON.stringify(user))
    await redis.set(`oauth_tokens:${this.id}`, encryption.encrypt(accessToken))
  }

  public async profile() {
    const profile = await redis.get(`users:${this.id}`)

    if (!profile) {
      return null
    }

    return JSON.parse(profile) as UserProfile
  }

  public async authClient() {
    const kibaauth = await app.container.make('auth.kibaauth')

    const accessToken = await this.getOauthAccessToken()

    if (!accessToken) {
      throw new Error('No access token found.')
    }

    return kibaauth.accessToken(accessToken)
  }

  private async getOauthAccessToken() {
    const encryptedAccessToken = await redis.get(`oauth_tokens:${this.id}`)

    if (!encryptedAccessToken) {
      return null
    }

    const accessToken = encryption.decrypt<string>(encryptedAccessToken)

    if (!accessToken) {
      return null
    }

    return accessToken
  }

  public async refreshProfile() {
    const encryptedAccessToken = await redis.get(`oauth_tokens:${this.id}`)

    if (!encryptedAccessToken) {
      return null
    }

    const accessToken = encryption.decrypt<string>(encryptedAccessToken)

    if (!accessToken) {
      return null
    }

    const kibaauth = await app.container.make('auth.kibaauth')

    const [profile, profileError] = await kibaauth.accessToken(accessToken).user().profile()

    if (profileError) {
      console.error('Failed to refresh user profile:', profileError)
      return null
    }

    await this.cache(profile as UserProfile, accessToken)

    return profile
  }
}
