import { BaseController } from '#controllers/Base/base_controller'
import SourceCodeProvider from '#models/source_code_provider'
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
    const workspaceSlug = await ctx.params.workspace

    const profile = await this.profile(ctx)

    const workspace = profile?.workspaces?.find((workspace) => workspace.slug === workspaceSlug)

    if (!workspace) {
      return ctx.response?.redirect('/404')
    }

    ctx.session.put('workspace', workspace.slug)

    return ctx.inertia.render('dashboard', {
      profile,
      workspace,
    })
  }
}
