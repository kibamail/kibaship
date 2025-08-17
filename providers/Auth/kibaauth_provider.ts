import { Kibaauth, ApiClient } from '@kibamail/auth-sdk'
import type { ApplicationService } from '@adonisjs/core/types'

declare module '@adonisjs/core/types' {
  export interface ContainerBindings {
    'auth.kibaauth': {
      api: () => ApiClient
      accessToken: (accessToken: string) => ApiClient
    }
  }
}

export default class KibaauthProvider {
  constructor(protected app: ApplicationService) {}

  /**
   * Register bindings to the container
   */
  register() {
    this.app.container.bind('auth.kibaauth', () => {
      const { oauth } = this.app.config.get<{
        oauth: {
          clientId: string
          clientSecret: string
          callbackUrl: string
          clientBaseUrl: string
        }
      }>('oauth')

      return {
        api() {
          return new Kibaauth()
            .clientId(oauth.clientId)
            .clientSecret(oauth.clientSecret)
            .callbackUrl(oauth.callbackUrl)
            .baseUrl(oauth.clientBaseUrl)
            .api()
        },
        accessToken(accessToken: string) {
          return new Kibaauth().baseUrl(oauth.clientBaseUrl).accessToken(accessToken).api()
        },
      }
    })
  }

  /**
   * The container bindings have booted
   */
  async boot() {}

  /**
   * The application has been booted
   */
  async start() {}

  /**
   * The process has been started
   */
  async ready() {}

  /**
   * Preparing to shutdown the app
   */
  async shutdown() {}
}
