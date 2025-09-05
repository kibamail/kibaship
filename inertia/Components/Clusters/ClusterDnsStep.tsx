import { Text } from '@kibamail/owly/text'
import * as Alert from '@kibamail/owly/alert'
import { WarningCircleSolidIcon } from '../Icons/warning-circle-solid.svg'
import { Cluster, PageProps } from '~/types'
import { Badge, Button } from '@kibamail/owly'
import { CopyIcon } from '../Icons/copy.svg'
import classNames from 'classnames'
import { CheckIcon } from '../Icons/check.svg'
import { useMutation } from '@tanstack/react-query'
import { axios } from '~/app/axios'
import { usePage } from '@inertiajs/react'
import { useState } from 'react'

export interface ClusterDnsStepProps {
  cluster: Cluster
}

export default function ClusterDnsStep({ cluster }: ClusterDnsStepProps) {
  const { props } = usePage<PageProps>()

  const [copied, setCopied] = useState({
    ingressName: false,
    ingressValue: false,
    clusterName: false,
    clusterValue: false,
  })

  function parseSubdomainIdentifier(domain: string, loadBalancerType: 'cluster' | 'ingress' | 'tcp' | 'udp'): string {
    if (!domain) return loadBalancerType === 'cluster' ? 'kube.*' : '*'
    
    const parts = domain.split('.')
    
    if (parts.length <= 2) {
      return loadBalancerType === 'cluster' ? 'kube.*' : '*'
    }
    
    const subdomain = parts.slice(0, -2).join('.')
    const prefix = loadBalancerType === 'cluster' ? 'kube.' : '*.'
    
    return `${prefix}${subdomain}`
  }

  function onValueCopied(key: keyof typeof copied, value: string) {
    navigator.clipboard?.writeText(value)

    setCopied((prev) => ({ ...prev, [key]: true }))

    setTimeout(() => {
      setCopied((prev) => ({ ...prev, [key]: false }))
    }, 1000)
  }

  const dnsVerificationMutation = useMutation({
    async mutationFn() {
      const response = await axios.post(
        `/w/${props.workspace.slug}/clusters/${cluster.id}/dns/verify`
      )
      return response.data
    },
  })

  const handleVerifyRecords = () => {
    dnsVerificationMutation.mutate()
  }

  return (
    <div className="w-full p-4">
      <Alert.Root variant="info">
        <Alert.Icon>
          <WarningCircleSolidIcon />
        </Alert.Icon>
        <div className="w-full flex flex-col">
          <Alert.Title>Configure dns to ensure your cluster is accessible.</Alert.Title>

          <Text className="mt-4 !text-sm text-owly-content-tertiary">
            We have provisioned your node balancers. You need to configure your DNS to point to the
            node balancers.
          </Text>

          <div className="flex flex-col gap-0.5 mt-2">
            <Text className="mt-2 !text-sm text-owly-content-tertiary flex items-center gap-2">
              <CheckIcon className="!size-4 text-owly-content-info" />
              The cluster load balancer configures traffic to your kubernetes API.
            </Text>
            <Text className="mt-2 !text-sm text-owly-content-tertiary flex items-center gap-2">
              <CheckIcon className="!size-4 text-owly-content-info" />
              The ingress load balancer configures traffic to your applications.
            </Text>
          </div>
        </div>
      </Alert.Root>

      <div className="mt-6">
        <div className="overflow-x-auto border border-owly-border-tertiary rounded-lg">
          <table className="w-full border-collapse">
            <thead>
              <tr className="border-b border-owly-border-tertiary">
                <th className="text-left py-2 px-4 font-medium text-owly-content-primary">
                  <Text size="sm">Service</Text>
                </th>
                <th className="text-left py-2 px-4 font-medium text-owly-content-primary">
                  <Text size="sm"> Record type</Text>
                </th>
                <th className="text-left py-2 px-4 font-medium text-owly-content-primary">
                  <Text size="sm">Name</Text>
                </th>
                <th className="text-left py-2 px-4 font-medium text-owly-content-primary">
                  <Text size="sm">Value</Text>
                </th>
              </tr>
            </thead>
            <tbody>
              {cluster?.loadBalancers?.map((loadBalancer, idx) => (
                <tr
                  key={loadBalancer?.id}
                  className={classNames({
                    'border-b border-owly-border-tertiary':
                      cluster?.loadBalancers && idx < cluster?.loadBalancers?.length - 1,
                  })}
                >
                  <td className="py-2 px-3">
                    <div className="flex items-center gapy-2 px-3">
                      <div
                        className={classNames('w-2 h-2 rounded-full mr-2', {
                          'bg-owly-background-positive': loadBalancer?.dnsVerifiedAt,
                          'bg-owly-background-notice': !loadBalancer?.dnsVerifiedAt,
                        })}
                      ></div>
                      <Text className="font-medium">{loadBalancer?.type}</Text>
                    </div>
                  </td>
                  <td className="py-2 px-3">
                    <Badge asChild variant="neutral" size="sm" className="!px-2 cursor-pointer">
                      <button>A</button>
                    </Badge>
                  </td>
                  <td className="py-2 px-3">
                    <Badge
                      asChild
                      variant="neutral"
                      size="sm"
                      className="!px-2 cursor-pointer"
                      onClick={() =>
                        onValueCopied(
                          loadBalancer?.type === 'cluster' ? 'clusterName' : 'ingressName',
                          parseSubdomainIdentifier(cluster?.subdomainIdentifier || '', loadBalancer?.type)
                        )
                      }
                    >
                      <button>
                        {parseSubdomainIdentifier(cluster?.subdomainIdentifier || '', loadBalancer?.type)}

                        {copied[
                          loadBalancer?.type === 'cluster' ? 'clusterName' : 'ingressName'
                        ] ? (
                          <CheckIcon />
                        ) : (
                          <CopyIcon />
                        )}
                      </button>
                    </Badge>
                  </td>
                  <td className="py-2 px-3">
                    <Badge
                      asChild
                      variant="neutral"
                      size="sm"
                      className="!px-2 cursor-pointer"
                      onClick={() =>
                        onValueCopied(
                          loadBalancer?.type === 'cluster' ? 'clusterValue' : 'ingressValue',
                          loadBalancer?.publicIpv4Address || ''
                        )
                      }
                    >
                      <button>
                        {loadBalancer?.publicIpv4Address}

                        {copied[
                          loadBalancer?.type === 'cluster' ? 'clusterValue' : 'ingressValue'
                        ] ? (
                          <CheckIcon />
                        ) : (
                          <CopyIcon />
                        )}
                      </button>
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="my-4 flex justify-end">
        <Button
          onClick={handleVerifyRecords}
          loading={dnsVerificationMutation.isPending}
          disabled={dnsVerificationMutation.isPending}
        >
          {dnsVerificationMutation.isPending ? 'Verifying...' : 'Verify records'}
        </Button>
      </div>
    </div>
  )
}
