import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { Button } from '@kibamail/owly/button'
import { Text } from '@kibamail/owly/text'
import { MoreHorizIcon } from './Icons/more-horiz.svg'
import type { ElementType, MouseEvent } from 'react'

export type OptionsDropdownItem = {
  icon: ElementType
  name: string
  onClick: (event: MouseEvent) => void
  type?: 'destructive' | 'default'
}

export function OptionsDropdown({
  id,
  items,
}: {
  id: string
  items: OptionsDropdownItem[]
}) {
  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <Button
          variant="tertiary"
          onClick={(event) => event.stopPropagation()}
        >
          <MoreHorizIcon className="w-4 h-4 text-owly-content-tertiary" />
        </Button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Content
        sideOffset={8}
        align="end"
        id={id}
        className="border kb-border-tertiary rounded-xl p-1 shadow-[0px_16px_24px_-8px_var(--black-10)] kb-background-primary w-48 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 z-50"
      >
        {items.map((item, idx) => {
          const Icon = item.icon
          const destructive = item.type === 'destructive'
          const rowClasses = destructive
            ? 'p-2 flex items-center hover:bg-owly-background-negative/20 rounded-lg cursor-pointer w-full'
            : 'p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer w-full'
          const iconClasses = destructive
            ? 'mr-1.5 w-4 h-4 text-owly-content-negative'
            : 'mr-1.5 w-4 h-4 kb-content-tertiary'
          const textClasses = destructive
            ? 'text-owly-content-negative'
            : 'kb-content-secondary'

          return (
            <DropdownMenu.Item asChild key={`${id}-${idx}`}>
              <button
                type="button"
                className={rowClasses}
                onClick={(e) => {
                  e.stopPropagation()
                  item.onClick(e)
                }}
              >
                <Icon className={iconClasses} />
                <Text className={textClasses}>{item.name}</Text>
              </button>
            </DropdownMenu.Item>
          )
        })}
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  )
}

