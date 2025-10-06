import { Text } from '@kibamail/owly/text'
import { useForm, usePage } from '@inertiajs/react'
import { CloudProvider, PageProps } from '~/types'
import { providerIcons } from '~/lib/providerIcons'
import { OptionsDropdown } from './OptionsDropdown'
import { TrashIcon } from './Icons/trash.svg'
import { useState } from 'react'
import { ConfirmDeletionModal } from './ConfirmDeletionModal'
import { Tooltip } from './Tooltip'
import Spinner from './Icons/Spinner'
import { WarningTriangleSolidIcon } from './Icons/warning-triangle-solid.svg'

interface ProjectCardProps {
  cloudProvider: CloudProvider
}

export function CloudProviderCard({ cloudProvider }: ProjectCardProps) {
  const { props } = usePage<PageProps>()
  const form = useForm({})
  const [open, setOpen] = useState(false)

  const Icon = providerIcons[cloudProvider.type]

  const clustersForProvider = props?.clusters?.filter(
    (cluster) => cluster.cloudProviderId === cloudProvider.id
  )

  return (
    <>
      <ConfirmDeletionModal
        open={open}
        onOpenChange={setOpen}
        loading={form.processing}
        title="Delete cloud provider"
        description="This will remove the cloud provider from your workspace. Existing clusters will not be deleted, but you will no longer be able to use this provider to provision new clusters"
        onConfirm={() => form.delete(`/connections/cloud-providers/${cloudProvider.id}`)}
      />

      <div className="w-full bg-white rounded-[11px] min-h-[40px] border border-owly-border-tertiary relative group">
        <div className="absolute w-[calc(100%+10px)] h-[calc(100%+10px)] rounded-[14px] transform translate-x-[-5px] translate-y-[-5px] bg-transparent transition-all ease-in-out border border-transparent group-hover:border-owly-border-tertiary"></div>
        <div className="px-4 py-3 z-1 flex flex-col justify-between h-full relative rounded-[11px]">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Icon />
              <Text className="text-owly-content-secondary font-semibold">
                {cloudProvider?.name}
              </Text>
            </div>

            <div className="flex items-center gap-1 self-end">
              {cloudProvider.status === 'preparing' ? (
                <Tooltip
                  content={
                    <div className="flex flex-col gap-4">
                      <Text>We are uploading the talos os image to your account.</Text>

                      <Text className="text-owly-content-tertiary-inverse">
                        Some cloud providers like Hetzner require that we pre upload the talos os
                        image to your account before using it to provision cluster.
                      </Text>

                      <Text className="text-owly-content-tertiary-inverse">
                        Once done, you will be able to use this provider to provision a cluster.
                        This will only take a few minutes.
                      </Text>
                    </div>
                  }
                >
                  <button>
                    <Spinner className="size-4" />
                  </button>
                </Tooltip>
              ) : cloudProvider.status === 'failed' ? (
                <Tooltip
                  content={
                    <div className="flex flex-col gap-4">
                      <Text>We couldn't upload the talos os image to your provider account.</Text>

                      <Text className="text-owly-content-tertiary-inverse">
                        Please reach out to us to find out why this might have happened.
                      </Text>

                      <Text className="text-owly-content-tertiary-inverse">
                        A faster way would be to delete this provider and try adding it again.
                      </Text>
                    </div>
                  }
                >
                  <button>
                    <WarningTriangleSolidIcon className="text-owly-content-notice" />
                  </button>
                </Tooltip>
              ) : (
                <Tooltip
                  content={
                    clustersForProvider.length === 0 ? (
                      <Text className="text-owly-content-tertiary-inverse">
                        You have not provisioned any clusters on this provider yet.
                      </Text>
                    ) : (
                      <div className="flex flex-col gap-2">
                        <Text className="text-owly-content-tertiary-inverse">
                          The following clusters were provisioned using this cloud provider:
                        </Text>

                        <ul className="list-disc pl-4">
                          {clustersForProvider?.map((cluster) => (
                            <li key={cluster.id}>{cluster.subdomainIdentifier}</li>
                          ))}
                        </ul>
                      </div>
                    )
                  }
                >
                  <Text className="text-owly-content-secondary font-semibold">
                    {clustersForProvider?.length > 0 ? clustersForProvider?.length : 'No'} cluster
                    {clustersForProvider?.length === 1 ? '' : 's'}
                  </Text>
                </Tooltip>
              )}

              <OptionsDropdown
                id={`cloud-provider-card-dropdown-menu-content-${cloudProvider.id}`}
                items={[
                  {
                    icon: TrashIcon,
                    name: 'Delete',
                    onClick: () => setOpen(true),
                    type: 'destructive',
                  },
                ]}
              />
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
