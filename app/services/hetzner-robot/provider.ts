import logger from '@adonisjs/core/services/logger'
import {
  createHetznerRobotClient,
  HetznerRobotClient,
  HetznerRobotCredentials,
  toFormUrlEncoded,
} from './openapi/client.js'

class HetznerRobotCredentialsProvider {
  constructor(protected client: HetznerRobotClient) {}

  async validate() {
    const { data, error } = await this.client.GET('/server')

    if (!error && data) {
      return true
    }

    return false
  }
}

class HetznerRobotServerProvider {
  constructor(
    protected client: HetznerRobotClient,
    protected credentials: HetznerRobotCredentials
  ) {}

  async list() {
    const { data, error } = await this.client.GET('/server')

    if (!error && data) {
      const servers = data.map(({ server }) => server).filter((server) => server !== undefined)

      return servers
    }

    logger.error(error)

    return []
  }
}

class HetznerRobotBootProvider {
  constructor(
    protected client: HetznerRobotClient,
    protected credentials: HetznerRobotCredentials
  ) {}

  async activateRescue(
    serverNumber: number,
    os: 'linux' | 'vkvm',
    options?: {
      arch?: 32 | 64
      keyboard?: string
    }
  ) {
    const { data, error } = await this.client.POST('/boot/{server-number}/rescue', {
      params: {
        path: {
          'server-number': serverNumber,
        },
      },
      body: toFormUrlEncoded({
        os,
        arch: options?.arch,
        keyboard: options?.keyboard,
      }),
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }

  async deactivateRescue(serverNumber: number) {
    const { data, error } = await this.client.DELETE('/boot/{server-number}/rescue', {
      params: {
        path: {
          'server-number': serverNumber,
        },
      },
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }

  async getRescueOptions(serverNumber: number) {
    const { data, error } = await this.client.GET('/boot/{server-number}/rescue', {
      params: {
        path: {
          'server-number': serverNumber,
        },
      },
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }

  async getBootConfiguration(serverNumber: number) {
    const { data, error } = await this.client.GET('/boot/{server-number}', {
      params: {
        path: {
          'server-number': serverNumber,
        },
      },
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }

  async resetServer(
    serverNumber: number,
    type: 'sw' | 'hw' | 'man' | 'power' | 'power_long' = 'hw'
  ) {
    const { data, error } = await this.client.POST('/reset/{server-number}', {
      params: {
        path: {
          'server-number': serverNumber,
        },
      },
      body: toFormUrlEncoded({
        type,
      }),
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }
}

class HetznerRobotVSwitchProvider {
  constructor(
    protected client: HetznerRobotClient,
    protected credentials: HetznerRobotCredentials
  ) {}

  async list() {
    const { data, error } = await this.client.GET('/vswitch')

    if (!error && data) {
      return data
    }

    logger.error(error)

    return []
  }

  async create(name: string, vlan: number) {
    const { data, error } = await this.client.POST('/vswitch', {
      body: toFormUrlEncoded({
        name,
        vlan,
      }),
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }

  async get(vswitchId: number) {
    const { data, error } = await this.client.GET('/vswitch/{vswitch-id}', {
      params: {
        path: {
          'vswitch-id': vswitchId,
        },
      },
    })

    if (!error && data) {
      return data
    }

    logger.error(error)

    return null
  }

  async addServers(vswitchId: number, serverIdentifiers: string[]) {
    const { error } = await this.client.POST('/vswitch/{vswitch-id}/server', {
      params: {
        path: {
          'vswitch-id': vswitchId,
        },
      },
      body: toFormUrlEncoded({
        server: serverIdentifiers,
      }),
    })

    if (error) {
      logger.error(error)
      return false
    }

    return true
  }

  async removeServers(vswitchId: number, serverIdentifiers: string[]) {
    const { error } = await this.client.DELETE('/vswitch/{vswitch-id}/server', {
      params: {
        path: {
          'vswitch-id': vswitchId,
        },
      },
      body: toFormUrlEncoded({
        server: serverIdentifiers,
      }),
    })

    if (error) {
      logger.error(error)
      return false
    }

    return true
  }
}

export class HetznerRobotProvider {
  protected client: HetznerRobotClient

  constructor(protected credentials: HetznerRobotCredentials) {
    this.client = createHetznerRobotClient(this.credentials)
  }

  auth() {
    return new HetznerRobotCredentialsProvider(this.client)
  }

  servers() {
    return new HetznerRobotServerProvider(this.client, this.credentials)
  }

  vswitches() {
    return new HetznerRobotVSwitchProvider(this.client, this.credentials)
  }

  boot() {
    return new HetznerRobotBootProvider(this.client, this.credentials)
  }
}

export function hetznerRobot(credentials: HetznerRobotCredentials) {
  return new HetznerRobotProvider(credentials)
}
