import type { HttpContext } from '@adonisjs/core/http'

export default class LogoutController {
    async logout(ctx: HttpContext) {
        await ctx.auth.use('web').logout()

        return ctx.response.redirect('/')
    }

}
