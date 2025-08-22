import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '~/Components/Sheet'
import { useProjectPageContext } from '~/pages/projects/projects-context'
import * as Tabs from '@kibamail/owly/tabs'
import { SettingsIcon } from '../Icons/settings.svg'
import { GraphUpIcon } from '../Icons/graph-up.svg'
import { CodeIcon } from '../Icons/code.svg'
import { CloudCheckIcon } from '../Icons/cloud-check.svg'
import { usePage } from '@inertiajs/react'
import { Drawer } from 'vaul'
import { PageProps } from '~/types'
import { XMarkIcon } from '../Icons/xmark.svg'
import { Deployments } from './Tabs/Deployments/Deployments'

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
    <Drawer.Root direction="right" onOpenChange={onOpenChange} open={isOpen}>
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/5" />
        <Drawer.Content
          className="right-0 top-0 bottom-0 lg:right-2 lg:top-2 lg:bottom-2 fixed z-20 outline-none w-full max-w-6xl flex shadow-lg"
          style={{ '--initial-transform': 'calc(100% + 8px)' } as React.CSSProperties}
        >
          <div className="bg-white h-full w-full grow flex flex-col lg:rounded-[16px]">
            <div className="w-full p-8 lg:px-6">
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center justify-between">
                  <Drawer.Title className="text-owly-content-secondary capitalize text-lg font-bold">
                    {selectedApplication?.name} settings
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
                  </div>

                  <Tabs.Content value="deployments" className="w-full">
                    <Deployments />
                  </Tabs.Content>
                </Tabs.Root>
              </div>
              {/* <Drawer.NestedRoot direction="right">
                <div>
                  <Drawer.Trigger className="rounded-md mt-4 bg-gray-900 px-3.5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-gray-800 focus-visible:outline focus-visible:outline-offset-2 focus-visible:outline-gray-600">
                    Open Second Drawer
                  </Drawer.Trigger>
                </div>
                <Drawer.Portal>
                  <Drawer.Overlay className="fixed inset-0 bg-black/10 z-20" />
                  <Drawer.Content
                    className="right-4 top-6 bottom-6 fixed z-30 outline-none w-full max-w-[68rem] flex"
                    style={{ '--initial-transform': 'calc(100% + 16px)' } as React.CSSProperties}
                  >
                    <div className="bg-white h-full w-full grow p-5 flex flex-col rounded-[16px]">
                      <div className="w-full">
                        <Drawer.Title className="font-medium mb-4 text-gray-900">
                          This drawer is nested.
                        </Drawer.Title>
                        <p className="text-gray-600 mb-2">
                          If you pull this drawer to the left a bit, it&apos;ll scale the drawer
                          underneath it as well.
                        </p>
                      </div>
                    </div>
                  </Drawer.Content>
                </Drawer.Portal>
              </Drawer.NestedRoot> */}
            </div>
          </div>
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  )

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
