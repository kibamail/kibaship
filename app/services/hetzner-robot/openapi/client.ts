import createClient from 'openapi-fetch'

import type { paths } from './schema.js'

export interface HetznerRobotCredentials {
  username: string
  password: string
}

/**
 * Helper to convert a data object to URLSearchParams for form-urlencoded requests
 */
export function toFormUrlEncoded<T extends Record<string, string | number | string[] | undefined>>(
  data: T
) {
  const params = new URLSearchParams()

  for (const [key, value] of Object.entries(data)) {
    if (value !== undefined && value !== null) {
      if (Array.isArray(value)) {
        value.forEach((v) => params.append(`${key}[]`, String(v)))
        continue
      }

      params.append(key, String(value))
    }
  }

  console.log('@@@@@@@@@@@@@ params', params.toString())

  return params.toString() as unknown as T
}

export function createHetznerRobotClient(credentials: HetznerRobotCredentials) {
  const client = createClient<paths>({
    baseUrl: 'https://robot-ws.your-server.de',
    headers: {
      'Authorization': `Basic ${Buffer.from(
        `${credentials.username}:${credentials.password}`
      ).toString('base64')}`,
      'Content-Type': 'application/x-www-form-urlencoded',
    },
  })

  return client
}

export type HetznerRobotClient = ReturnType<typeof createHetznerRobotClient>
