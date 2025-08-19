import { OauthConfig } from '#config/oauth'
import { inject } from '@adonisjs/core'
import app from '@adonisjs/core/services/app'
import Kibaauth from '@kibamail/auth-sdk'

@inject()
export class OauthService {
  protected config: OauthConfig['oauth']
  constructor() {
    this.config = app.config.get<OauthConfig>('oauth').oauth
  }

  api() {
    return new Kibaauth()
      .clientId(this.config.clientId)
      .clientSecret(this.config.clientSecret)
      .callbackUrl(this.config.callbackUrl)
      .baseUrl(this.config.clientBaseUrl)
      .api()
  }

  accessToken(accessToken: string) {
    return new Kibaauth().baseUrl(this.config.clientBaseUrl).accessToken(accessToken).api()
  }
}
