import { TerraformStage } from '#services/terraform/terraform_executor'
import { RedisStream } from '#utils/redis_stream'
import { ClusterLogEntry } from './cluster_logs_service.js'
import { RedisStreamConfig } from './redis_stream_config.js'
import logger from '@adonisjs/core/services/logger'

export class ClusterLogConsumer {
  private running = false
  private streamName: string
  private clusterId: string

  constructor(clusterId: string, protected onClusterLogsUpdated: (log: ClusterLogEntry) => void) {
    this.clusterId = clusterId
    this.streamName = RedisStreamConfig.getClusterStream(clusterId)
  }

  async start(): Promise<void> {
    if (this.running) {
      return
    }

    this.running = true
    logger.info(`Starting Redis stream consumer for cluster: ${this.clusterId}`)

    this.consume()
  }

  async stop(): Promise<void> {
    this.running = false
    logger.info(`Stopping Redis stream consumer for cluster: ${this.clusterId}`)
  }

  private async consume(): Promise<void> {
    let lastId = '$'

    while (this.running) {
      try {
        const messages = await new RedisStream()
          .stream(this.streamName)
          .from(lastId)
          .block(1000)
          .count(10)
          .read()

        for (const streamMessage of messages) {
          for (const entry of streamMessage.entries) {
            const logMessage: ClusterLogEntry = {
              id: entry.id,
              type: entry.fields.type || 'info',
              message: entry.fields.message || '',
              timestamp: entry.fields.timestamp || new Date().toISOString(),
              cluster_id: entry.fields.cluster_id || '',
              stage: entry.fields.stage as TerraformStage
            }

            this.onClusterLogsUpdated?.(logMessage)
            lastId = entry.id
          }
        }
      } catch (error) {
        if (this.running) {
          logger.error(`Error consuming stream for cluster ${this.clusterId}:`, error)
          await this.sleep(5000)
        }
      }
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }
}
