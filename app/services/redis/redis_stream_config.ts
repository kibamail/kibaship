export class RedisStreamConfig {
  static readonly TERRAFORM_STREAM = 'terraform:clusters'

  static getClusterLogsRoom(clusterId: string): string {
    return `cluster:${clusterId}:logs`
  }

  static getClusterStream(clusterId: string): string {
    return `terraform:cluster:${clusterId}`
  }
}
