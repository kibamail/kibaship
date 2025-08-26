'use client'

import { Drawer } from 'vaul'
import { Cluster } from '~/types'

interface ClusterProvisioningSheetProps {
  cluster: Cluster | null
  isOpen: boolean
  onOpenChange: (open: boolean) => void
}

export function ClusterProvisioningSheet({ isOpen, onOpenChange }: ClusterProvisioningSheetProps) {
  return (
    <Drawer.Root direction="right" open={isOpen} onOpenChange={onOpenChange}>
      <Drawer.Portal>
        <Drawer.Overlay className="fixed inset-0 bg-black/40" />
        <Drawer.Content
          className="right-2 top-2 bottom-2 fixed z-20 outline-none w-full max-w-6xl flex"
          style={{ '--initial-transform': 'calc(100% + 8px)' } as React.CSSProperties}
        >
          <div className="bg-white h-full w-full grow p-5 flex flex-col rounded-[16px]">
            <h1>Content of provisioning here</h1>
          </div>
        </Drawer.Content>
      </Drawer.Portal>
    </Drawer.Root>
  )
}
