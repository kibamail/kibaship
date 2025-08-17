import { createWorkspaceValidator } from '#validators/create_workspace'
import type { HttpContext } from '@adonisjs/core/http'

export default class WorkspacesController {
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

    console.log({ workspace })

    return ctx.response.redirect(`/w/${workspace?.slug}`)
  }
}
