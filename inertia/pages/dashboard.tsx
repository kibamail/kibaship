import { Head } from '@inertiajs/react'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'

export default function Dashboard() {
  return (
    <AuthenticatedLayout>
      <Head title="Dashboard" />

      <div className="w-full max-w-6xl mx-auto py-12">
        <h1>The dashboard. Coming soon.</h1>
      </div>
    </AuthenticatedLayout>
  )
}
