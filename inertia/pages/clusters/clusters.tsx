import { Head, usePage } from '@inertiajs/react'
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

export default function Clusters() {
  const { props } = usePage<
    PageProps & {
      connectedProviders: CloudProvider[]
      clusters: Cluster[]
    }
  >()
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)

  const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null)

  return (
    <Authenticated>
      <Head title="Workspace clusters" />

      <div className="px-4 lg:px-0 max-w-6xl pb-12 mx-auto">
        {props.connectedProviders.length === 0 ? <NoCloudProviders /> : null}
        {props.clusters.length === 0 && props.connectedProviders.length > 0 ? (
          <NoWorkspaceCluster />
        ) : null}

        {props.clusters.length > 0 ? (
          <>
            <div className="w-full pt-4 md:pt-6 lg:pt-12">
              <div className="flex w-full justify-between items-center">
                <Heading size="sm">Workspace clusters</Heading>

                <Button onClick={() => setIsCreateModalOpen(true)}>
                  <PlusIcon className="!size-5" />
                  New cluster
                </Button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6 pt-4 md:pt-6 lg:pt-12">
                {props.clusters.map((cluster) => (
                  <ClusterCard
                    cluster={cluster}
                    key={cluster?.id}
                    onClusterSelected={setSelectedCluster}
                  />
                ))}
              </div>
            </div>

            <CreateClusterDialog isOpen={isCreateModalOpen} onOpenChange={setIsCreateModalOpen} />

            <ClusterProvisioningDialog
              cluster={selectedCluster}
              isOpen={!!selectedCluster}
              onOpenChange={() => setSelectedCluster(null)}
            />
          </>
        ) : null}
      </div>
    </Authenticated>
  )
}
