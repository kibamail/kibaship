import { Head, usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly'
import { Heading } from '@kibamail/owly/heading'
import { useState } from 'react'
import { NewApplicationCommand } from '~/Components/Applications/NewApplicationCommand'
import { NoApplicationsInWorkspace } from '~/Components/Dashboard/NoApplications'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { ProjectCard } from '~/Components/Projects/ProjectCard'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'
import { PageProps } from '~/types'

export default function Dashboard() {
  const { props } = usePage<PageProps>()
  const [newApplicationCommandOpen, setNewApplicationCommandOpen] = useState(false)

  return (
    <AuthenticatedLayout>
      <Head title="Dashboard" />

      <div className="px-4 lg:px-0 max-w-6xl mx-auto">
        {props?.projects?.length === 0 ? (
          <NoApplicationsInWorkspace />
        ) : (
          <div className="w-full pt-4 md:pt-6 lg:pt-12">
            <div className="flex w-full justify-between items-center">
              <Heading size="sm">Workspace projects</Heading>

              <Button onClick={() => setNewApplicationCommandOpen(true)}>
                <PlusIcon className="!size-5" />
                New application
              </Button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6 pt-4 md:pt-6 lg:pt-12">
              {props?.projects?.map((project) => (
                <ProjectCard key={project?.id} project={project} />
              ))}
            </div>
          </div>
        )}
      </div>

      <NewApplicationCommand
        open={newApplicationCommandOpen}
        onOpenChange={setNewApplicationCommandOpen}
      />
    </AuthenticatedLayout>
  )
}
