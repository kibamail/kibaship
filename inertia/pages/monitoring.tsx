import { Head } from '@inertiajs/react'
import { Heading } from '@kibamail/owly/heading'
import { SidebarPageLayout } from '~/Layouts/SidebarPageLayout'

export default function Monitoring() {
  return (
    <SidebarPageLayout>
      <Head title="Monitoring" />
      <div className="px-4 lg:px-0 max-w-6xl mx-auto py-8">
        <Heading size="sm">Monitoring</Heading>
      </div>
    </SidebarPageLayout>
  )
}
