import { Socket } from 'socket.io'
import { redisStreamManager } from '#services/redis/redis_stream_manager'
import logger from '@adonisjs/core/services/logger'

export interface ClusterLogSubscription {
  clusterId: string
}

export class ClusterLogsHandler {
  private socketClusterMap = new Map<string, string>()

  async onSubscribe(socket: Socket, data: ClusterLogSubscription): Promise<void> {
    const { clusterId } = data

    if (!clusterId) {
      socket.emit('error', { message: 'clusterId is required' })
      return
    }

    try {
      this.socketClusterMap.set(socket.id, clusterId)

      await redisStreamManager.ensureConsumer(clusterId, (log) => {
        logger.info(`Log message received for cluster ${clusterId}`)
        socket.emit(`cluster:${clusterId}:logs`, {
          ...log
        })
      })

      logger.info(`Socket ${socket.id} subscribed to cluster logs: ${clusterId}`)

    } catch (error) {
      logger.error(`Failed to subscribe socket ${socket.id} to cluster ${clusterId}:`, error)
      socket.emit('error', { message: 'Failed to subscribe to cluster logs' })
    }
  }

  async onUnsubscribe(socket: Socket, data: ClusterLogSubscription): Promise<void> {
    const { clusterId } = data

    if (!clusterId) {
      return
    }

    try {
      this.socketClusterMap.delete(socket.id)

      await redisStreamManager.removeConsumer(clusterId)

      socket.emit('unsubscribed', { clusterId })
      logger.info(`Socket ${socket.id} unsubscribed from cluster logs: ${clusterId}`)

    } catch (error) {
      logger.error(`Failed to unsubscribe socket ${socket.id} from cluster ${clusterId}:`, error)
    }
  }

  async onDisconnect(socket: Socket): Promise<void> {
    const clusterId = this.socketClusterMap.get(socket.id)

    if (clusterId) {
      try {
        await redisStreamManager.removeConsumer(clusterId)
        this.socketClusterMap.delete(socket.id)

        logger.info(`Socket ${socket.id} disconnected, cleaned up cluster subscription: ${clusterId}`)
      } catch (error) {
        logger.error(`Failed to cleanup cluster subscription for socket ${socket.id}:`, error)
      }
    }
  }

  getActiveSubscriptions(): Record<string, string> {
    const subscriptions: Record<string, string> = {}

    for (const [socketId, clusterId] of this.socketClusterMap.entries()) {
      subscriptions[socketId] = clusterId
    }

    return subscriptions
  }
}
