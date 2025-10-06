import { SidebarPageLayout } from '~/Layouts/SidebarPageLayout'
import { NoWorkspaceCluster } from '../clusters/NoClusters'
import { useForm, usePage } from '@inertiajs/react'
import { Cluster, PageProps } from '~/types'
import { Heading } from '@kibamail/owly/heading'
import { K8sIcon } from '~/Components/Icons/k8s.svg'
import { CreateClusterDialog } from '../clusters/CreateClusterDialog'
import { useState } from 'react'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { Button } from '@kibamail/owly'
import { ClusterCard } from '~/Components/Clusters/ClusterCard'
import { ConfirmDeletionModal } from '~/Components/ConfirmDeletionModal'
import { ClusterProvisioningDialog } from '~/Components/Clusters/ClusterProvisioningDialog'
import { ClusterHetznerRobotProvisioningDialog } from '~/Components/Clusters/ClusterHetznerRobotProvisioningDialog'

export default function Clusters() {
  const { props } = usePage<PageProps>()
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
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
    <SidebarPageLayout>
      <ConfirmDeletionModal
        open={!!selectedDeletionCluster}
        onOpenChange={() => setSelectedDeletionCluster(null)}
        title="Destroy cluster"
        description="This will completely destroy all cluster resources, including volumes, servers, networks, load balancers and all application data and deployments."
        onConfirm={() =>
          form.delete(`/w/${props.workspace.slug}/clusters/${selectedDeletionCluster?.id}`)
        }
        loading={form.processing}
        requireTextMatch={selectedDeletionCluster?.subdomainIdentifier}
        confirmText="Destroy cluster"
      />

      {!props.clusters.length ? (
        <NoWorkspaceCluster />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <Heading size="md" className="flex items-center gap-2">
              <K8sIcon />
              Clusters
            </Heading>

            <Button onClick={() => setCreateDialogOpen(true)}>
              <PlusIcon />
              Create cluster
            </Button>
          </div>

          <p className="text-sm text-owly-content-tertiary mt-6">
            Connected cloud providers are used to provision infrastructure for your clusters. If you
            provision a cluster on your own infrastructure, this will incure additional charges from
            your provider.
          </p>

          <div className="mt-6 flex flex-col gap-3">
            {Object.values(clusters).map((cluster) => (
              <ClusterCard
                key={cluster.id}
                cluster={cluster}
                onClusterSelected={setSelectedCluster}
                onClusterDeletion={setSelectedDeletionCluster}
              />
            ))}
          </div>
        </>
      )}
      <ClusterHetznerRobotProvisioningDialog
        cluster={selectedCluster}
        isOpen={!!selectedCluster && selectedCluster?.cloudProvider?.type === 'hetzner_robot'}
        onClusterUpdated={onClusterUpdated}
        onOpenChange={() => setSelectedCluster(null)}
      />

      <ClusterProvisioningDialog
        cluster={selectedCluster}
        isOpen={!!selectedCluster && selectedCluster?.cloudProvider?.type !== 'hetzner_robot'}
        onClusterUpdated={onClusterUpdated}
        onOpenChange={() => setSelectedCluster(null)}
      />
      <CreateClusterDialog isOpen={createDialogOpen} onOpenChange={setCreateDialogOpen} />
    </SidebarPageLayout>
  )
}
