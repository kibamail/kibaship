import { Head } from '@inertiajs/react'
import { NoApplicationsInWorkspace } from '~/Components/Dashboard/NoApplications'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'

interface DashboardProps {}

export default function Dashboard() {
  return (
    <AuthenticatedLayout>
      <Head title="Dashboard" />

      <div className="px-4 lg:px-0 max-w-4xl mx-auto">
        <NoApplicationsInWorkspace />
      </div>
    </AuthenticatedLayout>
  )
}
