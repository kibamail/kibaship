import { Head, usePage } from '@inertiajs/react'
import Authenticated from '~/Layouts/AuthenticatedLayout'
import { NoCloudProviders } from './NoCloudProviders'
import { CloudProvider, PageProps } from '~/types'
import { NoWorkspaceCluster } from './NoClusters'

export default function Clusters() {
  const { props } = usePage<
    PageProps & {
      connectedProviders: CloudProvider[]
    }
  >()

  return (
    <Authenticated>
      <Head title="Workspace clusters" />

      <div className="px-4 lg:px-0 max-w-6xl pb-12 mx-auto">
        {props.connectedProviders.length === 0 ? <NoCloudProviders /> : null}
        <NoWorkspaceCluster />
      </div>
    </Authenticated>
  )
}
