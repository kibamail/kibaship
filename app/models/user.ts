import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { UserProfile } from '@kibamail/auth-sdk'
import redis from '@adonisjs/redis/services/main'
import encryption from '@adonisjs/core/services/encryption'
import app from '@adonisjs/core/services/app'
import { randomUUID } from 'node:crypto'

/**
 * User model for OAuth-authenticated users with cached profiles and encrypted tokens
 */
export default class User extends BaseModel {
  /** Auto-generated UUID primary key */
  @column({ isPrimary: true })
  declare id: string

  /** OAuth ID from KibaMail auth service */
  @column()
  declare oauthId: string

  /** Email from OAuth provider */
  @column()
  declare email: string

  /** Creation timestamp */
  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  /** Last update timestamp */
  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime | null

  /** Generate UUID before creating user */
  @beforeCreate()
  public static async generateId(user: User) {
    user.id = randomUUID()
  }

  /**
   * Cache user profile in Redis and store encrypted OAuth token
   * @param user UserProfile from KibaMail API
   * @param accessToken OAuth access token to encrypt and store
   */
  public async cache(user: UserProfile, accessToken: string) {
    await redis.set(`users:${this.id}`, JSON.stringify(user))
    await redis.set(`oauth_tokens:${this.id}`, encryption.encrypt(accessToken))
  }

  /** Get cached user profile from Redis, returns null if not found */
  public async profile() {
    const profile = await redis.get(`users:${this.id}`)

    if (!profile) {
      return null
    }

    return JSON.parse(profile) as UserProfile
  }

  /** Create authenticated KibaMail API client using stored OAuth token */
  public async authClient() {
    const kibaauth = await app.container.make('auth.kibaauth')

    const accessToken = await this.getOauthAccessToken()

    if (!accessToken) {
      throw new Error('No access token found.')
    }

    return kibaauth.accessToken(accessToken)
  }

  /** Retrieve and decrypt OAuth access token from Redis */
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

  /** Fetch fresh profile from KibaMail API and update Redis cache */
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
