import env from '#start/env'

const githubConfig = {
  /*
  |--------------------------------------------------------------------------
  | GitHub App configuration
  |--------------------------------------------------------------------------
  */
  app: {
    id: env.get('SOURCE_GITHUB_APP_ID'),
    name: env.get('SOURCE_GITHUB_APP_NAME'),
    secret: env.get('SOURCE_GITHUB_APP_SECRET'),
    privateKey: env.get('SOURCE_GITHUB_APP_PRIVATE_KEY'),
  },

  /*
  |--------------------------------------------------------------------------
  | GitHub App callback and webhook URLs
  |--------------------------------------------------------------------------
  */
  urls: {
    callbackUrl: env.get('SOURCE_GITHUB_CALLBACK_URL'),
    webhookUrl: env.get('SOURCE_GITHUB_WEBHOOK_URL'),
  },

  /*
  |--------------------------------------------------------------------------
  | GitHub webhook configuration
  |--------------------------------------------------------------------------
  */
  webhooks: {
    secret: env.get('SOURCE_GITHUB_APP_WEBHOOKS_SECRET'),
  },
}

export default githubConfig

export interface GithubConfig {
  app: {
    name: string
    id: string
    secret: string
    privateKey: string
  }
  urls: {
    callbackUrl: string
    webhookUrl: string
  }
  webhooks: {
    secret: string
  }
}
