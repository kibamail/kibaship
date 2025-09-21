import { Head } from '@inertiajs/react'
import { Heading } from '@kibamail/owly/heading'
import { SidebarPageLayout } from '~/Layouts/SidebarPageLayout'

export default function Cloud() {
  return (
    <SidebarPageLayout>
      <Head title="Cloud" />
      <div className="px-4 lg:px-0 max-w-6xl mx-auto py-8">
        <Heading size="sm">Cloud</Heading>
      </div>
    </SidebarPageLayout>
  )
}
