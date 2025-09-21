/*
|--------------------------------------------------------------------------
| Routes file
|--------------------------------------------------------------------------
|
| The routes file is used for defining the HTTP routes.
|
*/

import RegisterController from '#controllers/Auth/register_controller'
import LoginController from '#controllers/Auth/login_controller'

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
import ClusterDnsVerifyController from '#controllers/clusters/cluster_dns_verify_controller'
import ByocController from '#controllers/clusters/byoc_controller'
import LogoutController from '#controllers/Auth/logout_controller'

import CloudController from '#controllers/Cloud/cloud_controller'
import IntegrationsController from '#controllers/Integrations/integrations_controller'
import MonitoringController from '#controllers/Monitoring/monitoring_controller'

router.on('/').renderInertia('home')

router
  .group(() => [
    router.get('/', [DashboardController, 'index']),
    router.get('/:workspace', [DashboardController, 'show']),
    router.post('/workspaces', [WorkspacesController, 'store']),
    router.get('/:workspace/dashboard', [DashboardController, 'show']).as('workspace.dashboard'),
    router.post('/:workspace/projects', [ProjectsController, 'store']).as('projects.store'),

    router.get('/:workspace/p/:project', [ProjectsController, 'show']),
    router.get('/:workspace/clusters', [ClustersController, 'index']).as('clusters.index'),
    router.post('/:workspace/clusters', [ClustersController, 'store']).as('clusters.store'),
    router
      .post('/:workspace/clusters/bring-your-own', [ByocController, 'store'])
      .as('clusters.bring-your-own.store'),
    router.get('/:workspace/clusters/:clusterId', [ClustersController, 'show']).as('clusters.show'),
    router
      .get('/:workspace/clusters/:clusterId/logs', [ClusterLogsController, 'show'])
      .as('clusters.logs'),
    router
      .post('/:workspace/clusters/:clusterId/restart', [ClustersController, 'restart'])
      .as('clusters.restart'),
    router.delete('/:workspace/clusters/:clusterId', [ClustersController, 'destroy']),
    router.post('/:workspace/clusters/providers', [CloudProvidersController, 'store']),
    router.post('/:workspace/clusters/:clusterId/dns/verify', [
      ClusterDnsVerifyController,
      'index',
    ]),
  ])
  .prefix('/w')
  .use(middleware.auth())

router
  .group(() => [
    router
      .get('/applications', [ApplicationController, 'index'])
      .as('workspace.applications.index'),
    router
      .get('/applications/:application', [ApplicationController, 'index'])
      .as('workspace.applications.show'),
    router.get('/cloud/clusters', [CloudController, 'clusters']).as('workspace.cloud.clusters'),
    router.get('/cloud/providers', [CloudController, 'providers']).as('workspace.cloud.providers'),
    router
      .get('/integrations', [IntegrationsController, 'index'])
      .as('workspace.integrations.index'),
    router.get('/monitoring', [MonitoringController, 'index']).as('workspace.monitoring.index'),
  ])
  .prefix('/w/:workspace')
  .use(middleware.auth())

router
  .group(() => [
    router.get('/source-code-providers', [SourceProvidersController, 'index']),
    router.get('/source-code-providers/:sourceCodeProviderId', [SourceProvidersController, 'show']),
    router.get('/:provider/redirect', [SourceProvidersController, 'redirect']),
    router.get('/:provider/callback', [SourceProvidersController, 'callback']),
    router.get('/cloud-providers/digital-ocean/redirect', [DigitalOceanController, 'redirect']),
    router.get('/cloud-providers/digital-ocean/callback', [DigitalOceanController, 'callback']),
    router.delete('/cloud-providers/:cloudProvider', [CloudProvidersController, 'destroy']),
  ])
  .prefix('/connections')
  .use(middleware.auth())

router.get('/provision', async ({ response }) => {
  let clusterFirst = await Cluster.firstOrFail()

  const cluster = await Cluster.complete(clusterFirst?.id)

  await queue.dispatch(ProvisionClusterJob, {
    clusterId: cluster?.id as string,
  })

  return response.json({ cluster })
})

router
  .group(() => [router.post('/', [ApplicationController, 'store'])])
  .prefix('/w/applications')
  .use(middleware.auth())

router.group(() => [router.post('/w/logout', [LogoutController, 'logout'])]).use(middleware.auth())

router.get('/auth/register', [RegisterController, 'show']).use(middleware.guest())
router.post('/auth/register', [RegisterController, 'store']).use(middleware.guest())
router.get('/auth/login', [LoginController, 'show']).use(middleware.guest())
router.post('/auth/login', [LoginController, 'store']).use(middleware.guest())
