import { Job } from '@rlanz/bull-queue'
import queue from '@rlanz/bull-queue/services/main'
import Cluster from '#models/cluster'
import { ClusterDnsVerificationService } from '#services/dns/cluster_dns_verification_service'
import { DateTime } from 'luxon'
import logger from '@adonisjs/core/services/logger'

interface ClusterDnsVerifyJobPayload {
  clusterId: string
  retryCount?: number
}

/**
 * ClusterDnsVerifyJob
 *
 * Automatically verifies DNS configuration for a cluster after Kubernetes boot.
 * This job replicates the functionality of ClusterDnsVerifyController but runs
 * automatically as part of the provisioning pipeline.
 *
 * Features:
 * - Automatically dispatched after ProvisionKubernetesBootJob completes
 * - Verifies DNS records are correctly configured by the user
 * - Updates cluster and load balancer DNS verification timestamps
 * - Automatically retries every 30 seconds if DNS verification fails
 * - Stops retrying after a reasonable number of attempts to avoid infinite loops
 */
export default class ClusterDnsVerifyJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Maximum number of retry attempts before giving up
   * This prevents infinite retries if DNS is never configured correctly
   */
  private static readonly MAX_RETRY_ATTEMPTS = 120 // 120 * 30 seconds = 1 hour

  /**
   * Delay between retry attempts in milliseconds (30 seconds)
   */
  private static readonly RETRY_DELAY_MS = 30 * 1000

  async handle(payload: ClusterDnsVerifyJobPayload) {
    const cluster = await Cluster.findOrFail(payload.clusterId)

    await cluster.load('loadBalancers')
    await cluster.load('cloudProvider')

    const retryCount = payload.retryCount || 0

    logger.info('Starting DNS verification', {
      clusterId: cluster.id,
      retryCount,
      subdomainIdentifier: cluster.subdomainIdentifier,
    })

    const ingressLoadBalancer = cluster.loadBalancers.find(
      (loadBalancer) => loadBalancer.type === 'ingress'
    )

    if (!ingressLoadBalancer) {
      logger.warn('No ingress load balancer found for cluster', {
        clusterId: cluster.id,
      })
      return
    }

    const result = await new ClusterDnsVerificationService(cluster).verify()

    cluster.dnsLastCheckedAt = DateTime.now()

    if (result.ingress) {
      logger.info('DNS verification successful', {
        clusterId: cluster.id,
        retryCount,
        subdomainIdentifier: cluster.subdomainIdentifier,
      })

      ingressLoadBalancer.dnsVerifiedAt = DateTime.now()
      await ingressLoadBalancer.save()

      cluster.dnsCompletedAt = DateTime.now()
      cluster.status = 'healthy'
      await cluster.save()

      logger.info('DNS verification completed successfully', {
        clusterId: cluster.id,
        totalRetries: retryCount,
      })

      return
    }

    await cluster.save()

    if (retryCount >= ClusterDnsVerifyJob.MAX_RETRY_ATTEMPTS) {
      logger.warn('DNS verification failed after maximum retry attempts', {
        clusterId: cluster.id,
        retryCount,
        maxRetries: ClusterDnsVerifyJob.MAX_RETRY_ATTEMPTS,
        subdomainIdentifier: cluster.subdomainIdentifier,
      })

      cluster.dnsErrorAt = DateTime.now()
      await cluster.save()
      return
    }

    logger.info('DNS verification failed, scheduling retry', {
      clusterId: cluster.id,
      retryCount,
      nextRetryIn: ClusterDnsVerifyJob.RETRY_DELAY_MS / 1000,
      subdomainIdentifier: cluster.subdomainIdentifier,
    })

    await queue.dispatch(
      ClusterDnsVerifyJob,
      {
        clusterId: payload.clusterId,
        retryCount: retryCount + 1,
      },
      {
        delay: ClusterDnsVerifyJob.RETRY_DELAY_MS,
      }
    )
  }

  async rescue(payload: ClusterDnsVerifyJobPayload) {
    logger.error('ClusterDnsVerifyJob failed permanently', {
      clusterId: payload.clusterId,
      retryCount: payload.retryCount || 0,
    })

    const cluster = await Cluster.findOrFail(payload.clusterId)
    cluster.dnsErrorAt = DateTime.now()
    await cluster.save()
  }
}
