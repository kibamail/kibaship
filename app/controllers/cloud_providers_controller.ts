import { inject } from '@adonisjs/core'
import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { createCloudProviderValidator } from '#validators/create_cloud_provider'
import CloudProvider from '#models/cloud_provider'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionHetznerTalosImageJob from '#jobs/cloud-providers/provision_hetzner_talos_image_job'
import { DateTime } from 'luxon'
import { hetznerRobot } from '#services/hetzner-robot/provider'
import { cache } from '#services/cache/cache'

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

  public async servers(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)

    const cloudProvider = await CloudProvider.query()
      .where('id', ctx.params.cloudProvider)
      .where('workspace_id', workspace.id)
      .firstOrFail()

    if (cloudProvider.type !== CloudProviderDefinitions.HETZNER_ROBOT) {
      return ctx.response.badRequest({
        error: 'Only Hetzner Robot providers support server listing',
      })
    }

    const cacheKey = `provider:${cloudProvider.id}`
    const clearCache = ctx.request.input('clearCache') === 'true'

    if (clearCache) {
      await cache('hetzner-robot').item(cacheKey).delete()
    }

    const cachedServers = await cache('hetzner-robot').item(cacheKey).read()

    if (cachedServers) {
      return ctx.response.json(cachedServers)
    }

    const servers = await hetznerRobot({
      username: cloudProvider.credentials.username as string,
      password: cloudProvider.credentials.password as string,
    })
      .servers()
      .list()

    await cache('hetzner-robot').item(cacheKey).write(servers, 3600)

    return ctx.response.json(servers)
  }

  public async vswitches(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)

    const cloudProvider = await CloudProvider.query()
      .where('id', ctx.params.cloudProvider)
      .where('workspace_id', workspace.id)
      .firstOrFail()

    if (cloudProvider.type !== CloudProviderDefinitions.HETZNER_ROBOT) {
      return ctx.response.badRequest({
        error: 'Only Hetzner Robot providers support vSwitch listing',
      })
    }

    const cacheKey = `vswitches:provider:${cloudProvider.id}`
    const clearCache = ctx.request.input('clearCache') === 'true'

    if (clearCache) {
      await cache('hetzner-robot').item(cacheKey).delete()
    }

    const cachedVswitches = await cache('hetzner-robot').item(cacheKey).read()

    if (cachedVswitches) {
      return ctx.response.json(cachedVswitches)
    }

    const vswitches = await hetznerRobot({
      username: cloudProvider.credentials.username as string,
      password: cloudProvider.credentials.password as string,
    })
      .vswitches()
      .list()

    await cache('hetzner-robot').item(cacheKey).write(vswitches, 3600)

    return ctx.response.json(vswitches)
  }
}
