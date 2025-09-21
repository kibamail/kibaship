import { BaseController } from '#controllers/Base/base_controller'
import Application from '#models/application'
import Project from '#models/project'
import SourceCodeRepository from '#models/source_code_repository'
import { createApplicationValidator } from '#validators/applications/create_application'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Application controller for creating messaging applications
 * Handles validation and optional project creation
 */
export default class ApplicationController extends BaseController {
  public async index(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const projects = await Project.query()
      .where('workspace_id', workspace.id)
      .preload('applications')

    const project = await this.project(ctx, projects)

    let applications: Application[] = []

    if (project) {
      applications = await Application.query().where('project_id', project.id)
    }

    return ctx.inertia.render(
      'projects/applications',
      await this.pageProps(ctx, async () => ({
        applications,
      }))
    )
  }

  public async store(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const project = await this.projectFromWorkspace(ctx, workspace.id)

    const data = await createApplicationValidator.validate(ctx.request.all())

    const sourceCodeRepository = await SourceCodeRepository.findOrFail(
      data?.gitConfiguration?.sourceCodeRepositoryId
    )

    const application = await Application.create({
      name: sourceCodeRepository.repository,
      projectId: project?.id,
      type: data.type,
      configurations: {
        gitConfiguration: data.gitConfiguration,
        dockerImageConfiguration: data.dockerImageConfiguration,
      },
    })

    return ctx.response.redirect(
      `/w/${workspace.slug}/applications/${application?.id}/?environment=production`
    )
  }
}
