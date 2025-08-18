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

router.on('/').renderInertia('home')

router
  .group(() => [
    router.get('/', [DashboardController, 'index']),
    router.get('/:workspace', [DashboardController, 'show']),
    router.post('/workspaces', [WorkspacesController, 'store']),
    router.get('/:workspace/p/:project', [ProjectsController, 'show']),
  ])
  .prefix('/w')
  .use(middleware.auth())

router
  .group(() => [
    router.get('/source-code-providers', [SourceProvidersController, 'index']),
    router.get('/source-code-providers/:sourceCodeProviderId', [SourceProvidersController, 'show']),
    router.get('/:provider/redirect', [SourceProvidersController, 'redirect']),
    router.get('/:provider/callback', [SourceProvidersController, 'callback']),
  ])
  .prefix('/connections')
  .use(middleware.auth())

router
  .group(() => [router.post('/', [ApplicationController, 'store'])])
  .prefix('/w/applications')
  .use(middleware.auth())

router.get('/auth/redirect', [Oauth2Controller, 'redirect']).use(middleware.guest())
router.get('/auth/callback', [Oauth2Controller, 'callback']).use(middleware.guest())
