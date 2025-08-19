import { Head, usePage } from '@inertiajs/react'
import { NoApplicationsInWorkspace } from '~/Components/Dashboard/NoApplications'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'
import { PageProps } from '~/types'

export default function Dashboard() {
  const { props } = usePage<PageProps>()

  return (
    <AuthenticatedLayout>
      <Head title="Dashboard" />

      <div className="px-4 lg:px-0 max-w-4xl mx-auto">
        {props?.projects?.length === 0 ? <NoApplicationsInWorkspace /> : null}
      </div>
    </AuthenticatedLayout>
  )
}
