import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import { createClusterValidator } from '#validators/create_cluster'
import { clusterRestartValidator } from '#validators/cluster_restart'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import db from '@adonisjs/lucid/services/db'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionClusterJob from '#jobs/clusters/provision_cluster_job'
import ProvisionNetworkJob from '#jobs/clusters/provision_network_job'
import ProvisionSshKeysJob from '#jobs/clusters/provision_ssh_keys_job'
import ProvisionLoadBalancersJob from '#jobs/clusters/provision_load_balancers_job'

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

    public async show(ctx: HttpContext) {
        const cluster = await Cluster.complete(ctx.params.clusterId)

        if (!cluster) {
            return ctx.response.status(404).json({ error: 'Cluster not found' })
        }

        return ctx.response.json(cluster)
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

    public async restart(ctx: HttpContext) {
        const workspace = await this.workspace(ctx)
        const clusterId = ctx.params.clusterId
        const payload = await ctx.request.validateUsing(clusterRestartValidator)

        const cluster = await Cluster.query()
            .where('id', clusterId)
            .where('workspace_id', workspace.id)
            .preload('cloudProvider')
            .firstOrFail()

        if (payload.type === 'start') {
            await queue.dispatch(ProvisionClusterJob, { clusterId })

            return ctx.response.json({
                message: 'Cluster provisioning restarted from beginning',
                type: 'start'
            })
        }

        const failedStage = cluster.getFirstFailedStage()

        if (!failedStage) {
            return ctx.response.status(400).json({
                error: 'No failed stages found to restart from'
            })
        }

        const job = this.getJobForStage(failedStage)

        await queue.dispatch(job, { clusterId })

        return ctx.response.json({
            message: `Cluster provisioning restarted from failed stage: ${failedStage}`,
            type: 'failed',
            stage: failedStage
        })
    }

    private getJobForStage(stage: string) {
        switch (stage) {
            case 'network':
                return ProvisionNetworkJob
            case 'ssh-keys':
                return ProvisionSshKeysJob
            case 'load-balancers':
                return ProvisionLoadBalancersJob
            case 'servers':
                throw new Error('ProvisionServersJob not implemented yet')
            case 'volumes':
                throw new Error('ProvisionVolumesJob not implemented yet')
            case 'kubernetes':
                throw new Error('ProvisionKubernetesJob not implemented yet')
            default:
                throw new Error(`Unknown stage: ${stage}`)
        }
    }
}
