import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '~/Components/Sheet'
import { useProjectPageContext } from '~/pages/projects/projects-context'
import * as Tabs from '@kibamail/owly/tabs'
import { SettingsIcon } from '../Icons/settings.svg'
import { GraphUpIcon } from '../Icons/graph-up.svg'
import { CodeIcon } from '../Icons/code.svg'
import { CloudCheckIcon } from '../Icons/cloud-check.svg'
import { Link, usePage, router } from '@inertiajs/react'
import { PageProps } from '~/types'

interface ProjectSettingsProps {
  isOpen?: boolean
  onOpenChange?: (open: boolean) => void
}

export function ProjectSettings({ isOpen, onOpenChange }: ProjectSettingsProps) {
  const { selectedApplication } = useProjectPageContext()
  const { url } = usePage<PageProps>()

  const activeTab = new URL(url, 'https://dummy.com').searchParams.get('tab') || 'deployments'

  function onValueChange(value: string) {
    const currentUrl = new URL(window.location.href)

    currentUrl.searchParams.set('tab', value)

    window.history.replaceState({}, '', currentUrl)
  }

  return (
    <Sheet onOpenChange={onOpenChange} open={isOpen}>
      <SheetContent>
        <SheetHeader>
          <SheetTitle className="text-owly-content-secondary capitalize text-lg">
            {selectedApplication?.name} settings
          </SheetTitle>
          <SheetDescription className="text-owly-content-tertiary">
            Lorem ipsum, dolor sit amet consectetur adipisicing elit.
          </SheetDescription>
        </SheetHeader>
        <div className="px-6 w-full">
          <Tabs.Root defaultValue={activeTab} onValueChange={onValueChange}>
            <Tabs.List>
              <Tabs.Trigger value="deployments">
                <CloudCheckIcon />
                Deployments
              </Tabs.Trigger>
              <Tabs.Trigger value="variables">
                <CodeIcon />
                Variables
              </Tabs.Trigger>
              <Tabs.Trigger value="metrics">
                <GraphUpIcon />
                Metrics
              </Tabs.Trigger>
              <Tabs.Trigger value="settings">
                <SettingsIcon />
                Settings
              </Tabs.Trigger>

              <Tabs.Indicator />
            </Tabs.List>
          </Tabs.Root>
        </div>
      </SheetContent>
    </Sheet>
  )
}
