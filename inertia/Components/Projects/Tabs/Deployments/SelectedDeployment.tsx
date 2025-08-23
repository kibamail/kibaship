import { usePage } from '@inertiajs/react'
import * as Tabs from '@kibamail/owly'
import { Drawer } from 'vaul'
import { XMarkIcon } from '~/Components/Icons/xmark.svg'
import { useProjectPageContext } from '~/pages/projects/projects-context'

export function SelectedDeployment() {
  const { url } = usePage()
  const { selectedDeployment } = useProjectPageContext()

  const activeTab =
    new URL(url, 'https://dummy.com').searchParams.get('deployment_tab') || 'details'

  function onValueChange(tab: string) {
    const currentUrl = new URL(window.location.href)

    currentUrl.searchParams.set('deployment_tab', tab)

    window.history.replaceState({}, '', currentUrl)
  }

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <div className="flex items-center justify-between">
          <Drawer.Title className="text-owly-content-secondary capitalize text-lg font-bold">
            Deployment settings
          </Drawer.Title>

          <Drawer.Close className="ring-offset-background focus:ring-ring data-[state=open]:bg-secondary absolute top-6 right-6 rounded-xs opacity-70 transition-opacity hover:opacity-100 focus:ring-2 focus:ring-offset-2 focus:outline-hidden disabled:pointer-events-none">
            <XMarkIcon className="size-6 text-owly-content-tertiary" />
            <span className="sr-only">Close</span>
          </Drawer.Close>
        </div>
        <p className="text-owly-content-tertiary text-muted-foreground text-sm">
          Lorem ipsum, dolor sit amet consectetur adipisicing elit.
        </p>
      </div>
      <div className="w-full mt-4">
        <Tabs.Root defaultValue={activeTab} onValueChange={onValueChange} width="full">
          <div className="max-w-lg">
            <Tabs.List>
              <Tabs.Trigger value="details">Details</Tabs.Trigger>
              <Tabs.Trigger value="build-logs">Build logs</Tabs.Trigger>
              <Tabs.Trigger value="deploy-logs">Deploy logs</Tabs.Trigger>
              <Tabs.Trigger value="pod-logs">Pod logs</Tabs.Trigger>

              <Tabs.Indicator />
            </Tabs.List>
          </div>

          <Tabs.Content value="deployments" className="w-full">
            <p></p>
          </Tabs.Content>
        </Tabs.Root>
      </div>
    </>
  )
}
