import { ClusterLogConsumer } from './cluster_log_consumer.js'
import logger from '@adonisjs/core/services/logger'
import { ClusterLogEntry } from './cluster_logs_service.js'

class RedisStreamManager {
  private consumers = new Map<string, ClusterLogConsumer>()
  private connectionCounts = new Map<string, number>()

  async ensureConsumer(clusterId: string, onConsumerLogUpdated: (log: ClusterLogEntry) => void): Promise<void> {
    const currentCount = this.connectionCounts.get(clusterId) || 0
    this.connectionCounts.set(clusterId, currentCount + 1)

    if (!this.consumers.has(clusterId)) {
      logger.info(`Creating new Redis stream consumer for cluster: ${clusterId}`)

      const consumer = new ClusterLogConsumer(clusterId, onConsumerLogUpdated)
      this.consumers.set(clusterId, consumer)
      await consumer.start()
    } else {
      logger.debug(`Reusing existing Redis stream consumer for cluster: ${clusterId}`)
    }
  }

  async removeConsumer(clusterId: string): Promise<void> {
    const currentCount = this.connectionCounts.get(clusterId) || 0
    const newCount = Math.max(0, currentCount - 1)
    this.connectionCounts.set(clusterId, newCount)

    if (newCount === 0) {
      const consumer = this.consumers.get(clusterId)
      if (consumer) {
        logger.info(`Stopping Redis stream consumer for cluster: ${clusterId}`)

        await consumer.stop()
        this.consumers.delete(clusterId)
        this.connectionCounts.delete(clusterId)
      }
    } else {
      logger.debug(`Keeping Redis stream consumer for cluster: ${clusterId} (${newCount} connections remaining)`)
    }
  }

  hasActiveConsumer(clusterId?: string): boolean {
    if (clusterId) {
      return this.consumers.has(clusterId)
    }
    return this.consumers.size > 0
  }

  getActiveConsumers(): string[] {
    return Array.from(this.consumers.keys())
  }

  getTotalConnections(): number {
    return Array.from(this.connectionCounts.values()).reduce((sum, count) => sum + count, 0)
  }

  getConnectionCount(clusterId: string): number {
    return this.connectionCounts.get(clusterId) || 0
  }

  async stopAllConsumers(): Promise<void> {
    logger.info('Stopping all Redis stream consumers')

    const stopPromises = Array.from(this.consumers.values()).map(consumer => consumer.stop())
    await Promise.all(stopPromises)

    this.consumers.clear()
    this.connectionCounts.clear()
  }
}

export const redisStreamManager = new RedisStreamManager()
