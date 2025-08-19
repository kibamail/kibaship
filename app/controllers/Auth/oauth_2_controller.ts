import { OauthService } from '#services/auth/oauth_service'
import { UserAuthService } from '#services/auth/user_auth_service'
import { inject } from '@adonisjs/core'
import type { HttpContext } from '@adonisjs/core/http'
import { UserProfile } from '@kibamail/auth-sdk'

/**
 * OAuth2 controller for KibaMail authentication flow
 * Handles authorization redirect, callback, token exchange, and user setup
 */
@inject()
export default class Oauth2Controller {

  constructor(
    protected auth: OauthService,
    protected userAuth: UserAuthService
  ) {}

  /** Redirect user to KibaMail authorization server for OAuth login */
  public async redirect(ctx: HttpContext) {
    return ctx.response.redirect(this.auth.api().auth().authorizationUrl())
  }

  /**
   * Handle OAuth callback: exchange code for token, create/update user, login
   * Creates default workspace for new users, redirects to /w on success
   */
  public async callback(ctx: HttpContext) {
    const { code } = ctx.request.qs()

    if (!code) {
      return ctx.response.redirect('/')
    }

    const [response, accessTokenError] = await this.auth.api().auth().accessToken(code)

    if (accessTokenError) {
      return ctx.response.redirect(`/?error=Failed to authenticate. Please try again.`)
    }

    const authenticatedApi = this.auth.accessToken(response?.access_token as string)

    let [user, profileError] = await authenticatedApi.user().profile()

    if (profileError) {
      return ctx.response.redirect('/?error=Failed to get your profile information. Please try again.')
    }

    const localUser = await this.userAuth.findOrCreateUser(user!)

    if (localUser.isExisting) {
      await this.userAuth.loginUser(ctx, localUser.user)
      return ctx.response.redirect('/w')
    }

    await this.userAuth.loginUser(ctx, localUser.user)

    const workspaceCreated = await this.setupNewUser(localUser.user, user!, response?.access_token as string)
    if (!workspaceCreated) {
      return ctx.response.redirect('/')
    }

    const profileRefreshed = await this.refreshUserProfile(response?.access_token as string)
    if (!profileRefreshed) {
      return ctx.response.redirect('/')
    }

    return ctx.response.redirect('/w')
  }

  /**
   * Setup new user by caching profile and creating default workspace
   */
  private async setupNewUser(localUser: any, oauthUser: any, accessToken: string) {
    await this.userAuth.cacheUserProfile(localUser, oauthUser as UserProfile, accessToken)

    const [email, domain] = oauthUser?.email?.split('@') || ['', '']
    const authenticatedApi = this.auth.accessToken(accessToken)

    const [, workspaceError] = await authenticatedApi.workspaces().create({
      name: `${email} ${domain}'s Workspace`,
    })

    if (workspaceError) {
      return false
    }

    return true
  }

  /**
   * Refresh user profile after workspace creation
   */
  private async refreshUserProfile(accessToken: string) {
    const authenticatedApi = this.auth.accessToken(accessToken)
    const [, profileError] = await authenticatedApi.user().profile()

    if (profileError) {
      return false
    }

    return true
  }
}
