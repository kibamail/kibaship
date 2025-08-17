import { errors as authErrors } from '@adonisjs/auth'
import type { HttpContext } from '@adonisjs/core/http'

export class BaseController {
  public async profile(ctx: HttpContext) {
    const user = ctx.auth.user

    const profile = await user?.profile()

    if (!user || !profile) {
      throw new authErrors.E_UNAUTHORIZED_ACCESS('You must be logged in to view this page.', {
        redirectTo: '/',
        guardDriverName: 'web',
      })
    }

    return profile
  }
}
