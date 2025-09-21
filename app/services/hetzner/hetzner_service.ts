import axios, { AxiosInstance } from 'axios'
import { $trycatch } from '@tszen/trycatch'

export class HetznerService {
  private client: AxiosInstance

  constructor(apiToken: string) {
    this.client = axios.create({
      baseURL: 'https://api.hetzner.cloud/v1',
      headers: {
        'Authorization': `Bearer ${apiToken}`,
        'Content-Type': 'application/json',
      },
      timeout: 30000,
    })
  }

  sshkeys() {
    const client = this.client
    return {
      async create(name: string, public_key: string) {
        return $trycatch(() =>
          client.post<{
            ssh_key: {
              id: string
            }
          }>('/ssh_keys', {
            name,
            public_key,
          })
        )
      },
      async delete(sshKeyId: string) {
        return $trycatch(() => client.delete(`/ssh_keys/${sshKeyId}`))
      },
    }
  }

  images() {
    const client = this.client
    return {
      async get(type?: 'snapshot' | 'app' | 'system' | 'backup') {
        return $trycatch(async () => {
          const response = await client.get<{
            images: {
              id: number
              type: typeof type
              name: string
              labels: Record<string, string>
            }[]
          }>('/images', {
            params: {
              type,
            },
          })

          return response.data
        })
      },
    }
  }
}
