import { VaultConfig } from '#config/vault'
import app from '@adonisjs/core/services/app'
import createClient from 'node-vault'

export class VaultService {
  protected workspaceId?: string
  protected config: VaultConfig
  protected accessTokens = {
    reads: {
      content: '',
      expiresAt: new Date(),
    },
    writes: {
      content: '',
      expiresAt: new Date(),
    },
  }

  protected client: ReturnType<typeof createClient>

  constructor() {
    this.config = app.config.get<VaultConfig>('vault')

    this.client = createClient({
      apiVersion: 'v1',
      endpoint: this.config.connection.address,
      token: this.config.connection.token,
    })
  }

  workspace(id: string) {
    this.workspaceId = id

    return this
  }

  public async read(_name: string) {
    await this.requestReadAccessToken()
  }

  public async write(_name: string, _data: Record<string, string>) {
    await this.requestWriteAccessToken()
  }

  private async requestReadAccessToken() {
    if (this.accessTokens.reads.content && this.accessTokens.reads.expiresAt > new Date()) {
      return
    }

    const response = await this.client.approleLogin({
      role_id: this.config.readRole.roleId,
      secret_id: this.config.writeRole.secretId,
    })

    console.log({ response })
  }

  private async requestWriteAccessToken() {
    if (this.accessTokens.writes.content && this.accessTokens.writes.expiresAt > new Date()) {
      return
    }

    const response = await this.client.approleLogin({
      role_id: this.config.writeRole.roleId,
      secret_id: this.config.writeRole.secretId,
    })

    console.log({ response })
  }
}
