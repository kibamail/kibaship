import type { HttpContext } from '@adonisjs/core/http'
import { loginUserValidator } from '#validators/login_user'
import User from '#models/user'

export default class LoginController {
  /** Show the login form */
  async show(ctx: HttpContext) {
    const passwordResetSuccess = ctx.session.flashMessages.get('passwordResetSuccess')
    return ctx.inertia.render('Auth/Login', { passwordResetSuccess })
  }

  /** Handle user login with email + password */
  async store(ctx: HttpContext) {
    const payload = await ctx.request.validateUsing(loginUserValidator)

    try {
      const user = await User.verifyCredentials(payload.email, payload.password)
      await ctx.auth.use('web').login(user)

      ctx.session.flash('success', 'Welcome back!')
      return ctx.response.redirect('/w')
    } catch (error) {
      console.error(error)
      ctx.session.flashErrors({ email: 'Invalid email or password' })
      return ctx.response.redirect('/auth/login')
    }
  }
}

