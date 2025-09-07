import * as k8s from '@kubernetes/client-node'
import Cluster from '#models/cluster'
import logger from '@adonisjs/core/services/logger'

export class KubernetesService {
  private kubeConfig: k8s.KubeConfig
  private coreV1Api: k8s.CoreV1Api

  constructor(private cluster: Cluster) {
    this.kubeConfig = new k8s.KubeConfig()
    this.setupKubeConfig()
    this.coreV1Api = this.kubeConfig.makeApiClient(k8s.CoreV1Api)
  }

  private setupKubeConfig(): void {
    if (!this.cluster.kubeconfig) {
      throw new Error('Kubeconfig not found in cluster')
    }

    const kubeconfigComponents = this.cluster.kubeconfig as {
      host: string
      clientCertificate: string
      clientKey: string
      clusterCaCertificate: string
    }

    const clusterName = this.cluster.subdomainIdentifier
    const userName = `admin@${clusterName}`

    const loadBalancer = this.cluster.loadBalancers.find(lb => lb.type === 'cluster')

    if (! loadBalancer) {
      throw new Error('Cluster load balancer not found. Cannot configure kubeconfig')
    }

    // Create cluster configuration
    const cluster = {
      name: clusterName,
      server: `https://${loadBalancer.publicIpv4Address}:6443`,
      caData: kubeconfigComponents.clusterCaCertificate,
    }

    // Create user configuration
    const user = {
      name: userName,
      certData: kubeconfigComponents.clientCertificate,
      keyData: kubeconfigComponents.clientKey,
    }

    // Create context configuration
    const context = {
      name: userName, // admin@<cluster-name>
      cluster: cluster.name,
      user: user.name,
      namespace: 'default',
    }

    const k8sConfigurationOptions = {
      clusters: [cluster],
      users: [user],
      contexts: [context],
      currentContext: context.name,
    }

    // Load the configuration
    this.kubeConfig.loadFromOptions(k8sConfigurationOptions)
  }

  /**
   * Get all nodes in the cluster
   */
  async getNodes(): Promise<k8s.V1Node[]> {
      const response = await this.coreV1Api.listNode()
      return response.items || []
  }

  /**
   * Get ready nodes count
   */
  async getReadyNodesCount(): Promise<number> {
    const nodes = await this.getNodes()
    return nodes.filter(node => this.isNodeReady(node)).length
  }

  /**
   * Get total nodes count
   */
  async getTotalNodesCount(): Promise<number> {
    const nodes = await this.getNodes()
    return nodes.length
  }

  /**
   * Check if a node is ready
   */
  private isNodeReady(node: k8s.V1Node): boolean {
    const conditions = node.status?.conditions || []
    const readyCondition = conditions.find(condition => condition.type === 'Ready')
    return readyCondition?.status === 'True'
  }

  /**
   * Get nodes with their status information
   */
  async getNodesWithStatus(): Promise<Array<{
    name: string
    ready: boolean
    status: string
    version: string
    createdAt: Date | undefined
  }>> {
    const nodes = await this.getNodes()
    
    return nodes.map(node => ({
      name: node.metadata?.name || 'Unknown',
      ready: this.isNodeReady(node),
      status: this.getNodeStatus(node),
      version: node.status?.nodeInfo?.kubeletVersion || 'Unknown',
      createdAt: node.metadata?.creationTimestamp,
    }))
  }

  /**
   * Get node status string
   */
  private getNodeStatus(node: k8s.V1Node): string {
    const conditions = node.status?.conditions || []
    
    if (this.isNodeReady(node)) {
      return 'Ready'
    }

    // Find other conditions that might indicate issues
    for (const condition of conditions) {
      if (condition.status === 'True' && condition.type !== 'Ready') {
        return condition.type
      }
    }

    return 'NotReady'
  }

  /**
   * Wait for nodes to be discovered
   */
  async waitForNodesDiscovered(expectedCount: number, timeoutMs: number = 300000): Promise<boolean> {
    const startTime = Date.now()
    
    while (Date.now() - startTime < timeoutMs) {
      console.log('Checking for discovered nodes...')
      try {
        const totalCount = await this.getTotalNodesCount()

        logger.info(`Discovered nodes: ${totalCount}/${expectedCount}`)
        
        if (totalCount >= expectedCount) {
          return true
        }
        
        // Wait 10 seconds before checking again
        await new Promise(resolve => setTimeout(resolve, 10000))
      } catch (error) {
        console.error('Error checking node discovery:', error)
        await new Promise(resolve => setTimeout(resolve, 10000))
      }
    }
    
    return false
  }

  /**
   * Wait for all nodes to be ready
   */
  async waitForNodesReady(expectedCount: number, timeoutMs: number = 300000): Promise<boolean> {
    const startTime = Date.now()
    
    while (Date.now() - startTime < timeoutMs) {
      try {
        const readyCount = await this.getReadyNodesCount()

        logger.info(`Ready nodes: ${readyCount}/${expectedCount}`)
        
        if (readyCount >= expectedCount) {
          return true
        }
        
        // Wait 10 seconds before checking again
        await new Promise(resolve => setTimeout(resolve, 10000))
      } catch (error) {
        console.error('Error checking node readiness:', error)
        await new Promise(resolve => setTimeout(resolve, 3000))
      }
    }
    
    return false
  }
}