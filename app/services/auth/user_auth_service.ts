import User from '#models/user'
import type { HttpContext } from '@adonisjs/core/http'
import { UserProfile } from '@kibamail/auth-sdk'

/**
 * Service responsible for user authentication operations during OAuth flow
 * Handles user lookup, creation, updates, profile caching, and session management
 */
export class UserAuthService {
  /**
   * Find existing user by OAuth ID or create new user from OAuth profile
   * Updates existing user email if it has changed in the OAuth provider
   */
  public async findOrCreateUser(oauthUser: UserProfile) {
    let localUser = await User.findBy('oauthId', oauthUser.id)

    if (localUser) {
      localUser.email = oauthUser.email || localUser.email
      await localUser.save()

      return {
        user: localUser,
        isExisting: true
      }
    }

    localUser = await User.create({
      email: oauthUser.email,
      oauthId: oauthUser.id,
    })

    return {
      user: localUser,
      isExisting: false
    }
  }

  /**
   * Cache user profile and OAuth token in Redis for future API calls
   * Stores encrypted access token and profile data
   */
  public async cacheUserProfile(user: User, profile: UserProfile, accessToken: string) {
    await user.cache(profile, accessToken)
  }

  /**
   * Log user into the web session using AdonisJS authentication
   * Establishes authenticated session for the user
   */
  public async loginUser(ctx: HttpContext, user: User) {
    await ctx.auth.use('web').login(user)
  }
}
