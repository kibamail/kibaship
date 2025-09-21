import { BaseController } from '#controllers/Base/base_controller'
import type { HttpContext } from '@adonisjs/core/http'

export default class IntegrationsController extends BaseController {
  public async index(ctx: HttpContext) {
    return ctx.inertia.render('integrations', await this.pageProps(ctx))
  }
}

