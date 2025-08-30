import env from '#start/env'
import Axios from 'axios'
import { $trycatch } from '@tszen/trycatch'
import type { HttpContext } from '@adonisjs/core/http'
import { BaseController } from '#controllers/Base/base_controller'
import CloudProvider from '#models/cloud_provider'

export default class DigitalOceanController extends BaseController {

    protected authorizeUrl = 'https://cloud.digitalocean.com/v1/oauth/authorize'

    public async redirect(ctx: HttpContext) {
        const scope = [
            'droplet:read',
            'droplet:create',
            'droplet:update',
            'droplet:delete',
            'ssh_key:read',
            'ssh_key:create',
            'ssh_key:update',
            'ssh_key:delete',
            'load_balancer:read',
            'load_balancer:create',
            'load_balancer:update',
            'load_balancer:delete',
            'block_storage:read',
            'block_storage:create',
            'block_storage:delete',
            'tag:create',
            'tag:delete',
            'tag:read',
            'vpc:read',
            'vpc:create',
            'vpc:update',
            'vpc:delete',
            'block_storage_action:create',
        ].join(',')

        const url = new URL(this.authorizeUrl)

        url.searchParams.set('client_id', env.get('DIGITAL_OCEAN_APP_CLIENT_ID'))
        url.searchParams.set('redirect_uri', env.get('DIGITAL_OCEAN_APP_CALLBACK_URL'))
        url.searchParams.set('scope', scope)
        url.searchParams.set('response_type', 'code')

        return ctx.response.redirect(
            url.toString()
        )
    }

    public async callback(ctx: HttpContext) {

        console.log(ctx.request.all())
        const workspace = await this.workspace(ctx)

        const [response, error] = await $trycatch(() => {
            return Axios.post<{
                access_token: string
                token_type: string
                expires_in: number
                refresh_token: string
                info: {
                    name: string
                    email: string
                    uuid: string
                    team_uuid: string
                    team_name: string
                }
            }>('https://cloud.digitalocean.com/v1/oauth/token', {}, {
                params: {
                    grant_type: 'authorization_code',
                    code: ctx.request.qs()?.code,
                    client_id: env.get('DIGITAL_OCEAN_APP_CLIENT_ID'),
                    client_secret: env.get('DIGITAL_OCEAN_APP_CLIENT_SECRET'),
                    redirect_uri: env.get('DIGITAL_OCEAN_APP_CALLBACK_URL'),
                }
            })
        })

        console.error((error?.cause as any)?.response)

        if (error) {
            return ctx.response.redirect(`/w/${workspace.slug}?error=Failed to authenticate with Digital Ocean. Please try again.`)
        }

        const cloudProvider = await CloudProvider.create({
            name: response?.data.info.team_name,
            type: 'digital_ocean',
            workspaceId: workspace.id,
            credentials: {
                access_token: response?.data.access_token,
                refresh_token: response?.data.refresh_token,
            },
        })

        return ctx.response.redirect(`/w/${workspace.slug}/clusters?success=Successfully connected digital ocean provider: ${cloudProvider?.name}.`)
    }
}
