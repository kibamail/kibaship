import { BaseController } from '#controllers/Base/base_controller'
import type { HttpContext } from '@adonisjs/core/http'

export default class DashboardController extends BaseController {
  public async index(ctx: HttpContext) {
    const profile = await this.profile(ctx)

    console.log('@@@@@@@profile', profile?.workspaces?.[0])

    return ctx.response?.redirect(`/w/${profile.workspaces?.[0]?.slug}`)
  }

  public async show(ctx: HttpContext) {
    const workspaceSlug = await ctx.params.workspace
    const profile = await this.profile(ctx)

    const workspace = profile?.workspaces?.find((workspace) => workspace.slug === workspaceSlug)

    if (!workspace) {
      return ctx.response?.redirect('/404')
    }

    return ctx.inertia.render('dashboard', {
      profile,
      workspace,
    })
  }
}
