import { RegionSelector } from '~/Components/CloudProviders/RegionSelector'
import * as NumberField from '~/Components/NumberField'
import type { CloudProvider, CloudProviderServerType, CloudProviderType, PageProps } from '~/types'
import { usePage } from '@inertiajs/react'
import { InputError } from '@kibamail/owly/input-hint'
import { InputLabel } from '@kibamail/owly/input-label'
import * as Select from '@kibamail/owly/select-field'
import { useEffect, useMemo } from 'react'
import { Badge } from '@kibamail/owly'

interface CreateCloudProviderClusterProps {
  data: {
    cloud_provider_id: string
    region: string
    control_plane_nodes_count: number
    worker_nodes_count: number
    server_type: string
    control_planes_volume_size: number
    workers_volume_size: number
  }
  setData: any
  errors: Record<string, string>
  connectedProviders: CloudProvider[]
}

export function CreateCloudProviderCluster({
  data,
  setData,
  errors,
  connectedProviders,
}: CreateCloudProviderClusterProps) {
  const { serverTypes, cloudProviderRegions } = usePage<
    PageProps & {
      serverTypes: Record<CloudProviderType, Record<string, CloudProviderServerType>>
    }
  >().props

  const selectedProvider = connectedProviders.find((p) => p.id === data.cloud_provider_id)
  const availableServerTypes = selectedProvider
    ? Object.entries(serverTypes[selectedProvider.type] || {}).map(([slug, specs]) => ({
        slug,
        ...specs,
      }))
    : []

  function onRegionChange(region: string) {
    setData('region', region)
  }

  function onServerTypeChange(serverType: string) {
    setData('server_type', serverType)
  }

  function formatErrorMessage(message: string) {
    return message?.replace('_', ' ')
  }

  const providerType = selectedProvider?.type
  const region = data?.region

  useEffect(() => {
    setData('server_type', '')
  }, [region])

  const availableServers = useMemo(() => {
    if (!providerType) {
      return {}
    }

    if (!region) {
      return {}
    }

    const cloudProvidersByContinent = cloudProviderRegions[providerType] || {}

    const regions = Object.values(cloudProvidersByContinent).flat()

    const selectedRegion = regions.find((_region) => _region.slug === region)

    return selectedRegion?.availableServerTypes || {}
  }, [providerType, region])

  return (
    <div className="px-5 pb-5 space-y-4">
      {selectedProvider && (
        <div>
          <InputLabel htmlFor="region-input">Region</InputLabel>
          <RegionSelector
            providerType={selectedProvider.type}
            selectedRegion={data.region}
            onRegionChange={onRegionChange}
            placeholder="Select a region"
          />
          {errors.region && (
            <InputError baseId="region-input">{formatErrorMessage(errors.region)}</InputError>
          )}
          <input type="hidden" id="region-input" name="region" value={data.region} />
        </div>
      )}

      <NumberField.Root
        name="control_plane_nodes_count"
        value={data.control_plane_nodes_count}
        min={1}
        max={7}
        onChange={(value: number) => setData('control_plane_nodes_count', value)}
      >
        <NumberField.Label>Control Plane Nodes</NumberField.Label>
        <NumberField.Field placeholder="Enter number of control plane nodes">
          <NumberField.DecrementButton />
          <NumberField.IncrementButton />
          <NumberField.Hint>
            Recommended: 1, 3, 5, or 7 nodes for high availability
          </NumberField.Hint>
          {errors.control_plane_nodes_count && (
            <NumberField.Error>
              {formatErrorMessage(errors.control_plane_nodes_count)}
            </NumberField.Error>
          )}
        </NumberField.Field>
      </NumberField.Root>

      <NumberField.Root
        name="worker_nodes_count"
        value={data.worker_nodes_count}
        min={1}
        max={50}
        onChange={(value: number) => setData('worker_nodes_count', value)}
      >
        <NumberField.Label>Worker Nodes</NumberField.Label>
        <NumberField.Field placeholder="Enter number of worker nodes">
          <NumberField.DecrementButton />
          <NumberField.IncrementButton />
          <NumberField.Hint>Minimum 3 worker nodes required</NumberField.Hint>
          {errors.worker_nodes_count && (
            <NumberField.Error>{formatErrorMessage(errors.worker_nodes_count)}</NumberField.Error>
          )}
        </NumberField.Field>
      </NumberField.Root>

      <NumberField.Root
        name="workers_volume_size"
        value={data.workers_volume_size}
        min={10}
        max={500}
        increments={10}
        onChange={(value: number) => setData('workers_volume_size', value)}
      >
        <NumberField.Label>Worker Volume Size (GB)</NumberField.Label>
        <NumberField.Field placeholder="Enter volume size in GB">
          <NumberField.DecrementButton />
          <NumberField.IncrementButton />
          <NumberField.Hint>Storage size for each worker node (10-500 GB)</NumberField.Hint>
          {errors.workers_volume_size && (
            <NumberField.Error>{formatErrorMessage(errors.workers_volume_size)}</NumberField.Error>
          )}
        </NumberField.Field>
      </NumberField.Root>

      {selectedProvider && availableServerTypes.length > 0 && (
        <Select.Root
          name="server_type"
          value={data.server_type}
          onValueChange={onServerTypeChange}
          disabled={!data.region}
        >
          <Select.Label>Node type</Select.Label>
          <Select.Trigger placeholder="Select node type" />
          <Select.Content className="z-100 !max-h-[300px] overflow-y-auto">
            {availableServerTypes.map((serverType) => (
              <Select.Item
                disabled={!availableServers[serverType.slug]}
                key={serverType.slug}
                value={serverType.slug}
                className="flex items-center w-full"
              >
                <span className="flex flex-col">
                  <span className="font-medium text-sm">
                    {serverType.name} -{' '}
                    <Badge size="sm" variant="neutral">
                      {serverType.arch}
                    </Badge>
                  </span>
                </span>
              </Select.Item>
            ))}
          </Select.Content>
          <Select.Hint>
            The available server types depend on the region you select above. Please select a region
            first.
          </Select.Hint>
          {errors.server_type && (
            <Select.Error>{formatErrorMessage(errors.server_type)}</Select.Error>
          )}
        </Select.Root>
      )}
    </div>
  )
}
