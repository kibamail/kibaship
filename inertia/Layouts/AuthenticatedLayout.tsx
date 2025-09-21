import { Topbar } from '~/Components/Dashboard/Topbar'
import { Text } from '@kibamail/owly/text'
import type { PropsWithChildren } from 'react'
import { TopNavigation } from '~/Components/Dashboard/TopNavigation'

export default function Authenticated({ children }: PropsWithChildren) {
  return (
    <div className="w-full h-screen border-l border-r kb-border-tertiary">
      <Topbar />
      <TopNavigation />
      <main className="w-full kb-background-secondary flex flex-col h-[calc(100vh-7.25rem)] overflow-y-hidden">
        <div className="w-full pr-2 flex pl-2 h-full">
          <div className="w-full rounded-lg border border-b kb-border-tertiary overflow-y-auto h-full flex-grow">
            <div className="flex flex-grow w-full h-full">
              <div className="w-full flex-grow">{children}</div>
              {/*<div className="w-full max-w-md p-6 border-l kb-border-tertiary">
                <OnboardingSidebar />
              </div>*/}
            </div>
          </div>
        </div>
      </main>
      <div className="w-screen h-8 px-2 flex items-center justify-between">
        <div className="h-full flex items-center gap-2">
          <div className="w-2 h-2 rounded-full kb-background-positive" />

          <Text>System operational</Text>
        </div>

        <div className="flex items-center gap-6">
          <Text>Docs</Text>
          <Text>Help</Text>
          <Text>Github</Text>
        </div>
      </div>
    </div>
  )
}
