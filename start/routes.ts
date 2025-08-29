/*
|--------------------------------------------------------------------------
| Routes file
|--------------------------------------------------------------------------
|
| The routes file is used for defining the HTTP routes.
|
*/

import Oauth2Controller from '#controllers/Auth/oauth_2_controller'
import router from '@adonisjs/core/services/router'
import { middleware } from './kernel.js'
import DashboardController from '#controllers/Dashboard/dashboard_controller'
import WorkspacesController from '#controllers/Workspaces/workspaces_controller'
import ApplicationController from '#controllers/applications/application_controller'
import SourceProvidersController from '#controllers/Connections/source_providers_controller'
import ProjectsController from '#controllers/Projects/projects_controller'
import ClustersController from '#controllers/clusters_controller'
import CloudProvidersController from '#controllers/cloud_providers_controller'
import ClusterLogsController from '#controllers/cluster_logs_controller'
import Cluster from '#models/cluster'
import queue from '@rlanz/bull-queue/services/main'
import ProvisionClusterJob from '#jobs/clusters/provision_cluster_job'
import DigitalOceanController from '#controllers/cloud_providers/digital_ocean_controller'

router.on('/').renderInertia('home')

router
  .group(() => [
    router.get('/', [DashboardController, 'index']),
    router.get('/:workspace', [DashboardController, 'show']),
    router.post('/workspaces', [WorkspacesController, 'store']),
    router.get('/:workspace/p/:project', [ProjectsController, 'show']),
    router.get('/:workspace/clusters', [ClustersController, 'index']).as('clusters.index'),
    router.post('/:workspace/clusters', [ClustersController, 'store']).as('clusters.store'),
    router.get('/:workspace/clusters/:clusterId', [ClustersController, 'show']).as('clusters.show'),
    router.get('/:workspace/clusters/:clusterId/logs', [ClusterLogsController, 'show']).as('clusters.logs'),
    router.post('/:workspace/clusters/:clusterId/restart', [ClustersController, 'restart']).as('clusters.restart'),
    router.post('/:workspace/clusters/providers', [CloudProvidersController, 'store']),
  ])
  .prefix('/w')
  .use(middleware.auth())

router
  .group(() => [
    router.get('/source-code-providers', [SourceProvidersController, 'index']),
    router.get('/source-code-providers/:sourceCodeProviderId', [SourceProvidersController, 'show']),
    router.get('/:provider/redirect', [SourceProvidersController, 'redirect']),
    router.get('/:provider/callback', [SourceProvidersController, 'callback']),
    router.get('/cloud-providers/digital-ocean/redirect', [DigitalOceanController, 'redirect']),
    router.get('/cloud-providers/digital-ocean/callback', [DigitalOceanController, 'callback'])
  ])
  .prefix('/connections')
  .use(middleware.auth())

router.get('/provision', async ({ response }) => {
  let clusterFirst = await Cluster.firstOrFail()

  const cluster = await Cluster.complete(clusterFirst?.id)

  await queue.dispatch(ProvisionClusterJob, {
    clusterId: cluster?.id as string
  })

  return response.json({ cluster })
})

router
  .group(() => [router.post('/', [ApplicationController, 'store'])])
  .prefix('/w/applications')
  .use(middleware.auth())

router.get('/auth/redirect', [Oauth2Controller, 'redirect']).use(middleware.guest())
router.get('/auth/callback', [Oauth2Controller, 'callback']).use(middleware.guest())
