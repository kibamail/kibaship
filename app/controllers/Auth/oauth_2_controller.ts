import User from '#models/user'
import type { HttpContext } from '@adonisjs/core/http'
import app from '@adonisjs/core/services/app'
import redis from '@adonisjs/redis/services/main'
import encryption from '@adonisjs/core/services/encryption'
import { UserProfile } from '@kibamail/auth-sdk'

/**
 * OAuth2 controller for KibaMail authentication flow
 * Handles authorization redirect, callback, token exchange, and user setup
 */
export default class Oauth2Controller {
  /** Redirect user to KibaMail authorization server for OAuth login */
  public async redirect(ctx: HttpContext) {
    const auth = await app.container.make('auth.kibaauth')

    return ctx.response.redirect(auth.api().auth().authorizationUrl())
  }

  /**
   * Handle OAuth callback: exchange code for token, create/update user, login
   * Creates default workspace for new users, redirects to /w on success
   */
  public async callback(ctx: HttpContext) {
    const { code } = ctx.request.qs()

    if (!code) {
      console.error('Failed, no code provided from auth server.')
      return ctx.response.redirect('/')
    }

    const auth = await app.container.make('auth.kibaauth')

    const [response, accessTokenError] = await auth.api().auth().accessToken(code)

    if (accessTokenError) {
      console.error('Failed to get access token from auth server.', accessTokenError?.cause)
      return ctx.response.redirect('/')
    }

    const authenticatedApi = auth.accessToken(response?.access_token as string)

    let [user, profileError] = await authenticatedApi.user().profile()

    if (profileError) {
      console.error('Failed to get profile from auth server.', profileError?.cause)
      return ctx.response.redirect('/')
    }

    let localUser = await User.findBy('oauthId', user?.id)

    if (localUser) {
      localUser.email = user?.email || localUser.email
      await localUser.save()

      await ctx.auth.use('web').login(localUser)

      return ctx.response.redirect('/w')
    } else {
      localUser = await User.create({
        email: user?.email,
        oauthId: user?.id,
      })
    }

    await ctx.auth.use('web').login(localUser)

    await localUser.cache(user as UserProfile, response?.access_token as string)

    const [email, domain] = user?.email?.split('@') || ['', '']

    const [, workspaceError] = await authenticatedApi.workspaces().create({
      name: `${email} ${domain}'s Workspace`,
    })

    if (workspaceError) {
      console.error('Failed to create workspace.', workspaceError?.cause)
      return ctx.response.redirect('/')
    }

    ;[user, profileError] = await authenticatedApi.user().profile()

    if (profileError) {
      console.error('Failed to get profile from auth server.', profileError?.cause)
      return ctx.response.redirect('/')
    }

    return ctx.response.redirect('/w')
  }
}
