import redis from '@adonisjs/redis/services/main'
import type { components } from '#services/hetzner-robot/openapi/schema'

/**
 * Define your cache namespaces and their item schemas here
 */
export interface CacheSchemas {
  'hetzner-robot:servers': {
    items: {
      [serverId: string]: NonNullable<components['schemas']['ServerBasic']['server']>
    }
  }
  'hetzner-robot': {
    items: {
      [vswitchId: string]:
        | NonNullable<components['schemas']['ServerBasic']['server']>[]
        | components['schemas']['VSwitchBasic'][]
    }
  }
}

/**
 * Cache item accessor for reading and writing cached data
 */
class CacheItem<T> {
  constructor(
    private namespace: string,
    private itemName: string
  ) {}

  /**
   * Get the Redis key for this cache item
   */
  private getKey(): string {
    return `cache:${this.namespace}:${this.itemName}`
  }

  /**
   * Write data to cache
   * @param data The data to cache
   * @param ttl Time to live in seconds (optional)
   */
  async write(data: T, ttl?: number): Promise<void> {
    const key = this.getKey()
    const serialized = JSON.stringify(data)

    if (ttl) {
      await redis.setex(key, ttl, serialized)
    } else {
      await redis.set(key, serialized)
    }
  }

  /**
   * Read data from cache
   * @returns The cached data or null if not found
   */
  async read(): Promise<T | null> {
    const key = this.getKey()
    const data = await redis.get(key)

    if (!data) {
      return null
    }

    return JSON.parse(data) as T
  }

  /**
   * Delete this cache item
   */
  async delete(): Promise<void> {
    const key = this.getKey()
    await redis.del(key)
  }

  /**
   * Check if this cache item exists
   */
  async exists(): Promise<boolean> {
    const key = this.getKey()
    const result = await redis.exists(key)
    return result === 1
  }

  /**
   * Get the TTL (time to live) of this cache item in seconds
   * @returns TTL in seconds, -1 if no expiry, -2 if key doesn't exist
   */
  async ttl(): Promise<number> {
    const key = this.getKey()
    return await redis.ttl(key)
  }
}

/**
 * Cache namespace accessor
 */
class CacheNamespace<TNamespace extends keyof CacheSchemas> {
  constructor(private namespace: TNamespace) {}

  /**
   * Access a specific item in this cache namespace
   * @param itemName The name of the item to access
   */
  item<TItemName extends keyof CacheSchemas[TNamespace]['items']>(
    itemName: TItemName
  ): CacheItem<CacheSchemas[TNamespace]['items'][TItemName]> {
    return new CacheItem(this.namespace, itemName as string)
  }

  /**
   * Delete all items in this namespace
   */
  async flush(): Promise<void> {
    const pattern = `cache:${this.namespace}:*`
    const keys = await redis.keys(pattern)

    if (keys.length > 0) {
      await redis.del(...keys)
    }
  }

  /**
   * Get all keys in this namespace
   */
  async keys(): Promise<string[]> {
    const pattern = `cache:${this.namespace}:*`
    return await redis.keys(pattern)
  }
}

/**
 * Main cache function for accessing cache namespaces
 * @param namespace The cache namespace to access
 * @returns A CacheNamespace instance
 *
 * @example
 * // Write to cache
 * await cache('hetzner-robot:servers').item('server-123').write({
 *   id: 123,
 *   name: 'web-server',
 *   product: 'EX42',
 *   dc: 'FSN1-DC14',
 *   traffic: 'unlimited',
 *   status: 'ready',
 *   cancelled: false,
 *   paid_until: '2024-12-31',
 *   ip: ['192.0.2.1'],
 *   subnet: []
 * }, 3600)
 *
 * // Read from cache
 * const server = await cache('hetzner-robot:servers').item('server-123').read()
 *
 * // Delete from cache
 * await cache('hetzner-robot:servers').item('server-123').delete()
 */
export function cache<TNamespace extends keyof CacheSchemas>(
  namespace: TNamespace
): CacheNamespace<TNamespace> {
  return new CacheNamespace(namespace)
}

/**
 * Utility function to clear all cache data
 */
export async function clearAllCache(): Promise<void> {
  const pattern = 'cache:*'
  const keys = await redis.keys(pattern)

  if (keys.length > 0) {
    await redis.del(...keys)
  }
}
