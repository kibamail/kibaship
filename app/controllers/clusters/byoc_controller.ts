import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from '#controllers/Base/base_controller'
import Cluster from '#models/cluster'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionByocJob from '#jobs/clusters/provision_byoc_job'
import { talosVersion } from '#config/app'
import { createByocClusterValidator } from '#validators/create_byoc_cluster'

export default class ByocController extends BaseController {
  public async store(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const payload = await ctx.request.validateUsing(createByocClusterValidator)

    const cluster = new Cluster()
    cluster.location = payload.location
    cluster.workspaceId = workspace.id
    cluster.status = 'provisioning'
    cluster.kind = 'all_purpose'
    cluster.subdomainIdentifier = `byoc-${Date.now()}`
    cluster.controlPlaneEndpoint = ''
    cluster.serverType = ''
    cluster.controlPlanesVolumeSize = 0
    cluster.workersVolumeSize = 0
    cluster.talosVersion = talosVersion
    cluster.talosConfig = {
      ca_certificate: payload.talosConfig.ca,
      client_certificate: payload.talosConfig.crt,
      client_key: payload.talosConfig.key,
    }

    await cluster.save()

    await queue.dispatch(ProvisionByocJob, { clusterId: cluster.id })

    ctx.session.flash('success', 'BYOC cluster created. Discovery has started.')

    return ctx.response.redirect().toRoute('clusters.index', {
      workspace: workspace.slug,
    })
  }
}

