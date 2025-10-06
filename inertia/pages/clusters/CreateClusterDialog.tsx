import type { CloudProvider, CloudProviderType, PageProps } from '~/types'
import { CreateBringYourOwnCluster } from './CreateBringYourOwnCluster'
import { CreateCloudProviderCluster } from './CreateCloudProviderCluster'
import { CreateHetznerRobotCluster } from './CreateHetznerRobotCluster'
import { useForm, usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import * as Select from '@kibamail/owly/select-field'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { K8sIcon } from '~/Components/Icons/k8s.svg'
import { providerIcons } from '~/lib/providerIcons'

interface CreateClusterModalProps {
  isOpen: boolean
  provider?: CloudProviderType
  onOpenChange: (open: boolean) => void
}

export function CreateClusterDialog({ isOpen, onOpenChange }: CreateClusterModalProps) {
  const { connectedProviders, workspace } = usePage<
    PageProps & {
      connectedProviders: CloudProvider[]
    }
  >().props

  const { data, setData, post, processing, errors, reset } = useForm({
    subdomain_identifier: '',
    cloud_provider_id: connectedProviders.length === 1 ? connectedProviders[0].id : '',
    provider: connectedProviders.length === 1 ? connectedProviders[0].id : null,
    region: '',
    control_plane_nodes_count: 3,
    worker_nodes_count: 3,
    server_type: '',
    control_planes_volume_size: 0,
    workers_volume_size: 20,
    robot_server_numbers: [] as number[],
    robot_cloud_provider_id: '',
  })

  function onCloudProviderChange(providerId: string) {
    if (providerId === 'bring_your_own') {
      setData((data) => ({
        ...data,
        provider: null,
        cloud_provider_id: '',
        region: '',
        server_type: '',
      }))
    } else {
      setData((data) => ({
        ...data,
        provider: providerId,
        cloud_provider_id: providerId,
        region: '',
        server_type: '',
      }))
    }
  }

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const selectedProvider = connectedProviders.find((p) => p.id === data.cloud_provider_id)
    const isHetznerRobot = selectedProvider?.type === 'hetzner_robot'

    const endpoint = isHetznerRobot
      ? `/w/${workspace.slug}/clusters/hetzner-robot`
      : `/w/${workspace.slug}/clusters`

    post(endpoint, {
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
      <Dialog.Content className="!max-w-[640px]">
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
          <div className="px-5 pb-2 space-y-4">
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
              name="provider"
              value={data.provider || 'bring_your_own'}
              onValueChange={onCloudProviderChange}
              disabled={connectedProviders.length === 0}
            >
              <Select.Label>Cloud provider</Select.Label>
              <Select.Trigger
                placeholder={
                  connectedProviders.length === 0
                    ? 'No cloud providers available'
                    : 'Select a cloud provider'
                }
              />
              <Select.Content className="z-100 !max-h-[300px] overflow-y-auto">
                <Select.Item value="bring_your_own">
                  <div className="flex items-center gap-2">
                    <K8sIcon className="w-4 h-4" />
                    <span>Bring your own cluster</span>
                  </div>
                </Select.Item>
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
              {errors.provider && (
                <Select.Error>{formatErrorMessage(errors.provider)}</Select.Error>
              )}
            </Select.Root>
          </div>

          {data.provider === null ? (
            <CreateBringYourOwnCluster
              onSubmit={() => {
                reset()
                onOpenChange(false)
              }}
            />
          ) : connectedProviders.find((p) => p.id === data.cloud_provider_id)?.type ===
            'hetzner_robot' ? (
            <CreateHetznerRobotCluster
              data={data}
              setData={setData}
              errors={errors}
              connectedProviders={connectedProviders}
            />
          ) : (
            <CreateCloudProviderCluster
              data={data}
              setData={setData}
              errors={errors}
              connectedProviders={connectedProviders}
            />
          )}

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild disabled={processing}>
              <Button variant="secondary">Cancel</Button>
            </Dialog.Close>
            <Button
              type="submit"
              loading={processing}
              disabled={
                processing ||
                (connectedProviders.find((p) => p.id === data.cloud_provider_id)?.type ===
                  'hetzner_robot' &&
                  (!data.robot_server_numbers || data.robot_server_numbers.length < 3))
              }
            >
              Create Cluster
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}
