import Project from '#models/project'
import User from '#models/user'
import Workspace from '#models/workspace'
import { errors as authErrors } from '@adonisjs/auth'
import type { HttpContext } from '@adonisjs/core/http'

export type ExtraProps = (_props: {
  workspace: Workspace
  projects: Project[]
  activeProject: Project | null
  profile: User | undefined
}) => Promise<Record<string, any>>
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

  public async activateProect(ctx: HttpContext, projectId: string) {
    ctx.session.put('project', projectId)
  }

  public async workspace(ctx: HttpContext) {
    let workspaceSlug = ctx.session.get('workspace')

    if (!workspaceSlug) {
      const firstWorkspace = await Workspace.query()
        .where('user_id', ctx.auth.user?.id as string)
        .first()

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

  public async projectFromWorkspace(ctx: HttpContext, workspaceId: string) {
    const projects = await Project.query().where('workspace_id', workspaceId)

    return this.project(ctx, projects)
  }

  public async project(ctx: HttpContext, projects: Project[]) {
    if (projects.length === 0) {
      return null
    }

    let projectId = ctx.session.get('project')

    console.log({ projectId })

    const project = projects.find((project) => project.id === projectId) || projects[0]

    if (!project) {
      throw new authErrors.E_UNAUTHORIZED_ACCESS(
        'You must select an active project to use this endpoint.',
        {
          redirectTo: '/',
          guardDriverName: 'web',
        }
      )
    }

    if (projectId !== project.id) {
      ctx.session.put('project', project.id)

      projectId = project.id
    }

    return project
  }

  public async pageProps(ctx: HttpContext, extraProps?: ExtraProps) {
    const profile = await this.profile(ctx)
    const workspace = await this.workspace(ctx)

    const projects = await Project.query()
      .where('workspace_id', workspace.id)
      .orderBy('created_at', 'desc')
      .preload('cluster')

    const activeProject = await this.project(ctx, projects)

    return {
      profile,
      workspace,
      projects,
      activeProject,
      ...(await extraProps?.({ workspace, projects, activeProject, profile })),
    }
  }
}
