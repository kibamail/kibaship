import { BoxIcon } from '~/Components/Icons/box.svg'
import { Button } from '@kibamail/owly'
import { Heading } from '@kibamail/owly/heading'
import { Text } from '@kibamail/owly/text'
import { PlusIcon } from '../Icons/plus.svg'
import { NewApplicationCommand } from '../Applications/NewApplicationCommand'
import { useState } from 'react'

export function NoApplicationsInWorkspace() {
  const [newApplicationCommandOpen, setNewApplicationCommandOpen] = useState(false)
  return (
    <div className="w-full h-full kb-background-hover flex flex-col items-center py-24 mt-12 border kb-border-tertiary rounded-lg px-6">
      <div className="flex flex-col items-center">
        <div className="w-24 h-24 rounded-xl flex items-center justify-center bg-white border kb-border-tertiary">
          <BoxIcon className="w-18 h-18 kb-content-positive" />
        </div>

        <div className="mt-4 flex flex-col items-center max-w-lg">
          <Heading size="md" className="font-bold kb-content-secondary">
            Create your first application
          </Heading>

          <Text className="text-center kb-content-tertiary mt-4">
            You haven't created any applications in this workspace yet. Applications help you
            organize and manage your application environments, deployments, infrastructure and
            resources.
          </Text>
        </div>

        <div className="mt-6">
          <Button variant="primary" onClick={() => setNewApplicationCommandOpen(true)}>
            <PlusIcon className="!size-5" />
            Create new application
          </Button>
          <NewApplicationCommand
            open={newApplicationCommandOpen}
            onOpenChange={setNewApplicationCommandOpen}
          />
        </div>
      </div>
    </div>
  )
}
