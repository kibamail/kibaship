import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import { createClusterValidator } from '#validators/create_cluster'
import { createHetznerRobotClusterValidator } from '#validators/create_hetzner_robot_cluster'
import { clusterRestartValidator } from '#validators/cluster_restart'
import { cache } from '#services/cache/cache'
import CloudProvider from '#models/cloud_provider'
import Cluster from '#models/cluster'
import db from '@adonisjs/lucid/services/db'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionClusterJob from '#jobs/clusters/provision_cluster_job'
import ProvisionNetworkJob from '#jobs/clusters/provision_network_job'
import ProvisionSshKeysJob from '#jobs/clusters/provision_ssh_keys_job'
import ProvisionLoadBalancersJob from '#jobs/clusters/provision_load_balancers_job'
import ProvisionServersJob from '#jobs/clusters/provision_servers_job'
import ProvisionVolumesJob from '#jobs/clusters/provision_volumes_job'
import DestroyClusterJob from '#jobs/clusters/destroy_cluster_job'
import ProvisionTalosImageJob from '#jobs/clusters/provision_talos_image_job'
import { TerraformStage } from '#services/terraform/terraform_executor'
import ProvisionKubernetesConfigJob from '#jobs/clusters/provision_kubernetes_config_job'
import ProvisionKubernetesBootJob from '#jobs/clusters/provision_kubernetes_boot_job'
import ProvisionHetznerBareMetalCluster from '#jobs/clusters/provision_hetzner_bare_metal_cluster'
import ProvisionBareMetalCloudLoadBalancerJob from '#jobs/clusters/provision_bare_metal_cloud_load_balancer_job'
import ProvisionBareMetalTalosImageJob from '#jobs/clusters/provision_bare_metal_talos_image_job'
import ProvisionBareMetalServersBootstrapJob from '#jobs/clusters/provision_bare_metal_servers_bootstrap_job'
import { DateTime } from 'luxon'

export default class ClustersController extends BaseController {
  protected async clustersPageProps(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)

    const connectedProviders = await CloudProvider.query().where('workspace_id', workspace.id)

    const clusters = await Cluster.query()
      .where('workspace_id', workspace.id)
      .preload('nodes')
      .preload('cloudProvider')

    return {
      ...(await this.pageProps(ctx)),
      clusters,
      connectedProviders,
      providers: CloudProviderDefinitions.allProviders(),
      regions: CloudProviderDefinitions.allRegions(),
      serverTypes: CloudProviderDefinitions.allServerTypes(),
      cloudProviderRegions: CloudProviderDefinitions.allRegions(),
    }
  }

  public async index(ctx: HttpContext) {
    return ctx.inertia.render('clusters/clusters', await this.clustersPageProps(ctx))
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

    ctx.session.flash(
      'success',
      `Cluster "${cluster.location}" has been created and is being provisioned.`
    )

    queue.dispatch(ProvisionClusterJob, {
      clusterId: cluster.id,
    })

    return ctx.response.redirect().toRoute('clusters.index', {
      workspace: workspace.slug,
    })
  }

  public async storeHetznerRobotCluster(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const payload = await ctx.request.validateUsing(createHetznerRobotClusterValidator)

    const cloudProvider = await CloudProvider.findOrFail(payload.cloud_provider_id)

    const serversCacheKey = `provider:${cloudProvider.id}`
    const cachedServers = (await cache('hetzner-robot').item(serversCacheKey).read()) as Array<{
      server_number: number
      server_ip: string
      server_name: string
    }> | null

    if (!cachedServers) {
      return ctx.response.badRequest({
        error: 'Server data not found in cache. Please refresh the page.',
      })
    }

    const selectedServers = cachedServers.filter((s) =>
      payload.robot_server_numbers.includes(s.server_number)
    )

    let vlanId: number | null = null
    let vswitchId: number | null = null
    let vlanName: string | null = null

    if (payload.robot_vswitch_id) {
      const cacheKey = `vswitches:provider:${cloudProvider.id}`
      const cachedVswitches = (await cache('hetzner-robot').item(cacheKey).read()) as Array<{
        id: number
        name: string
        vlan: number
      }> | null

      const vswitch = cachedVswitches?.find((v) => v.id === Number(payload.robot_vswitch_id))
      if (vswitch) {
        vlanId = vswitch.vlan
        vswitchId = vswitch.id
        vlanName = vswitch.name
      }
    }

    const cluster = await db.transaction(async (trx) => {
      return await Cluster.createWithHetznerRobotServers(
        {
          subdomain_identifier: payload.subdomain_identifier,
          cloud_provider_id: payload.cloud_provider_id,
          robot_cloud_provider_id: payload.robot_cloud_provider_id,
          region: payload.region,
          robot_server_numbers: payload.robot_server_numbers,
          servers: selectedServers,
          vlan_id: vlanId,
          vswitch_id: vswitchId,
          vlan_name: vlanName,
        },
        workspace.id,
        trx
      )
    })

    ctx.session.flash(
      'success',
      `Cluster "${cluster.subdomainIdentifier}" has been created and is being provisioned.`
    )

    queue.dispatch(ProvisionHetznerBareMetalCluster, {
      clusterId: cluster.id,
    })

    return ctx.response.redirect().toRoute('clusters.index', {
      workspace: workspace.slug,
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
      if (cluster.robotCloudProviderId) {
        await queue.dispatch(ProvisionHetznerBareMetalCluster, { clusterId }, { attempts: 1 })
      } else {
        await queue.dispatch(ProvisionClusterJob, { clusterId }, { attempts: 1 })
      }

      return ctx.response.json({
        message: 'Cluster provisioning restarted from beginning',
        type: 'start',
      })
    }

    const failedStage = cluster.firstFailedStage

    if (!failedStage) {
      return ctx.response.status(400).json({
        error: 'No failed stages found to restart from',
      })
    }

    const job = this.getJobForStage(failedStage, cluster)

    await queue.dispatch(job, { clusterId })

    return ctx.response.json({
      message: `Cluster provisioning restarted from failed stage: ${failedStage}`,
      type: 'failed',
      stage: failedStage,
    })
  }

  public async destroy(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const clusterId = ctx.params.clusterId

    const cluster = await Cluster.query()
      .where('id', clusterId)
      .where('workspace_id', workspace.id)
      .firstOrFail()

    cluster.deletedAt = DateTime.now()
    await cluster.save()

    await queue.dispatch(DestroyClusterJob, { clusterId: cluster.id })

    return ctx.response.redirect().toRoute('workspace.cloud.clusters', {
      workspace: workspace.slug,
    })
  }

  private getJobForStage(stage: TerraformStage, cluster: Cluster) {
    // Handle Hetzner Robot (bare metal) stages separately
    if (cluster.robotCloudProviderId) {
      switch (stage) {
        case 'bare-metal-networking':
          return ProvisionHetznerBareMetalCluster
        case 'bare-metal-cloud-load-balancer':
          return ProvisionBareMetalCloudLoadBalancerJob
        case 'bare-metal-talos-image':
          return ProvisionBareMetalTalosImageJob
        case 'bare-metal-servers-bootstrap':
          return ProvisionBareMetalServersBootstrapJob
        default:
          throw new Error(`Unknown Hetzner Robot stage: ${stage}`)
      }
    }

    // Handle standard cloud provider stages
    switch (stage) {
      case 'network':
        return ProvisionNetworkJob
      case 'ssh-keys':
        return ProvisionSshKeysJob
      case 'load-balancers':
        return ProvisionLoadBalancersJob
      case 'servers':
        return ProvisionServersJob
      case 'volumes':
        return ProvisionVolumesJob
      case 'talos-image':
        return ProvisionTalosImageJob
      case 'kubernetes-config':
        return ProvisionKubernetesConfigJob
      case 'kubernetes-boot':
        return ProvisionKubernetesBootJob
      default:
        throw new Error(`Unknown stage: ${stage}`)
    }
  }
}
