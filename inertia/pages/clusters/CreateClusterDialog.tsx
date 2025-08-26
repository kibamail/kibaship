import { RegionSelector } from '~/Components/CloudProviders/RegionSelector'
import { AWSIcon } from '~/Components/Icons/aws.svg'
import { DigitalOceanIcon } from '~/Components/Icons/digital-ocean.svg'
import { GoogleCloudIcon } from '~/Components/Icons/google-cloud.svg'
import { HetznerIcon } from '~/Components/Icons/hetzner.svg'
import { LeaseWebIcon } from '~/Components/Icons/leaseweb.svg'
import { LinodeIcon } from '~/Components/Icons/linode.svg'
import { OVHIcon } from '~/Components/Icons/ovh.svg'
import { VultrIcon } from '~/Components/Icons/vultr.svg'
import * as NumberField from '~/Components/NumberField'
import type { CloudProvider, CloudProviderServerType, CloudProviderType, PageProps } from '~/types'
import { useForm, usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import { InputError } from '@kibamail/owly/input-hint'
import { InputLabel } from '@kibamail/owly/input-label'
import * as Select from '@kibamail/owly/select-field'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'

interface CreateClusterModalProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
}

const providerIcons: Record<CloudProviderType, React.ComponentType<{ className?: string }>> = {
  aws: AWSIcon,
  hetzner: HetznerIcon,
  leaseweb: LeaseWebIcon,
  google_cloud: GoogleCloudIcon,
  digital_ocean: DigitalOceanIcon,
  linode: LinodeIcon,
  vultr: VultrIcon,
  ovh: OVHIcon,
}

export function CreateClusterDialog({ isOpen, onOpenChange }: CreateClusterModalProps) {
  const { connectedProviders, serverTypes, workspace } = usePage<
    PageProps & {
      connectedProviders: CloudProvider[]
      serverTypes: Record<CloudProviderType, Record<string, CloudProviderServerType>>
    }
  >().props

  const { data, setData, post, processing, errors, reset } = useForm({
    subdomain_identifier: '',
    cloud_provider_id: connectedProviders.length === 1 ? connectedProviders[0].id : '',
    region: '',
    control_plane_nodes_count: 3,
    worker_nodes_count: 3,
    server_type: '',
    control_planes_volume_size: 20,
    workers_volume_size: 20,
  })

  const selectedProvider = connectedProviders.find((p) => p.id === data.cloud_provider_id)
  const availableServerTypes = selectedProvider
    ? Object.entries(serverTypes[selectedProvider.type] || {}).map(([slug, specs]) => ({
        slug,
        ...specs,
      }))
    : []

  function onCloudProviderChange(providerId: string) {
    setData((data) => ({
      ...data,
      cloud_provider_id: providerId,
      region: '',
      server_type: '',
    }))
  }

  function onRegionChange(region: string) {
    setData('region', region)
  }

  function onServerTypeChange(serverType: string) {
    setData('server_type', serverType)
  }

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    post(`/w/${workspace.slug}/clusters`, {
      onSuccess: () => {
        reset()
        onOpenChange(false)
      },
    })
  }

  function formatErrorMessage(message: string) {
    return message?.replace('_', ' ')
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>Create New Cluster</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Set up a new Nomad cluster to run your applications
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="kb-content-secondary text-sm leading-relaxed">
            Set up a new Nomad cluster to run your applications. Configure your cluster
            specifications and deployment settings.
          </Text>
        </div>

        <form onSubmit={onSubmit}>
          <div className="px-5 pb-5 space-y-4">
            <TextField.Root
              name="name"
              value={data.subdomain_identifier}
              onChange={(e) => setData('subdomain_identifier', e.target.value)}
              placeholder="eu.kibaship.com"
              required
            >
              <TextField.Label>Cluster domain</TextField.Label>
              {errors.subdomain_identifier && (
                <TextField.Error>{formatErrorMessage(errors.subdomain_identifier)}</TextField.Error>
              )}
              <TextField.Hint>
                All the applications you provision in this cluster will have a subdomain under this
                domain.
              </TextField.Hint>
            </TextField.Root>

            <Select.Root
              name="cloud_provider_id"
              value={data.cloud_provider_id}
              onValueChange={onCloudProviderChange}
              disabled={connectedProviders.length === 0}
            >
              <Select.Label>Cloud Provider</Select.Label>
              <Select.Trigger
                placeholder={
                  connectedProviders.length === 0
                    ? 'No cloud providers available'
                    : 'Select a cloud provider'
                }
              />
              <Select.Content className="z-100">
                {connectedProviders.map((provider) => {
                  const IconComponent = providerIcons[provider.type]
                  return (
                    <Select.Item key={provider.id} value={provider.id}>
                      <div className="flex items-center gap-2">
                        <IconComponent className="w-4 h-4" />
                        <span>{provider.name}</span>
                      </div>
                    </Select.Item>
                  )
                })}
              </Select.Content>
              {errors.cloud_provider_id && (
                <Select.Error>{formatErrorMessage(errors.cloud_provider_id)}</Select.Error>
              )}
            </Select.Root>

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
              min={3}
              max={50}
              onChange={(value: number) => setData('worker_nodes_count', value)}
            >
              <NumberField.Label>Worker Nodes</NumberField.Label>
              <NumberField.Field placeholder="Enter number of worker nodes">
                <NumberField.DecrementButton />
                <NumberField.IncrementButton />
                <NumberField.Hint>Minimum 3 worker nodes required</NumberField.Hint>
                {errors.worker_nodes_count && (
                  <NumberField.Error>
                    {formatErrorMessage(errors.worker_nodes_count)}
                  </NumberField.Error>
                )}
              </NumberField.Field>
            </NumberField.Root>

            <NumberField.Root
              name="control_planes_volume_size"
              value={data.control_planes_volume_size}
              min={10}
              max={500}
              increments={10}
              onChange={(value: number) => setData('control_planes_volume_size', value)}
            >
              <NumberField.Label>Control Plane Volume Size (GB)</NumberField.Label>
              <NumberField.Field placeholder="Enter volume size in GB">
                <NumberField.DecrementButton />
                <NumberField.IncrementButton />
                <NumberField.Hint>
                  Storage size for each control plane node (10-500 GB)
                </NumberField.Hint>
                {errors.control_planes_volume_size && (
                  <NumberField.Error>
                    {formatErrorMessage(errors.control_planes_volume_size)}
                  </NumberField.Error>
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
                  <NumberField.Error>
                    {formatErrorMessage(errors.workers_volume_size)}
                  </NumberField.Error>
                )}
              </NumberField.Field>
            </NumberField.Root>

            {selectedProvider && availableServerTypes.length > 0 && (
              <Select.Root
                name="server_type"
                value={data.server_type}
                onValueChange={onServerTypeChange}
              >
                <Select.Label>Node type</Select.Label>
                <Select.Trigger placeholder="Select node type" />
                <Select.Content className="z-100">
                  {availableServerTypes.map((serverType) => (
                    <Select.Item key={serverType.slug} value={serverType.slug}>
                      <div className="flex flex-col">
                        <span className="font-medium">{serverType.name}</span>
                      </div>
                    </Select.Item>
                  ))}
                </Select.Content>
                {errors.server_type && (
                  <Select.Error>{formatErrorMessage(errors.server_type)}</Select.Error>
                )}
              </Select.Root>
            )}
          </div>

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild disabled={processing}>
              <Button variant="secondary">Cancel</Button>
            </Dialog.Close>
            <Button type="submit" loading={processing}>
              Create Cluster
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}
