import { BaseController } from '#controllers/Base/base_controller'
import Project from '#models/project'
import type { HttpContext } from '@adonisjs/core/http'

export default class ProjectsController extends BaseController {
  public async show(ctx: HttpContext) {
    const project = await Project.findOrFail(ctx.params.project)

    return ctx.inertia.render(
      'projects/project',
      await this.pageProps(ctx, {
        project,
      })
    )
  }
}
