import { inject } from '@adonisjs/core'
import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { createCloudProviderValidator } from '#validators/create_cloud_provider'
import CloudProvider from '#models/cloud_provider'

@inject()
export default class CloudProvidersController extends BaseController {
    public async store(ctx: HttpContext) {
        const data = await ctx.request.validateUsing(createCloudProviderValidator)
        const workspace = await this.workspace(ctx)

        await CloudProvider.create({
            name: data.name,
            type: data.type,
            workspaceId: workspace.id,
            credentials: data.credentials
        })

        return ctx.response.redirect().toRoute('clusters.index', {
            workspace: workspace.slug
        })
    }
}
