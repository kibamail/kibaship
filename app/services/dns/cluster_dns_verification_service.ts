import { Resolver } from 'node:dns'
import logger from '@adonisjs/core/services/logger'
import Cluster from '#models/cluster'
import { randomUUID } from 'node:crypto'

export interface DnsVerificationResult {
  ingress: boolean
  cluster: boolean
}

export class ClusterDnsVerificationService {
  protected resolver = new Resolver()

  protected dnsServersToVerify = [
    // Google DNS
    ['8.8.8.8', '8.8.4.4'],

    // Cloudflare DNs
    ['1.1.1.1', '1.1.0.1'],

    // OpenDNS
    ['208.67.222.222', '208.67.220.220'],
  ]

  constructor(protected cluster: Cluster) {}

  public async verify() {
    for (const servers of this.dnsServersToVerify) {
      const { ingress } = await this.verifyOnDnsServers(servers)

      if (!ingress) {
        return { ingress: false, cluster: true }
      }
    }

    return { ingress: true, cluster: true }
  }

  protected async verifyOnDnsServers(servers: string[]): Promise<DnsVerificationResult> {
    this.resolver = new Resolver()

    this.resolver.setServers(servers)

    const ingressLoadBalancer = this.cluster.loadBalancers.find((lb) => lb.type === 'ingress')

    /**
     *
     * In order to check the definition of the wildcard domain
     * Check a random subdomain
     */
    const ingressDomain = `${randomUUID()}.${this.cluster.subdomainIdentifier}`

    const ingressVerified = await this.verifyDnsRecord(
      ingressDomain,
      ingressLoadBalancer?.publicIpv4Address || null
    )

    logger.info('DNS verification completed', {
      clusterId: this.cluster.id,
      ingressVerified,
    })

    return {
      ingress: ingressVerified,
      cluster: true, // Always return true for cluster since we don't need kube subdomain verification
    }
  }

  private async verifyDnsRecord(domain: string, expectedIp: string | null): Promise<boolean> {
    if (!expectedIp) {
      return false
    }

    const self = this

    try {
      await new Promise((resolve, reject) => {
        self.resolver.resolve4(domain, (error, hostnames) => {
          if (error) {
            console.error(error)
            logger.warn('DNS resolution failed', {
              domain,
              expectedIp,
              error: error.message,
            })
            return reject(error)
          }

          const verified = hostnames.includes(expectedIp)

          logger.debug('DNS record verification', {
            domain,
            expectedIp,
            resolvedIps: hostnames,
            verified,
          })

          return resolve(verified)
        })
      })

      return true
    } catch (error) {
      console.error(error)
      return false
    }
  }
}
