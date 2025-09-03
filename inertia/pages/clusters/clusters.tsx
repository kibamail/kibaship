import { Head, useForm, usePage } from '@inertiajs/react'
import Authenticated from '~/Layouts/AuthenticatedLayout'
import { NoCloudProviders } from './NoCloudProviders'
import { CloudProvider, Cluster, PageProps } from '~/types'
import { NoWorkspaceCluster } from './NoClusters'
import { ClusterCard } from '~/Components/Clusters/ClusterCard'
import { Heading } from '@kibamail/owly/heading'
import { Button } from '@kibamail/owly'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { useState } from 'react'
import { CreateClusterDialog } from './CreateClusterDialog'
import { ClusterProvisioningDialog } from '~/Components/Clusters/ClusterProvisioningDialog'
import * as Dialog from '~/Components/DialogWithConfirmation'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { Text } from '@kibamail/owly/text'
import { WarningCircleSolidIcon } from '~/Components/Icons/warning-circle-solid.svg'

export default function Clusters() {
  const { props } = usePage<
    PageProps & {
      connectedProviders: CloudProvider[]
      clusters: Cluster[]
    }
  >()
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)

  const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null)
  const [selectedDeletionCluster, setSelectedDeletionCluster] = useState<Cluster | null>(null)
  const [clusters, setClusters] = useState<Record<string, Cluster>>(() => {
    const initialClusters: Record<string, Cluster> = {}
    props.clusters.forEach((cluster) => {
      initialClusters[cluster.id] = cluster
    })
    return initialClusters
  })

  function onClusterUpdated(updatedCluster: Cluster) {
    setClusters((prevClusters) => ({
      ...prevClusters,
      [updatedCluster.id]: updatedCluster,
    }))
  }

  const form = useForm({})

  return (
    <Authenticated>
      <Head title="Workspace clusters" />

      <Dialog.Root
        open={!!selectedDeletionCluster}
        onOpenChange={() => setSelectedDeletionCluster(null)}
      >
        <Dialog.Content>
          <Dialog.Header>
            <Dialog.Title>Destroy cluster</Dialog.Title>
            <VisuallyHidden>
              <Dialog.Description>Destroy cluster</Dialog.Description>
            </VisuallyHidden>
          </Dialog.Header>

          <div className="px-6 w-full pt-6 flex flex-col gap-2">
            <div className="flex flex-col gap-2 justify-center items-center">
              <div className="h-12 w-12 rounded-md flex items-center bg-owly-border-negative/10 justify-center border border-owly-border-negative">
                <WarningCircleSolidIcon className="text-owly-content-negative" />
              </div>
              <Heading size="sm" className="text-owly-content-primary">
                Are you sure you want to destroy this cluster?
              </Heading>
            </div>

            <Text className="text-owly-content-tertiary text-center">
              This will completely destroy all cluster resources, including volumes, servers,
              networks, load balancers and all application data and deployments.
            </Text>
          </div>

          <Dialog.Confirmation
            onDelete={() => {
              form.delete(`/w/${props.workspace.slug}/clusters/${selectedDeletionCluster?.id}`)
            }}
            processing={form.processing}
            value={selectedDeletionCluster?.subdomainIdentifier}
          />
        </Dialog.Content>
      </Dialog.Root>

      <div className="px-4 lg:px-0 max-w-6xl pb-12 mx-auto">
        {props.connectedProviders.length === 0 ? <NoCloudProviders /> : null}
        {props.clusters.length === 0 && props.connectedProviders.length > 0 ? (
          <NoWorkspaceCluster />
        ) : null}

        {props.clusters.length > 0 ? (
          <>
            <div className="px-6 w-full pt-4 md:pt-6 lg:pt-12">
              <div className="flex w-full justify-between items-center">
                <Heading size="sm">Workspace clusters</Heading>

                <Button onClick={() => setIsCreateModalOpen(true)}>
                  <PlusIcon className="!size-5" />
                  New cluster
                </Button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6 pt-4 md:pt-6 lg:pt-12">
                {Object.values(clusters).map((cluster) => (
                  <ClusterCard
                    cluster={cluster}
                    key={cluster?.id}
                    onClusterSelected={setSelectedCluster}
                    onClusterDeletion={setSelectedDeletionCluster}
                  />
                ))}
              </div>
            </div>

            <CreateClusterDialog isOpen={isCreateModalOpen} onOpenChange={setIsCreateModalOpen} />

            <ClusterProvisioningDialog
              cluster={selectedCluster}
              isOpen={!!selectedCluster}
              onClusterUpdated={onClusterUpdated}
              onOpenChange={() => setSelectedCluster(null)}
            />
          </>
        ) : null}
      </div>
    </Authenticated>
  )
}
