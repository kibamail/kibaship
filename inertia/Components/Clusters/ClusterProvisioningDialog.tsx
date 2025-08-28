'use client'

import * as Dialog from '@kibamail/owly/dialog'
import { Text } from '@kibamail/owly/text'
import { Button } from '@kibamail/owly/button'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '../Accordion'
import { Cluster, ProvisioningStepStatus } from '~/types'
import { CheckIcon } from '../Icons/check.svg'
import { ClockIcon } from '../Icons/clock.svg'
import { Spinner } from '../Icons/Spinner'
import { XMarkIcon } from '../Icons/xmark.svg'
import { CloudCheckIcon } from '../Icons/cloud-check.svg'
import { SettingsIcon } from '../Icons/settings.svg'
import { StackIcon } from '../Icons/stack.svg'
import { K8sIcon } from '../Icons/k8s.svg'
import { KibaIcon } from '../Icons/kiba.svg'

interface ClusterProvisioningDialogProps {
  cluster: Cluster | null
  isOpen: boolean
  onOpenChange: (open: boolean) => void
}

interface ProvisioningStepInfo {
  key: string
  title: string
  description: string
  icon: React.ReactNode
}

const provisioningSteps: ProvisioningStepInfo[] = [
  {
    key: 'networking',
    title: 'Network infrastructure',
    description: 'Setting up virtual networks, subnets, and security groups',
    icon: <CloudCheckIcon className="!size-4.5" />,
  },
  {
    key: 'sshKeys',
    title: 'SSH keys',
    description: 'Generating and configuring SSH keys for secure access to cluster nodes',
    icon: <SettingsIcon className="!size-5" />,
  },

  {
    key: 'loadBalancers',
    title: 'Load balancers',
    description: 'Configuring load balancers for high availability and traffic distribution',
    icon: <StackIcon className="!size-5" />,
  },
  {
    key: 'servers',
    title: 'Server provisioning',
    description: 'Creating and configuring virtual machines for control plane and worker nodes',
    icon: <StackIcon className="!size-5" />,
  },
  {
    key: 'volumes',
    title: 'Storage volumes',
    description: 'Attaching and configuring persistent storage volumes',
    icon: <StackIcon className="!size-5" />,
  },
  {
    key: 'kubernetesCluster',
    title: 'Kubernetes cluster',
    description: 'Installing and configuring Kubernetes on the provisioned infrastructure',
    icon: <K8sIcon className="!size-5" />,
  },
  {
    key: 'kibashipOperator',
    title: 'Kibaship operator',
    description: 'Installing Kibaship operator for cluster management and monitoring',
    icon: <KibaIcon className="!size-5" />,
  },
]

function getStatusIcon(status: ProvisioningStepStatus, className: string = 'h-5 w-5') {
  switch (status) {
    case 'completed':
      return <CheckIcon className={`${className} text-owly-content-positive !size-4`} />
    case 'in_progress':
      return <Spinner className={`${className} text-owly-content-info !size-4`} />
    case 'failed':
      return <XMarkIcon className={`${className} text-owly-content-negative !size-4`} />
    case 'pending':
    default:
      return <ClockIcon className={`${className} text-owly-content-tertiary !size-4`} />
  }
}

export function ClusterProvisioningDialog({
  cluster,
  isOpen,
  onOpenChange,
}: ClusterProvisioningDialogProps) {
  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content className="max-w-4xl">
        <Dialog.Header>
          <Dialog.Title>
            {cluster ? `Provisioning ${cluster?.name || cluster?.id}` : 'Cluster provisioning'}
          </Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Monitor the provisioning progress of your cluster infrastructure
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pb-5 pt-5">
          <Accordion type="single" className="w-full">
            <div className="flex flex-col gap-2">
              {provisioningSteps.map((stepInfo) => {
                return (
                  <AccordionItem key={stepInfo.key} value={stepInfo.key}>
                    <AccordionTrigger>
                      <div className="flex items-center gap-1.5">
                        <div className="flex items-center gap-1">
                          {stepInfo.icon}
                          <span className="font-medium"></span>
                          <Text className="text-owly-content-secondary">{stepInfo.title}</Text>
                        </div>
                        <div className="ml-auto mr-4">{getStatusIcon('pending')}</div>
                      </div>
                    </AccordionTrigger>
                    <AccordionContent>
                      <div className="px-6 py-2 w-full">
                        <Text className="!text-sm text-owly-content-tertiary">
                          {stepInfo.description}
                        </Text>
                      </div>

                      <div className="w-full bg-owly-background-brand h-[300px] overflow-y-auto rounded-b-md"></div>
                    </AccordionContent>
                  </AccordionItem>
                )
              })}
            </div>
          </Accordion>
        </div>

        <Dialog.Footer className="flex justify-end">
          <Dialog.Close asChild>
            <Button variant="secondary">Close</Button>
          </Dialog.Close>
        </Dialog.Footer>
      </Dialog.Content>
    </Dialog.Root>
  )
}
