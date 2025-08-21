'use client'

import { Drawer } from 'vaul'

export function VaulDrawer() {
  return (
    <Drawer.Root direction="right">
      <Drawer.Trigger className="relative flex h-10 flex-shrink-0 items-center justify-center gap-2 overflow-hidden rounded-full bg-white px-4 text-sm font-medium shadow-sm transition-all hover:bg-[#FAFAFA] dark:bg-[#161615] dark:hover:bg-[#1A1A19] dark:text-white">
        Open Drawer
      </Drawer.Trigger>
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/40" />
        <Drawer.Content
          className="right-2 top-2 bottom-2 fixed z-20 outline-none w-full max-w-6xl flex"
          style={{ '--initial-transform': 'calc(100% + 8px)' } as React.CSSProperties}
        >
          <div className="bg-white h-full w-full grow p-5 flex flex-col rounded-[16px]">
            <div className="w-full">
              <Drawer.Title className="font-medium mb-4 text-gray-900">
                Nested Drawers.
              </Drawer.Title>
              <p className="text-gray-600 mb-2">
                Nesting drawers creates a{' '}
                <a href="https://sonner.emilkowal.ski/" target="_blank" className="underline">
                  Sonner-like
                </a>{' '}
                stacking effect .
              </p>
              <p className="text-gray-600 mb-2">
                You can nest as many drawers as you want. All you need to do is add a
                `Drawer.NestedRoot` component instead of `Drawer.Root`.
              </p>
              <Drawer.NestedRoot direction="right">
                <Drawer.Trigger className="rounded-md mt-4 w-full bg-gray-900 px-3.5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-gray-800 focus-visible:outline focus-visible:outline-offset-2 focus-visible:outline-gray-600">
                  Open Second Drawer
                </Drawer.Trigger>
                <Drawer.Portal>
                  <Drawer.Overlay className="fixed inset-0 bg-black/40 z-20" />
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
              </Drawer.NestedRoot>
            </div>
          </div>
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  )
}
