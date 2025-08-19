import env from '#start/env'

/**
 * The oauth configuration settings are used by the Kibaauth server and package
 * to automate authentication handling.
 */
export const oauth = {
  clientId: env.get('OAUTH_CLIENT_ID'),
  clientSecret: env.get('OAUTH_CLIENT_SECRET'),
  callbackUrl: env.get('OAUTH_CALLBACK_URL'),
  clientBaseUrl: env.get('OAUTH_CLIENT_BASE_URL'),
}

export type OauthConfig = {
  oauth: {
  clientId: string
  clientSecret: string
  callbackUrl: string
  clientBaseUrl: string
}
}