import { VaultConfig } from '#config/vault'
import app from '@adonisjs/core/services/app'
import createClient from 'node-vault'

class VaultService {
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
    })
  }

  public async read(name: string) {
    await this.requestReadAccessToken()

    return this.client.read(`/secrets/data/workspaces/${name}`, {
      token: this.accessTokens.reads.content,
    })
  }

  public async write(name: string, data: Record<string, string>) {
    await this.requestWriteAccessToken()

    return this.client.write(`/secrets/data/workspaces/${name}`, data, {
      token: this.accessTokens.writes.content,
    })
  }

  private async requestReadAccessToken() {
    if (this.accessTokens.reads.content && this.accessTokens.reads.expiresAt > new Date()) {
      return
    }

    const auth = await this.client.approleLogin({
      role_id: this.config.readRole.roleId,
      secret_id: this.config.readRole.secretId,
    })

    this.accessTokens.reads['content'] = auth.auth.client_token
    this.accessTokens.reads['expiresAt'] = new Date(Date.now() + auth.auth.lease_duration * 1000)
  }

  private async requestWriteAccessToken() {
    if (this.accessTokens.writes.content && this.accessTokens.writes.expiresAt > new Date()) {
      return
    }

    const auth = await this.client.approleLogin({
      role_id: this.config.writeRole.roleId,
      secret_id: this.config.writeRole.secretId,
    })

    this.accessTokens.writes['content'] = auth.auth.client_token
    this.accessTokens.writes['expiresAt'] = new Date(Date.now() + auth.auth.lease_duration * 1000)

  }
}

export const vault = new VaultService()
