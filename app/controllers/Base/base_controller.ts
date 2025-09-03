import Project from '#models/project'
import Workspace from '#models/workspace'
import { errors as authErrors } from '@adonisjs/auth'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Base controller with shared authentication and profile retrieval methods
 */
export class BaseController {
  /** Get authenticated user's cached profile, throws E_UNAUTHORIZED_ACCESS if invalid */
  public async profile(ctx: HttpContext) {
    await ctx.auth.user?.load((preloader) => {
      preloader.load('workspaces')
    })

    return ctx.auth.user
  }

  public async workspace(ctx: HttpContext) {
    let workspaceSlug = ctx.session.get('workspace')

    if (! workspaceSlug) {
      const firstWorkspace = await Workspace.query().where('user_id', ctx.auth.user?.id as string).first()

      workspaceSlug = firstWorkspace?.slug
    }

    if (ctx.params.workspace && ctx.params.workspace !== ctx.session.get('workspace')) {
      ctx.session.put('workspace', ctx.params.workspace)

      workspaceSlug = ctx.params.workspace
    }

    const workspace = await Workspace.query().where('slug', workspaceSlug).first()

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
