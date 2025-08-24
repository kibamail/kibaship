import { BaseController } from '#controllers/Base/base_controller'
import Application from '#models/application'
import Project from '#models/project'
import type { HttpContext } from '@adonisjs/core/http'

export default class ProjectsController extends BaseController {
  public async show(ctx: HttpContext) {
    const applicationId = ctx.request.qs()?.application
    const [project] = await Project.query().preload('cluster').preload('applications').where('id', ctx.params.project).limit(1)


    let application = applicationId ? await Application.findOrFail(applicationId) : null

    if (!project) {
      throw new Error('Project not found')
    }

    return ctx.inertia.render(
      'projects/project',
      await this.pageProps(ctx, {
        project,
        application
      })
    )
  }
}
