import { BaseController } from '#controllers/Base/base_controller'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Dashboard controller for workspace routing and rendering
 * Handles automatic workspace redirects and workspace-specific dashboard display
 */

export default class DashboardController extends BaseController {
  /** Redirect user to their first available workspace */
  public async index(ctx: HttpContext) {
    const profile = await this.profile(ctx)

    return ctx.response?.redirect(`/w/${profile.workspaces?.[0]?.slug}`)
  }

  /** Render dashboard for specific workspace, validates user access first */
  public async show(ctx: HttpContext) {
    const props = await this.pageProps(ctx)

    return ctx.inertia.render('dashboard', props)
  }
}
