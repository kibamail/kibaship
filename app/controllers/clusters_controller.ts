import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import CloudProvider from '#models/cloud_provider'

export default class ClustersController extends BaseController {
    public async index(ctx: HttpContext) {
        const workspace = await this.workspace(ctx)

        const connectedProviders = await CloudProvider.query().where('workspace_id', workspace.id)

        return ctx.inertia.render('clusters/clusters', await this.pageProps(ctx, {
            providers: CloudProviderDefinitions.allProviders(),
            regions: CloudProviderDefinitions.allRegions(),
            serverTypes: CloudProviderDefinitions.allServerTypes(),
            connectedProviders,
            cloudProviderRegions: CloudProviderDefinitions.allRegions(),
        }))
    }

    public async store(ctx: HttpContext) {
        // create cluster
        // create nodes
        // create ssh keys for cluster (by generating ssh keys with ssh keys generator)
        // dispatch job to provision cluster
    }
}
