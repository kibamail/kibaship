/*
|--------------------------------------------------------------------------
| Ally Oauth driver
|--------------------------------------------------------------------------
|
| Make sure you through the code and comments properly and make necessary
| changes as per the requirements of your implementation.
|
*/

/**
|--------------------------------------------------------------------------
 *  Search keyword "DigitalOcean" and replace it with a meaningful name
|--------------------------------------------------------------------------
 */

import { Oauth2Driver, RedirectRequest } from '@adonisjs/ally'
import type { HttpContext } from '@adonisjs/core/http'
import type { AllyDriverContract, AllyUserContract, ApiRequestContract, LiteralStringUnion } from '@adonisjs/ally/types'

/**
 *
 * Access token returned by your driver implementation. An access
 * token must have "token" and "type" properties and you may
 * define additional properties (if needed)
 */
export type DigitalOceanAccessToken = {
    token: string
    type: 'bearer'
}

/**
 * Scopes accepted by the driver implementation.
 */
export type DigitalOceanScopes =
    | 'droplets:read'
    | 'droplets:write'
    | 'droplets:update'
    | 'droplets:delete'
    | 'ssh_keys:read'
    | 'ssh_keys:write'
    | 'ssh_keys:update'
    | 'ssh_keys:delete'
    | 'load_balancers:read'
    | 'load_balancers:write'
    | 'load_balancers:update'
    | 'load_balancers:delete'
    | 'volumes:read'
    | 'volumes:write'
    | 'volumes:update'
    | 'volumes:delete'

/**
 * The configuration accepted by the driver implementation.
 */
export type DigitalOceanConfig = {
    clientId: string
    clientSecret: string
    callbackUrl: string
    authorizeUrl?: string
    accessTokenUrl?: string
    userInfoUrl?: string
    scopes?: LiteralStringUnion<DigitalOceanScopes>[]
}

/**
 * Driver implementation. It is mostly configuration driven except the API call
 * to get user info.
 */
export class DigitalOcean
    extends Oauth2Driver<DigitalOceanAccessToken, DigitalOceanScopes>
    implements AllyDriverContract<DigitalOceanAccessToken, DigitalOceanScopes> {
    /**
     * The URL for the redirect request. The user will be redirected on this page
     * to authorize the request.
     *
     * Do not define query strings in this URL.
     */
    protected authorizeUrl = 'https://cloud.digitalocean.com/v1/oauth/authorize'

    /**
     * The URL to hit to exchange the authorization code for the access token
     *
     * Do not define query strings in this URL.
     */
    protected accessTokenUrl = 'https://cloud.digitalocean.com/v1/oauth/token'

    /**
     * The URL to hit to get the user details
     *
     * Do not define query strings in this URL.
     */
    protected userInfoUrl = 'https://cloud.digitalocean.com/v1/oauth/'

    /**
     * The param name for the authorization code. Read the documentation of your oauth
     * provider and update the param name to match the query string field name in
     * which the oauth provider sends the authorization_code post redirect.
     */
    protected codeParamName = 'code'

    /**
     * The param name for the error. Read the documentation of your oauth provider and update
     * the param name to match the query string field name in which the oauth provider sends
     * the error post redirect
     */
    protected errorParamName = 'error'

    /**
     * Cookie name for storing the CSRF token. Make sure it is always unique. So a better
     * approach is to prefix the oauth provider name to `oauth_state` value. For example:
     * For example: "facebook_oauth_state"
     */
    protected stateCookieName = 'digital_ocean_oauth_state'

    /**
     * Parameter name to be used for sending and receiving the state from.
     * Read the documentation of your oauth provider and update the param
     * name to match the query string used by the provider for exchanging
     * the state.
     */
    protected stateParamName = 'state'

    /**
     * Parameter name for sending the scopes to the oauth provider.
     */
    protected scopeParamName = 'scope'

    /**
     * The separator indentifier for defining multiple scopes
     */
    protected scopesSeparator = ','

    constructor(
        ctx: HttpContext,
        public config: DigitalOceanConfig
    ) {
        super(ctx, config)

        /**
         * Extremely important to call the following method to clear the
         * state set by the redirect request.
         *
         * DO NOT REMOVE THE FOLLOWING LINE
         */
        this.loadState()
    }

    /**
     * Optionally configure the authorization redirect request. The actual request
     * is made by the base implementation of "Oauth2" driver and this is a
     * hook to pre-configure the request.
     */
    protected configureRedirectRequest(request: RedirectRequest<DigitalOceanScopes>) {
        /**
         * Define user defined scopes or the default one's
         */
        request.scopes(this.config.scopes || [])
    }

    /**
     * Optionally configure the access token request. The actual request is made by
     * the base implementation of "Oauth2" driver and this is a hook to pre-configure
     * the request
     */
    // protected configureAccessTokenRequest(request: ApiRequest) {}

    /**
     * Update the implementation to tell if the error received during redirect
     * means "ACCESS DENIED".
     */
    accessDenied() {
        return this.ctx.request.input('error') === 'user_denied'
    }

    /**
     * Get the user details by query the provider API. This method must return
     * the access token and the user details both. Checkout the google
     * implementation for same.
     *
     * https://github.com/adonisjs/ally/blob/develop/src/Drivers/Google/index.ts#L191-L199
     */
    async user(
        callback?: (request: ApiRequestContract) => void
    ): Promise<AllyUserContract<DigitalOceanAccessToken>> {
        const accessToken = await this.accessToken()
        const request = this.httpClient(this.config.userInfoUrl || this.userInfoUrl)

        /**
         * Allow end user to configure the request. This should be called after your custom
         * configuration, so that the user can override them (if needed)
         */
        if (typeof callback === 'function') {
            callback(request)
        }

        /**
         * Write your implementation details here.
         */
    }

    async userFromToken(
        accessToken: string,
        callback?: (request: ApiRequestContract) => void
    ): Promise<AllyUserContract<{ token: string; type: 'bearer' }>> {
        const request = this.httpClient(this.config.userInfoUrl || this.userInfoUrl)

        /**
         * Allow end user to configure the request. This should be called after your custom
         * configuration, so that the user can override them (if needed)
         */
        if (typeof callback === 'function') {
            callback(request)
        }

        /**
         * Write your implementation details here
         */
    }
}

/**
 * The factory function to reference the driver implementation
 * inside the "config/ally.ts" file.
 */
export function DigitalOceanService(config: DigitalOceanConfig): (ctx: HttpContext) => DigitalOcean {
    return (ctx) => new DigitalOcean(ctx, config)
}
