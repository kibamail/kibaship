import { BaseController } from '#controllers/Base/base_controller'
import Project from '#models/project'
import SourceCodeRepository from '#models/source_code_repository'
import { createApplicationValidator } from '#validators/applications/create_application'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Application controller for creating messaging applications
 * Handles validation and optional project creation
 */
export default class ApplicationController extends BaseController {
  public async store(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const data = await createApplicationValidator.validate(ctx.request.all())

    const sourceCodeRepository = await SourceCodeRepository.findOrFail(
      data?.gitConfiguration?.sourceCodeRepositoryId
    )

    const project = data?.projectId
      ? await Project.findOrFail(data?.projectId)
      : await Project.create({
          name: sourceCodeRepository?.repository,
          workspaceId: workspace.id,
        })

    return ctx.response.redirect(`/w/${workspace.slug}/p/${project.id}/?environment=production`)
  }
}
