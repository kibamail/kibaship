/*
|--------------------------------------------------------------------------
| Environment variables service
|--------------------------------------------------------------------------
|
| The `Env.create` method creates an instance of the Env service. The
| service validates the environment variables and also cast values
| to JavaScript data types.
|
*/

import { Env } from '@adonisjs/core/env'

export default await Env.create(new URL('../', import.meta.url), {
  NODE_ENV: Env.schema.enum(['development', 'production', 'test'] as const),
  APP_URL: Env.schema.string({ format: 'url', tld: false }),
  PORT: Env.schema.number(),
  APP_KEY: Env.schema.string(),
  HOST: Env.schema.string({ format: 'host' }),
  LOG_LEVEL: Env.schema.string(),

  /*
  |----------------------------------------------------------
  | Variables for configuring session package
  |----------------------------------------------------------
  */
  SESSION_DRIVER: Env.schema.enum(['cookie', 'memory'] as const),

  /*
  |----------------------------------------------------------
  | Variables for configuring database connection
  |----------------------------------------------------------
  */
  DB_HOST: Env.schema.string({ format: 'host' }),
  DB_PORT: Env.schema.number(),
  DB_USER: Env.schema.string(),
  DB_PASSWORD: Env.schema.string.optional(),
  DB_DATABASE: Env.schema.string(),

  /*
  |----------------------------------------------------------
  | Redis server variables
  |----------------------------------------------------------
  */
  REDIS_HOST: Env.schema.string({ format: 'host' }),
  REDIS_PORT: Env.schema.number(),
  REDIS_PASSWORD: Env.schema.string.optional(),

  /*
  |----------------------------------------------------------
  | HashiCorp Vault variables
  |----------------------------------------------------------
  */
  VAULT_ADDR: Env.schema.string({ format: 'url', tld: false }),
  VAULT_READ_ROLE_ID: Env.schema.string(),
  VAULT_READ_SECRET_ID: Env.schema.string(),
  VAULT_WRITE_ROLE_ID: Env.schema.string(),
  VAULT_WRITE_SECRET_ID: Env.schema.string(),

  /*
  |----------------------------------------------------------
  | Ngrok tunnel variables
  |----------------------------------------------------------
  */
  NGROK_AUTHTOKEN: Env.schema.string(),

  /*
  |----------------------------------------------------------
  | GitHub App integration variables
  |----------------------------------------------------------
  */
  SOURCE_GITHUB_APP_NAME: Env.schema.string(),
  SOURCE_GITHUB_APP_ID: Env.schema.string(),
  SOURCE_GITHUB_CALLBACK_URL: Env.schema.string({ format: 'url', tld: false }),
  SOURCE_GITHUB_WEBHOOK_URL: Env.schema.string({ format: 'url', tld: false }),
  SOURCE_GITHUB_APP_WEBHOOKS_SECRET: Env.schema.string(),
  SOURCE_GITHUB_APP_SECRET: Env.schema.string(),
  SOURCE_GITHUB_APP_PRIVATE_KEY: Env.schema.string(),

  /*
  |----------------------------------------------------------
  | Variables for @rlanz/bull-queue
  |----------------------------------------------------------
  */
  QUEUE_REDIS_HOST: Env.schema.string({ format: 'host' }),
  QUEUE_REDIS_PORT: Env.schema.number(),
  QUEUE_REDIS_PASSWORD: Env.schema.string.optional(),

  /*
  |----------------------------------------------------------
  | Variables for configuring the drive package
  |----------------------------------------------------------
  */
  DRIVE_DISK: Env.schema.enum(['fs'] as const),

  /*
  |----------------------------------------------------------
  | Variables for configuring the remote terraform state drive
  |----------------------------------------------------------
  */
  S3_ACCESS_KEY: Env.schema.string(),
  S3_ACCESS_SECRET: Env.schema.string(),
  S3_ENDPOINT: Env.schema.string({ format: 'url' }),
  S3_REGION: Env.schema.string(),
  S3_BUCKET: Env.schema.string(),

  /*
  |----------------------------------------------------------
  | Digital Ocean OAuth App variables
  |----------------------------------------------------------
  */
  DIGITAL_OCEAN_APP_CLIENT_ID: Env.schema.string(),
  DIGITAL_OCEAN_APP_CLIENT_SECRET: Env.schema.string(),
  DIGITAL_OCEAN_APP_CALLBACK_URL: Env.schema.string({ format: 'url', tld: false }),

  /*
  |----------------------------------------------------------
  | Talos configuration
  |----------------------------------------------------------
  */
  TALOS_VERSION: Env.schema.string(),
  TALOS_FACTORY_HASH_DIGITAL_OCEAN: Env.schema.string(),
  TALOS_FACTORY_HASH_HETZNER: Env.schema.string(),
})
