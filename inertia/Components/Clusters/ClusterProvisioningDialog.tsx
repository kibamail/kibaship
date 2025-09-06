'use client'

import * as Dialog from '@kibamail/owly/dialog'
import { Button } from '@kibamail/owly/button'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { Accordion } from '../Accordion'
import { Cluster, PageProps, ProvisioningStepInfo } from '~/types'
import { Spinner } from '../Icons/Spinner'
import { CloudCheckIcon } from '../Icons/cloud-check.svg'
import { SettingsIcon } from '../Icons/settings.svg'
import { StackIcon } from '../Icons/stack.svg'
import { K8sIcon } from '../Icons/k8s.svg'
import { NavArrowDownIcon } from '../Icons/nav-arrow-down.svg'
import { useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { axios } from '~/app/axios'
import { usePage } from '@inertiajs/react'
import { type ClusterLogEntry } from '#services/redis/cluster_logs_service'
import React, { useEffect, useState } from 'react'
import { TerraformStage } from '#services/terraform/terraform_executor'
import { useSocketIo } from '~/hooks/useSocketIo'
import { ServerIcon } from '../Icons/server.svg'
import { NetworkIcon } from '../Icons/network.svg'
import { DNSIcon } from '../Icons/dns.svg'
import { DockerIcon } from '../Icons/docker.svg'

const ClusterProvisioningStep = React.lazy(() => import('./ClusterProvisioningStep'))

interface ClusterProvisioningDialogProps {
  cluster: Cluster | null
  isOpen: boolean
  onClusterUpdated: (cluster: Cluster) => void
  onOpenChange: (open: boolean) => void
}

const provisioningSteps: ProvisioningStepInfo[] = [
  {
    stage: 'talos-image',
    title: 'Talos OS image',
    description: 'Creating or locating the Talos OS system image for cluster nodes',
    icon: <DockerIcon className="!size-4.5" />,
  },
  {
    stage: 'network',
    title: 'Network infrastructure',
    description: 'Setting up virtual networks, subnets, and security groups',
    icon: <CloudCheckIcon className="!size-4.5" />,
  },
  {
    stage: 'ssh-keys',
    title: 'SSH keys',
    description: 'Generating and configuring SSH keys for secure access to cluster nodes',
    icon: <SettingsIcon className="!size-5" />,
  },
  {
    stage: 'load-balancers',
    title: 'Load balancers',
    description: 'Configuring load balancers for high availability and traffic distribution',
    icon: <NetworkIcon className="!size-5" />,
  },
  {
    stage: 'servers',
    title: 'Server provisioning',
    description: 'Creating and configuring control plane and worker servers',
    icon: <ServerIcon className="!size-4" />,
  },
  {
    stage: 'volumes',
    title: 'Storage volumes',
    description: 'Attaching and configuring persistent storage volumes',
    icon: <StackIcon className="!size-5.5" />,
  },
  {
    stage: 'dns',
    title: 'DNS configuration',
    description: 'Setup DNS for the cluster ingress.',
    icon: <DNSIcon className="!size-5" />,
  },
  {
    stage: 'kubernetes-config',
    title: 'Kubernetes configuration',
    description: 'Generating Talos machine configurations and client configuration',
    icon: <SettingsIcon className="!size-5" />,
  },
  {
    stage: 'kubernetes',
    title: 'Kubernetes cluster',
    description: 'Installing and configuring Kubernetes on the provisioned infrastructure',
    icon: <K8sIcon className="!size-5" />,
  },
]

function getLatestStage(cluster: Cluster | null | undefined): TerraformStage {
  if (!cluster) {
    return 'talos-image'
  }

  const stagesWithStatus = provisioningSteps.map((step) => ({
    stage: step.stage,
    status: cluster ? cluster?.progress[step.stage] : 'pending',
  }))

  const latestStage = stagesWithStatus.find((stage) => stage.status === 'in_progress')?.stage
  const latestFailedStage = stagesWithStatus.find((stage) => stage.status === 'failed')?.stage

  return latestStage || latestFailedStage || 'talos-image'
}

export function ClusterProvisioningDialog({
  cluster: initialCluster,
  isOpen,
  onOpenChange,
  onClusterUpdated
}: ClusterProvisioningDialogProps) {
  const { props } = usePage<PageProps>()
  const queryClient = useQueryClient()
  const [expandedStage, setExpandedStage] = useState<TerraformStage>(getLatestStage(initialCluster))

  const socket = useSocketIo()
  const socketRegisteredRef = useRef({
    update: false,
    logs: false,
  })

  const clusterQuery = useQuery<Cluster>({
    queryKey: ['clusters', initialCluster?.id],
    async queryFn() {
      const response = await axios.get(`/w/${props.workspace.slug}/clusters/${initialCluster?.id}`)

      const latestStage = getLatestStage(response.data)

      setExpandedStage(latestStage)
      onClusterUpdated(response.data)

      return response.data
    },
    enabled: Boolean(initialCluster) && isOpen,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
  })

  const cluster = clusterQuery?.data

  const logsQuery = useQuery<ClusterLogEntry[]>({
    queryKey: ['clusters', cluster?.id, 'logs'],
    async queryFn() {
      const response = await axios.get(`/w/${props.workspace.slug}/clusters/${cluster?.id}/logs`)

      return response.data?.logs
    },
    enabled: Boolean(cluster) && isOpen,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
  })

  useEffect(() => {
    if (!cluster) {
      return
    }

    if (socketRegisteredRef.current.update) {
      return
    }

    socketRegisteredRef.current = {
      ...socketRegisteredRef.current,
      update: true,
    }

    socket.on(`cluster:${cluster.id}:updated`, () => {
      clusterQuery.refetch()
    })

    return () => {
      // console.log('=== executed disconnect')
      // listener.disconnect()
    }
  }, [cluster])

  const existingLogsFetched = logsQuery?.isSuccess
  const clusterId = cluster?.id

  useEffect(() => {
    if (existingLogsFetched && clusterId) {
      socket.emit('cluster:logs', {
        clusterId: cluster?.id,
      })

      if (socketRegisteredRef.current.logs) {
        return
      }

      socketRegisteredRef.current = {
        ...socketRegisteredRef.current,
        logs: true,
      }

      socket.on(`cluster:${clusterId}:logs`, (data) => {
        queryClient.setQueryData(
          ['clusters', clusterId, 'logs'],
          (oldData: ClusterLogEntry[] | undefined) => {
            return [...(oldData || []), data]
          }
        )
      })

      return () => {
        // console.log('=== executed logs disconnect')
        // socket.emit('cluster:logs:unsubscribe')
        // listener.disconnect()
      }
    }
  }, [existingLogsFetched, clusterId])

  const restartMutation = useMutation({
    async mutationFn({ type }: { type: 'start' | 'failed' }) {
      const response = await axios.post(
        `/w/${props.workspace.slug}/clusters/${cluster?.id}/restart`,
        {
          type,
        }
      )
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['clusters', cluster?.id, 'logs'] })
    },
  })

  useEffect(() => {
    setExpandedStage(getLatestStage(cluster))
  }, [initialCluster])

  const atLeastOneFailed = cluster?.firstFailedStage !== null

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content className="!max-w-2xl">
        <Dialog.Header>
          <Dialog.Title>
            {cluster
              ? `Provisioning cluster ${cluster?.subdomainIdentifier}`
              : 'Cluster provisioning'}
          </Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Monitor the provisioning progress of your cluster infrastructure
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="p-5">
          {logsQuery?.isLoading ? (
            <div className="flex justify-center py-16">
              <Spinner className="size-6" />
            </div>
          ) : cluster ? (
            <Accordion
              type="single"
              className="w-full"
              value={expandedStage}
              onValueChange={(value) => setExpandedStage(value as TerraformStage)}
            >
              <div className="flex flex-col gap-2">
                {provisioningSteps.map((info) => (
                  <ClusterProvisioningStep
                    info={info}
                    cluster={cluster}
                    key={info.stage}
                    logs={logsQuery.data || []}
                    active={expandedStage === info.stage}
                  />
                ))}
              </div>
            </Accordion>
          ) : null}
        </div>

        <Dialog.Footer className="flex justify-between items-center">
          <Dialog.Close asChild>
            <Button variant="secondary">Close</Button>
          </Dialog.Close>

          <DropdownMenu.Root modal={false}>
            <DropdownMenu.Trigger asChild>
              <Button
                disabled={!atLeastOneFailed || restartMutation.isPending}
                className="flex items-center gap-2"
              >
                {restartMutation.isPending ? 'Restarting...' : 'Rerun provisioning'}
                <NavArrowDownIcon className="w-4 h-4" />
              </Button>
            </DropdownMenu.Trigger>

            <DropdownMenu.Content
              align="end"
              sideOffset={8}
              className="border kb-border-tertiary rounded-xl p-1 shadow-[0px_16px_24px_-8px_var(--black-10)] kb-background-primary w-48 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 z-50"
            >
              <DropdownMenu.Item
                className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
                onSelect={() => {
                  restartMutation.mutate({ type: 'start' })
                }}
                disabled={restartMutation.isPending}
              >
                <span className="text-sm">Rerun from start</span>
              </DropdownMenu.Item>

              <DropdownMenu.Item
                className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
                onSelect={() => {
                  restartMutation.mutate({ type: 'failed' })
                }}
                disabled={restartMutation.isPending}
              >
                <span className="text-sm">Rerun from failed</span>
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Root>
        </Dialog.Footer>
      </Dialog.Content>
    </Dialog.Root>
  )
}
