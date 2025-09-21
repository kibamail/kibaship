import { BaseController, ExtraProps } from '#controllers/Base/base_controller'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import type { HttpContext } from '@adonisjs/core/http'

export default class CloudController extends BaseController {
  protected props(): ExtraProps {
    return async ({ workspace }) => {
      const connectedProviders = await CloudProvider.query()
        .where('workspace_id', workspace.id)
        .whereNull('deleted_at')
      const clusters = await Cluster.query()
        .where('workspace_id', workspace.id)
        .preload('nodes')
        .preload('cloudProvider')

      return {
        clusters,
        connectedProviders,
        providers: CloudProviderDefinitions.allProviders(),
        regions: CloudProviderDefinitions.allRegions(),
        serverTypes: CloudProviderDefinitions.allServerTypes(),
        cloudProviderRegions: CloudProviderDefinitions.allRegions(),
      }
    }
  }

  public async index(ctx: HttpContext) {
    return ctx.inertia.render('cloud', await this.pageProps(ctx, this.props()))
  }

  public async providers(ctx: HttpContext) {
    return ctx.inertia.render('cloud/providers', await this.pageProps(ctx, this.props()))
  }

  public async clusters(ctx: HttpContext) {
    return ctx.inertia.render('cloud/clusters', await this.pageProps(ctx, this.props()))
  }
}
