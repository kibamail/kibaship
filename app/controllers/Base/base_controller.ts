import Project from '#models/project'
import { errors as authErrors } from '@adonisjs/auth'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Base controller with shared authentication and profile retrieval methods
 */
export class BaseController {
  /** Get authenticated user's cached profile, throws E_UNAUTHORIZED_ACCESS if invalid */
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

  public async workspace(ctx: HttpContext) {
    const profile = await this.profile(ctx)

    let workspaceSlug = ctx.session.get('workspace')

    if (ctx.params.workspace && ctx.params.workspace !== ctx.session.get('workspace')) {
      ctx.session.put('workspace', ctx.params.workspace)

      workspaceSlug = ctx.params.workspace
    }


    const workspace = profile.workspaces.find((workspace) => workspace.slug === workspaceSlug)

    if (!workspace) {
      throw new authErrors.E_UNAUTHORIZED_ACCESS(
        'You must select an active workspace to use this endpoint.',
        {
          redirectTo: '/',
          guardDriverName: 'web',
        }
      )
    }

    return workspace
  }

  public async pageProps(ctx: HttpContext, extraProps?: Record<string, any>) {
    const profile = await this.profile(ctx)
    const workspace = await this.workspace(ctx)

    const projects = await Project.query()
      .where('workspace_id', workspace.id)
      .orderBy('created_at', 'desc')
      .preload('cluster')

    return {
      profile,
      workspace,
      projects,
      ...extraProps,
    }
  }
}
