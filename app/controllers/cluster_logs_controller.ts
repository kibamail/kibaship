import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from './Base/base_controller.js'
import { ClusterLogsService } from '#services/redis/cluster_logs_service'
import Cluster from '#models/cluster'

export default class ClusterLogsController extends BaseController {
  public async show(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)
    const clusterId = ctx.params.clusterId

    await Cluster.query()
      .where('id', clusterId)
      .where('workspace_id', workspace.id)
      .firstOrFail()

    const logs = await ClusterLogsService.getLogsForCluster(clusterId)

    return ctx.response.json({
      cluster_id: clusterId,
      logs
    })
  }


}
