import { BaseController } from '#controllers/Base/base_controller'
import Project from '#models/project'
import type { HttpContext } from '@adonisjs/core/http'

export default class ProjectsController extends BaseController {
  public async store(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const name: string = ctx.request.input('name') || 'New project'

    const project = await Project.create({ name, workspaceId: workspace.id })

    ctx.session.put('project', project.id)

    return ctx.response.redirect(`/w/${workspace.slug}/dashboard`)
  }

  public async show(ctx: HttpContext) {
    const project = await Project.findOrFail(ctx.params.project)

    const workspace = await this.workspace(ctx)

    await this.activateProect(ctx, project.id)

    return ctx.response.redirect(`/w/${workspace.slug}/dashboard`)
  }
}
