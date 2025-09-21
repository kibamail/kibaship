import { Head } from '@inertiajs/react'
import { Heading } from '@kibamail/owly/heading'
import { SidebarPageLayout } from '~/Layouts/SidebarPageLayout'

export default function Integrations() {
  return (
    <SidebarPageLayout>
      <Head title="Integrations" />
      <div className="px-4 lg:px-0 max-w-6xl mx-auto py-8">
        <Heading size="sm">Integrations</Heading>
      </div>
    </SidebarPageLayout>
  )
}
