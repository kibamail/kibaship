import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import { createClusterValidator } from '#validators/create_cluster'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import db from '@adonisjs/lucid/services/db'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionClusterJob from '#jobs/clusters/provision_cluster_job'

export default class ClustersController extends BaseController {
    public async index(ctx: HttpContext) {
        const workspace = await this.workspace(ctx)

        const connectedProviders = await CloudProvider.query().where('workspace_id', workspace.id)
        const clusters = await Cluster.query().where('workspace_id', workspace.id).preload('nodes')

        return ctx.inertia.render('clusters/clusters', await this.pageProps(ctx, {
            providers: CloudProviderDefinitions.allProviders(),
            regions: CloudProviderDefinitions.allRegions(),
            serverTypes: CloudProviderDefinitions.allServerTypes(),
            connectedProviders,
            clusters,
            cloudProviderRegions: CloudProviderDefinitions.allRegions(),
        }))
    }

    public async store(ctx: HttpContext) {
        const workspace = await this.workspace(ctx)
        const payload = await ctx.request.validateUsing(createClusterValidator)

        const cluster = await db.transaction(async (trx) => {
            return await Cluster.createWithInfrastructure(payload, workspace.id, trx)
        })

        ctx.session.flash('success', `Cluster "${cluster.location}" has been created and is being provisioned.`)

        queue.dispatch(ProvisionClusterJob, {
            clusterId: cluster.id
        })

        return ctx.response.redirect().toRoute('clusters.index', {
            workspace: workspace.slug
        })
    }
}
