import { createWorkspaceValidator } from '#validators/create_workspace'
import type { HttpContext } from '@adonisjs/core/http'

/**
 * Workspaces controller for creating and managing workspace containers
 * Integrates with KibaMail API for workspace operations
 */
export default class WorkspacesController {
  /** Create new workspace via KibaMail API, refresh profile, redirect to workspace */
  public async store(ctx: HttpContext) {
    const payload = await createWorkspaceValidator.validate(ctx.request.all())

    const api = await ctx.auth.user?.authClient()

    if (!api) {
      return ctx.response.status(401).json({ error: 'Unauthorized' })
    }

    const [workspace, workspaceError] = await api.workspaces().create(payload)

    if (workspaceError) {
      return ctx.response.status(400).json({ error: workspaceError.cause?.response?.data })
    }

    await ctx.auth.user?.refreshProfile()

    return ctx.response.redirect(`/w/${workspace?.slug}`)
  }
}
