'use client'

import { type ClusterLogEntry } from '#services/redis/cluster_logs_service'
import { useEffect, useMemo, useRef } from 'react'
import { Cluster, ProvisioningStepInfo, ProvisioningStepStatus } from '~/types'
import { AccordionContent, AccordionItem, AccordionTrigger } from '../Accordion'
import { Text } from '@kibamail/owly/text'
import Spinner from '../Icons/Spinner'
import Ansi from 'ansi-to-react'
import dayjs from 'dayjs'
import { type TerraformStage } from '#services/terraform/terraform_executor'
import { CheckCircleSolidIcon } from '../Icons/check-circle-solid.svg'
import { XMarkCircleSolidIcon } from '../Icons/xmark-circle-solid.svg'
import { ClockSolidIcon } from '../Icons/clock-solid.svg'
import ClusterDnsStep from './ClusterDnsStep'

export interface ClusterProvisioningStepProps {
  logs: ClusterLogEntry[]
  cluster: Cluster
  active: boolean
  info: ProvisioningStepInfo
}

function getStatusIcon(status: ProvisioningStepStatus, className: string = 'h-5 w-5') {
  switch (status) {
    case 'completed':
      return <CheckCircleSolidIcon className={`${className} text-owly-content-positive !size-4`} />
    case 'in_progress':
      return <Spinner className={`${className} text-owly-content-info !size-4`} />
    case 'failed':
      return <XMarkCircleSolidIcon className={`${className} text-owly-content-negative !size-4`} />
    case 'pending':
    default:
      return <ClockSolidIcon className={`${className} text-owly-content-tertiary !size-4`} />
  }
}

export function getStepStatus(
  cluster: Cluster | null,
  stage: TerraformStage
): ProvisioningStepStatus {
  if (!cluster) {
    return 'pending'
  }

  switch (stage) {
    case 'network':
      if (cluster.networkingCompletedAt) return 'completed'
      if (cluster.networkingErrorAt) return 'failed'
      if (cluster.networkingStartedAt) return 'in_progress'
      return 'pending'

    case 'ssh-keys':
      if (cluster.sshKeysCompletedAt) return 'completed'
      if (cluster.sshKeysErrorAt) return 'failed'
      if (cluster.sshKeysStartedAt) return 'in_progress'
      return 'pending'

    case 'load-balancers':
      if (cluster.loadBalancersCompletedAt) return 'completed'
      if (cluster.loadBalancersErrorAt) return 'failed'
      if (cluster.loadBalancersStartedAt) return 'in_progress'
      return 'pending'

    case 'servers':
      if (cluster.serversCompletedAt) return 'completed'
      if (cluster.serversErrorAt) return 'failed'
      if (cluster.serversStartedAt) return 'in_progress'
      return 'pending'

    case 'volumes':
      if (cluster.volumesCompletedAt) return 'completed'
      if (cluster.volumesErrorAt) return 'failed'
      if (cluster.volumesStartedAt) return 'in_progress'
      return 'pending'

    case 'kubernetes':
      if (cluster.kubernetesClusterCompletedAt) return 'completed'
      if (cluster.kubernetesClusterErrorAt) return 'failed'
      if (cluster.kubernetesClusterStartedAt) return 'in_progress'
      return 'pending'

    case 'dns':
      if (cluster.dnsCompletedAt) return 'completed'
      if (cluster.dnsErrorAt) return 'failed'
      if (cluster.dnsStartedAt) return 'in_progress'
      return 'pending'

    default:
      return 'pending'
  }
}

export default function ClusterProvisioningStep({
  logs: allLogs,
  cluster,
  active,
  info,
}: ClusterProvisioningStepProps) {
  const ref = useRef<HTMLDivElement | null>(null)

  const logs = useMemo(() => {
    return allLogs.filter((log) => log.stage === info.stage)
  }, [allLogs, info])

  useEffect(() => {
    setTimeout(() => {
      if (ref.current) {
        ref.current.scrollTop = ref.current.scrollHeight
      }
    }, 5)
  }, [logs, active])

  const status = getStepStatus(cluster, info.stage)

  return (
    <AccordionItem value={info.stage}>
      <AccordionTrigger>
        <div className="flex items-center gap-1.5">
          <div className="flex items-center gap-1">
            {info.icon}
            <span className="font-medium"></span>
            <Text className="text-owly-content-secondary">{info.title}</Text>
          </div>
          <div className="ml-auto mr-4">{getStatusIcon(status)}</div>
        </div>
      </AccordionTrigger>
      <AccordionContent>
        <div className="px-4 py-2 w-full border-b border-owly-border-tertiary">
          <Text className="!text-sm text-owly-content-tertiary">{info.description}</Text>
        </div>

        {info.stage === 'dns' ? (
          <ClusterDnsStep cluster={cluster} />
        ) : (
          <div
            ref={ref}
            className="w-full h-[300px] overflow-y-auto rounded-b-md px-4 bg-[hsl(0,0%,98%)]"
          >
            {logs.map((log) => (
              <div key={log?.id} className="w-full flex items-start gap-4 py-1">
                <Text className="!text-xs !font-mono shrink-0">
                  {dayjs(log?.timestamp).format('MMM DD HH:mm:ss')}
                </Text>

                <Text className="font-mono !text-xs">
                  <Ansi>{log?.message}</Ansi>
                </Text>
              </div>
            ))}
          </div>
        )}
      </AccordionContent>
    </AccordionItem>
  )
}
