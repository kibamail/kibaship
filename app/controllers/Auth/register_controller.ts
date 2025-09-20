import type { HttpContext } from '@adonisjs/core/http'
import db from '@adonisjs/lucid/services/db'
import User from '#models/user'
import Workspace from '#models/workspace'
import { registerUserValidator } from '#validators/register_user'

export default class RegisterController {
  /**
   * Handle user registration
   * Creates new user with email and hashed password, and default workspace
   */
  async store(ctx: HttpContext) {
    const payload = await ctx.request.validateUsing(registerUserValidator)

    const { user, workspace } = await db.transaction(async (trx) => {
      const user = new User()
      user.email = payload.email
      user.password = payload.password
      user.useTransaction(trx)
      await user.save()

      const [username] = payload.email.split('@')
      const workspaceName = `${username}'s Workspace`
      const slug = username.toLowerCase().replace(/[^a-z0-9]/g, '-')

      const workspace = new Workspace()
      workspace.name = workspaceName
      workspace.slug = slug
      workspace.userId = user.id
      workspace.useTransaction(trx)
      await workspace.save()

      return { user, workspace }
    })

    await ctx.auth.use('web').login(user)

    ctx.session.flash('success', 'Account created successfully! Welcome to Kibaship.')

    return ctx.response.redirect(`/w/${workspace.slug}`)
  }

  /**
   * Show registration form
   */
  async show(ctx: HttpContext) {
    return ctx.inertia.render('Auth/Register')
  }
}