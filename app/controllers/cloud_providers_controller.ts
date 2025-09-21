import { inject } from '@adonisjs/core'
import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { createCloudProviderValidator } from '#validators/create_cloud_provider'
import CloudProvider from '#models/cloud_provider'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionHetznerTalosImageJob from '#jobs/cloud-providers/provision_hetzner_talos_image_job'
import { DateTime } from 'luxon'

@inject()
export default class CloudProvidersController extends BaseController {
  public async store(ctx: HttpContext) {
    const data = await ctx.request.validateUsing(createCloudProviderValidator)
    const workspace = await this.workspace(ctx)

    const cloudProvider = await CloudProvider.create({
      name: data.name,
      type: data.type,
      workspaceId: workspace.id,
      credentials: data.credentials,
      providerImageProvisioningCompletedAt:
        data.type === CloudProviderDefinitions.HETZNER ? null : DateTime.now(),
      providerImageProvisioningStartedAt:
        data.type === CloudProviderDefinitions.HETZNER ? null : DateTime.now(),
    })

    if (data.type === CloudProviderDefinitions.HETZNER) {
      await queue.dispatch(ProvisionHetznerTalosImageJob, {
        cloudProviderId: cloudProvider.id,
      })
    }

    return ctx.response.redirect().toRoute('workspace.cloud.providers', {
      workspace: workspace.slug,
    })
  }

  public async destroy(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)

    const cloudProvider = await CloudProvider.query()
      .where('id', ctx.params.cloudProvider)
      .where('workspace_id', workspace.id)
      .firstOrFail()

    cloudProvider.deletedAt = DateTime.now()
    cloudProvider.credentials = {}

    await cloudProvider.save()

    return ctx.response.redirect().toRoute('workspace.cloud.providers', {
      workspace: workspace.slug,
    })
  }
}
