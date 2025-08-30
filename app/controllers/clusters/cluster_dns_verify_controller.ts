import ProvisionKubernetesJob from '#jobs/clusters/provision_kubernetes_job'
import Cluster from '#models/cluster'
import { ClusterDnsVerificationService } from '#services/dns/cluster_dns_verification_service'
import type { HttpContext } from '@adonisjs/core/http'
import queue from '@rlanz/bull-queue/services/main'
import { DateTime } from 'luxon'

export default class ClusterDnsVerifyController {
    public async index(ctx: HttpContext) {
        const cluster = await Cluster.findOrFail(ctx.params.clusterId)

        await cluster.load('loadBalancers')

        const clusterLoadBalancer = cluster.loadBalancers.find(loadBalancer => loadBalancer.type === 'cluster')
        const ingressLoadBalancer = cluster.loadBalancers.find(loadBalancer => loadBalancer.type === 'ingress')

        const result = await ClusterDnsVerificationService.verify(cluster)

        cluster.dnsLastCheckedAt = DateTime.now()

        if (result.cluster && clusterLoadBalancer) {
            clusterLoadBalancer.dnsVerifiedAt = DateTime.now()
            await clusterLoadBalancer.save()
        }

        if (result.ingress && ingressLoadBalancer) {
            ingressLoadBalancer.dnsVerifiedAt = DateTime.now()
            await ingressLoadBalancer.save()
        }

        if (result.cluster && result.ingress) {
            cluster.dnsCompletedAt = DateTime.now()

            await queue.dispatch(ProvisionKubernetesJob, {
                clusterId: cluster.id
            })
        }

        await cluster.save()

        return ctx.response.json(result)
    }
}
