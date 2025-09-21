import Cluster from '#models/cluster'
import { ClusterDnsVerificationService } from '#services/dns/cluster_dns_verification_service'
import type { HttpContext } from '@adonisjs/core/http'
import { DateTime } from 'luxon'

export default class ClusterDnsVerifyController {
  public async index(ctx: HttpContext) {
    const cluster = await Cluster.findOrFail(ctx.params.clusterId)

    await cluster.load('loadBalancers')
    await cluster.load('cloudProvider')

    const ingressLoadBalancer = cluster.loadBalancers.find(
      (loadBalancer) => loadBalancer.type === 'ingress'
    )

    const result = await new ClusterDnsVerificationService(cluster).verify()

    cluster.dnsLastCheckedAt = DateTime.now()

    if (result.ingress && ingressLoadBalancer) {
      ingressLoadBalancer.dnsVerifiedAt = DateTime.now()
      await ingressLoadBalancer.save()
    }

    if (result.ingress) {
      cluster.dnsCompletedAt = DateTime.now()
    }

    await cluster.save()

    return ctx.response.json(result)
  }
}
