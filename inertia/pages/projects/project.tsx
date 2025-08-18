import { Head } from '@inertiajs/react'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'

export default function Project() {
  return (
    <AuthenticatedLayout>
      <Head title="Project" />

      <h1 className="font-sans kb-content-secondary">This is the project screen</h1>
    </AuthenticatedLayout>
  )
}
