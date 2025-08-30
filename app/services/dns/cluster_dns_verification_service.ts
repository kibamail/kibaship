import { promisify } from 'node:util'
import { resolve4 } from 'node:dns'
import logger from '@adonisjs/core/services/logger'
import Cluster from '#models/cluster'
import { randomUUID } from 'node:crypto'

const resolveIpv4Address = promisify(resolve4)

export interface DnsVerificationResult {
  ingress: boolean
  cluster: boolean
}

export class ClusterDnsVerificationService {
  public static async verify(cluster: Cluster): Promise<DnsVerificationResult> {
    const ingressLoadBalancer = cluster.loadBalancers.find(lb => lb.type === 'ingress')
    const clusterLoadBalancer = cluster.loadBalancers.find(lb => lb.type === 'cluster')

    /**
     * 
     * In order to check the definition of the wildcard domain
     * Check a random subdomain
     */
    const ingressDomain = `${randomUUID()}.${cluster.subdomainIdentifier}`
    const clusterDomain = `kube.${cluster.subdomainIdentifier}`

    const ingressVerified = await this.verifyDnsRecord(
      ingressDomain,
      ingressLoadBalancer?.publicIpv4Address || null
    )

    const clusterVerified = await this.verifyDnsRecord(
      clusterDomain,
      clusterLoadBalancer?.publicIpv4Address || null
    )

    logger.info('DNS verification completed', {
      clusterId: cluster.id,
      ingressVerified,
      clusterVerified
    })

    return {
      ingress: ingressVerified,
      cluster: clusterVerified
    }
  }

  private static async verifyDnsRecord(
    domain: string,
    expectedIp: string | null
  ): Promise<boolean> {
    if (!expectedIp) {
      return false
    }

    try {
      const resolvedIps = await resolveIpv4Address(domain)

      const verified = resolvedIps.includes(expectedIp)

      logger.debug('DNS record verification', {
        domain,
        expectedIp,
        resolvedIps,
        verified
      })

      return verified
    } catch (error) {
      console.error(error)
      logger.warn('DNS resolution failed', {
        domain,
        expectedIp,
        error: error.message
      })

      return false
    }
  }
}
