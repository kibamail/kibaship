import Project from '#models/project'
import { createApplicationValidator } from '#validators/applications/create_application'
import type { HttpContext } from '@adonisjs/core/http'

export default class ApplicationController {
  public async store(ctx: HttpContext) {
    const data = await createApplicationValidator.validate(ctx.request.all())

    if (!data.projectId) {
      await Project.create({
        // name: data?.name
      })
    }

    return ctx.response.json(data)
  }
}
