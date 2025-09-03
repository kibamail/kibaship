import { createWorkspaceValidator } from '#validators/create_workspace'
import Workspace from '#models/workspace'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Workspaces controller for creating and managing workspace containers
 */
export default class WorkspacesController {
  /** Create new workspace using Workspace model */
  public async store(ctx: HttpContext) {
    const payload = await createWorkspaceValidator.validate(ctx.request.all())

    const slug = payload.name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')

    const workspace = await Workspace.create({
      name: payload.name,
      slug: slug,
      userId: ctx.auth.user?.id
    })

    return ctx.response.redirect(`/w/${workspace.slug}`)
  }
}
