import { BoxIcon } from '~/Components/Icons/box.svg'
import { Button } from '@kibamail/owly'
import { Heading } from '@kibamail/owly/heading'
import { Text } from '@kibamail/owly/text'
import { PlusIcon } from '../Icons/plus.svg'
import { useState } from 'react'
import { CreateApplicationDialog } from '../Applications/CreateApplicationDialog'
import { usePage } from '@inertiajs/react'
import { PageProps } from '~/types'
import { CreateProjectFlow } from './CreateProjectFlow'

export function NoApplicationsInProject() {
  const { props } = usePage<PageProps>()

  const hasNoProjects = props.projects.length == 0

  const [newApplicationCommandOpen, setNewApplicationCommandOpen] = useState(false)
  const [createProjectDialogOpen, setCreateProjectDialogOpen] = useState(false)

  return (
    <div className="w-full h-full kb-background-hover flex flex-col items-center py-24 mt-12 border kb-border-tertiary rounded-lg px-6">
      <div className="flex flex-col items-center">
        <div className="w-24 h-24 rounded-xl flex items-center justify-center bg-white border kb-border-tertiary">
          <BoxIcon className="w-18 h-18 kb-content-positive" />
        </div>

        <div className="mt-4 flex flex-col items-center max-w-lg">
          <Heading size="md" className="font-bold kb-content-secondary">
            {hasNoProjects
              ? 'You have no projects in this workspace.'
              : 'Create your first application'}
          </Heading>

          <Text className="text-center kb-content-tertiary mt-4">
            {hasNoProjects
              ? 'Projects help you organize your application environments, deployments, infrastructure and resources. Applications in a project can talk to each other. To deploy your first application, please create a project.'
              : `You haven't created any applications in this project yet. Applications help you organize
            and manage your environments, deployments, infrastructure and resources.`}
          </Text>
        </div>

        <div className="mt-6">
          <Button
            variant="primary"
            onClick={
              hasNoProjects
                ? () => setCreateProjectDialogOpen(true)
                : () => setNewApplicationCommandOpen(true)
            }
          >
            <PlusIcon className="!size-5" />
            {hasNoProjects ? 'Add new project' : 'Create new application'}
          </Button>
          <CreateProjectFlow
            isOpen={createProjectDialogOpen}
            onOpenChange={setCreateProjectDialogOpen}
          />
          <CreateApplicationDialog
            onOpenChange={setNewApplicationCommandOpen}
            isOpen={newApplicationCommandOpen}
          />
        </div>
      </div>
    </div>
  )
}
