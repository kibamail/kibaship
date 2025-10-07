import type { CloudProvider } from '~/types'
import * as MultiSelect from '@kibamail/owly/multi-select'
import * as Select from '@kibamail/owly/select-field'
import { Text } from '@kibamail/owly/text'
import { useQuery } from '@tanstack/react-query'
import Spinner from '~/Components/Icons/Spinner'
import Refresh from '~/Components/Icons/Refresh'
import { RegionSelector } from '~/Components/CloudProviders/RegionSelector'
import { InputError, InputHint } from '@kibamail/owly/input-hint'
import { Button } from '@kibamail/owly/button'
import { useState } from 'react'

interface HetznerRobotServer {
  server_ip: string
  server_ipv6_net?: string
  server_number: number
  server_name: string
  product: string
  dc: string
  traffic: string
  status: 'ready' | 'in process'
  cancelled: boolean
  paid_until: string
  ip: string[]
  subnet?: Array<{ ip?: string; mask?: string }> | null
}

interface HetznerRobotVSwitch {
  id: number
  name: string
  vlan: number
  cancelled: boolean
}

interface CreateHetznerRobotClusterProps {
  data: {
    subdomain_identifier: string
    cloud_provider_id: string
    robot_cloud_provider_id?: string
    region?: string
    robot_server_numbers?: number[]
    robot_vswitch_id?: number | 'create_new'
  }
  setData: any
  errors: Record<string, string>
  connectedProviders: CloudProvider[]
}

export function CreateHetznerRobotCluster({
  data,
  setData,
  errors,
  connectedProviders,
}: CreateHetznerRobotClusterProps) {
  const [cache, setCache] = useState({
    servers: false,
    vswitches: false,
  })
  const selectedProvider = connectedProviders.find((p) => p.id === data.cloud_provider_id)

  const {
    data: servers = [],
    isLoading: loadingServers,
    refetch: refetchServers,
  } = useQuery<HetznerRobotServer[]>({
    queryKey: ['hetzner-robot-servers', selectedProvider?.id, cache],
    queryFn: async () => {
      if (!selectedProvider || selectedProvider.type !== 'hetzner_robot') {
        return []
      }
      const response = await fetch(
        `/connections/cloud-providers/${selectedProvider.id}/servers?clearCache=${cache.servers}`
      )
      if (!response.ok) {
        throw new Error('Failed to fetch servers')
      }
      return response.json()
    },
    enabled: !!selectedProvider && selectedProvider.type === 'hetzner_robot',
  })

  const {
    data: vswitches = [],
    isLoading: loadingVswitches,
    refetch: refetchVswitches,
  } = useQuery<HetznerRobotVSwitch[]>({
    queryKey: ['hetzner-robot-vswitches', selectedProvider?.id, cache],
    queryFn: async () => {
      if (!selectedProvider || selectedProvider.type !== 'hetzner_robot') {
        return []
      }
      const response = await fetch(
        `/connections/cloud-providers/${selectedProvider.id}/vswitches?clearCache=${cache.vswitches}`
      )
      if (!response.ok) {
        throw new Error('Failed to fetch vswitches')
      }
      return response.json()
    },
    enabled: !!selectedProvider && selectedProvider.type === 'hetzner_robot',
  })

  const handleServerChange = (values: string[]) => {
    setData(
      'robot_server_numbers',
      values.map((v) => parseInt(v, 10))
    )
  }

  const handleVSwitchChange = (value: string) => {
    if (value === 'create_new') {
      setData('robot_vswitch_id', undefined)
    } else {
      setData('robot_vswitch_id', parseInt(value, 10))
    }
  }

  function formatErrorMessage(message: string) {
    return message?.replace('_', ' ')
  }

  const hetznerCloudProviders = connectedProviders.filter((p) => p.type === 'hetzner')

  return (
    <div className="px-5 pb-5 space-y-4">
      <Select.Root
        name="robot_cloud_provider_id"
        value={data.robot_cloud_provider_id || ''}
        onValueChange={(value) => setData('robot_cloud_provider_id', value)}
      >
        <Select.Label>Hetzner cloud provider</Select.Label>
        <Select.Trigger
          placeholder={
            hetznerCloudProviders.length === 0
              ? 'No Hetzner Cloud providers available'
              : 'Select a Hetzner Cloud provider'
          }
        />
        <Select.Content className="z-100 !max-h-[300px] overflow-y-auto">
          {hetznerCloudProviders.map((provider) => (
            <Select.Item key={provider.id} value={provider.id}>
              <div className="flex items-center gap-2">
                <span className="font-medium">{provider.name}</span>
              </div>
            </Select.Item>
          ))}
        </Select.Content>
        <Select.Hint>
          A Hetzner cloud provider is required to provision the ingress load balancer for your
          cluster.
        </Select.Hint>
        {errors.robot_cloud_provider_id && (
          <Select.Error>{formatErrorMessage(errors.robot_cloud_provider_id)}</Select.Error>
        )}
      </Select.Root>

      <div className="space-y-2">
        <Select.Label>Region</Select.Label>
        <RegionSelector
          providerType="hetzner"
          selectedRegion={data.region}
          onRegionChange={(region) => setData('region', region)}
          placeholder="Select a region"
        />
        <InputHint baseId="region-input">
          Select the region where the ingress load balancer will be deployed.
        </InputHint>
        {errors.region && (
          <InputError baseId="region-input">{formatErrorMessage(errors.region)}</InputError>
        )}
      </div>

      <MultiSelect.Root
        name="robot_server_numbers"
        value={data.robot_server_numbers?.map((n) => n.toString()) || []}
        onValueChange={handleServerChange}
        disabled={loadingServers}
      >
        <div className="flex items-center justify-between">
          <MultiSelect.Label>Dedicated servers</MultiSelect.Label>

          <Button
            type="button"
            variant="tertiary"
            size="sm"
            onClick={() => setCache((current) => ({ ...current, servers: true }))}
            disabled={loadingServers}
            className="h-7 px-2"
          >
            <Refresh className={`w-4 h-4 ${loadingServers ? 'animate-spin' : ''}`} />
          </Button>
        </div>
        <MultiSelect.Trigger
          placeholder={loadingServers ? 'Loading servers...' : 'Select servers'}
        />
        <MultiSelect.Content className="z-100 !max-h-[300px] overflow-y-auto">
          {servers.map((server) => (
            <MultiSelect.Item key={server.server_number} value={server.server_number.toString()}>
              <div className="flex items-center justify-between w-full">
                <div className="flex flex-col">
                  <span className="font-medium text-owly-content-secondary">
                    {server.server_name}
                    <span className="text-xs kb-content-tertiary ml-1 inline-block">
                      • {server.product} • {server.dc} • {server.server_ip}
                    </span>
                  </span>
                </div>
              </div>
            </MultiSelect.Item>
          ))}
        </MultiSelect.Content>
        <MultiSelect.Hint>
          Select at least 3 dedicated servers from your Hetzner Robot account to provision your
          cluster.
        </MultiSelect.Hint>
        {errors.robot_server_numbers && (
          <MultiSelect.Error>{formatErrorMessage(errors.robot_server_numbers)}</MultiSelect.Error>
        )}
      </MultiSelect.Root>

      {loadingServers && (
        <div className="flex items-center gap-2 py-2">
          <Spinner className="w-4 h-4 kb-content-tertiary" />
          <Text className="kb-content-tertiary text-sm">
            Fetching servers from your Hetzner Robot account...
          </Text>
        </div>
      )}

      {loadingVswitches && (
        <div className="flex items-center gap-2 py-2">
          <Spinner className="w-4 h-4 kb-content-tertiary" />
          <Text className="kb-content-tertiary text-sm">
            Fetching vSwitches from your Hetzner Robot account...
          </Text>
        </div>
      )}

      {data.robot_server_numbers && data.robot_server_numbers.length > 0 && (
        <div className="w-full flex flex-col p-3 rounded-md bg-owly-background-secondary border border-owly-border-tertiary">
          <Text className="text-left text-owly-content-secondary font-semibold mb-2">
            Selected servers
          </Text>

          <div className="space-y-2">
            {data.robot_server_numbers.map((serverNumber) => {
              const server = servers.find((s) => s.server_number === serverNumber)
              if (!server) return null

              return (
                <div
                  key={server.server_number}
                  className="flex items-center justify-between p-2 rounded-md bg-white border border-owly-border-tertiary"
                >
                  <div className="flex items-center">
                    <Text className="font-medium text-sm">{server.server_name}</Text>
                    <span className="text-xs text-owly-content-tertiary inline-block ml-1">
                      {server.product} • {server.dc} • {server.server_ip}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      <Select.Root
        name="robot_vswitch_id"
        value={data.robot_vswitch_id?.toString() || ''}
        onValueChange={handleVSwitchChange}
        disabled={loadingVswitches}
      >
        <div className="flex items-center justify-between">
          <Select.Label>Vswitch</Select.Label>
          <Button
            type="button"
            variant="tertiary"
            size="sm"
            onClick={() => setCache((current) => ({ ...current, vswitches: true }))}
            disabled={loadingVswitches}
            className="h-7 px-2"
          >
            <Refresh className={`w-4 h-4 ${loadingVswitches ? 'animate-spin' : ''}`} />
          </Button>
        </div>
        <Select.Trigger
          placeholder={loadingVswitches ? 'Loading vSwitches...' : 'Select a vSwitch'}
        />
        <Select.Content className="z-100 !max-h-[300px] overflow-y-auto">
          <Select.Item value="create_new">
            <div className="flex items-center gap-2">
              <span className="font-medium">Create new vswitch</span>
              <span className="text-sm kb-content-tertiary">
                We'll automatically create a new vswitch during provisioning
              </span>
            </div>
          </Select.Item>
          {vswitches.map((vswitch) => (
            <Select.Item key={vswitch.id} value={vswitch.id.toString()}>
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center">
                  <span className="font-medium">{vswitch.name}</span>
                  <span className="text-xs inline-block ml-1 kb-content-tertiary">
                    VLAN {vswitch.vlan}
                  </span>
                </div>
              </div>
            </Select.Item>
          ))}
        </Select.Content>
        <Select.Hint>
          Select an existing vswitch or create a new one for your cluster networking.
        </Select.Hint>
        {errors.robot_vswitch_id && (
          <Select.Error>{formatErrorMessage(errors.robot_vswitch_id)}</Select.Error>
        )}
      </Select.Root>
    </div>
  )
}
