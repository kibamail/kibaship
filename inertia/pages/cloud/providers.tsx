import { Button } from '@kibamail/owly'
import { Heading } from '@kibamail/owly/heading'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { SidebarPageLayout } from '~/Layouts/SidebarPageLayout'
import { CreateCloudProviderDialog } from '../clusters/CreateCloudProviderDialog'
import { useEffect, useState } from 'react'
import { usePage, usePoll } from '@inertiajs/react'
import { PageProps } from '~/types'
import { NoCloudProviders } from '../clusters/NoCloudProviders'
import { CloudCheckIcon } from '~/Components/Icons/cloud-check.svg'
import { CloudProviderCard } from '~/Components/CloudProviderCard'

export default function CloudProviders() {
  const [createDialogOpen, setCreateDialogOpen] = useState(false)

  const { props } = usePage<PageProps>()

  const { start: startPolling, stop: stopPolling } = usePoll(
    5000,
    {},
    {
      autoStart: false,
    }
  )

  const cloudProvidersPreparing = props.connectedProviders
    .filter((provider) => provider.status === 'preparing')
    .map((provider) => provider.id)

  useEffect(() => {
    if (cloudProvidersPreparing.length > 0) {
      startPolling()
    } else {
      stopPolling()
    }

    return () => stopPolling()
  }, [cloudProvidersPreparing])

  return (
    <SidebarPageLayout>
      {/* {props.workspace.} */}
      {!props.connectedProviders?.length ? (
        <NoCloudProviders />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <Heading size="md" className="flex items-center gap-2">
              <CloudCheckIcon />
              Cloud providers
            </Heading>

            <Button onClick={() => setCreateDialogOpen(true)}>
              <PlusIcon />
              Connect cloud provider
            </Button>
          </div>

          <p className="text-sm text-owly-content-tertiary mt-6">
            Connected cloud providers are used to provision infrastructure for your clusters. If you
            provision a cluster on your own infrastructure, this will incure additional charges from
            your provider.
          </p>

          <div className="mt-6 flex flex-col gap-3">
            {props.connectedProviders.map((provider) => (
              <CloudProviderCard key={provider.id} cloudProvider={provider} />
            ))}
          </div>
        </>
      )}

      <CreateCloudProviderDialog isOpen={createDialogOpen} onOpenChange={setCreateDialogOpen} />
    </SidebarPageLayout>
  )
}
