import { TerraformStage } from '#services/terraform/terraform_executor'
import { RedisStream, StreamMessage } from '#utils/redis_stream'
import { RedisStreamConfig } from './redis_stream_config.js'

export interface ClusterLogEntry {
  id: string
  type: string
  message: string
  timestamp: string
  cluster_id: string
  stage?: TerraformStage
}

export class ClusterLogsService {
  static parseStreamMessages(messages: StreamMessage[], defaultClusterId?: string): ClusterLogEntry[] {
    return messages
      .flatMap(msg =>
        msg.entries.map(entry => ({
          id: entry.id,
          type: entry.fields.type || 'info',
          message: entry.fields.message || '',
          timestamp: entry.fields.timestamp || '',
          cluster_id: entry.fields.cluster_id || defaultClusterId || '',
          stage: entry.fields.stage as TerraformStage
        }))
      )
      .sort((a, b) => a.id.localeCompare(b.id))
  }

  static async getLogsForCluster(clusterId: string): Promise<ClusterLogEntry[]> {
    const messages = await new RedisStream()
      .stream(RedisStreamConfig.getClusterStream(clusterId))
      .from('0')
      .read()

    return this.parseStreamMessages(messages, clusterId)
  }
}
