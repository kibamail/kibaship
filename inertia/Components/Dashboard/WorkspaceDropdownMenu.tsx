import { CheckIcon } from '~/Components/Icons/check.svg'
import { NavArrowDownIcon } from '~/Components/Icons/nav-arrow-down.svg'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { SettingsIcon } from '~/Components/Icons/settings.svg'
import { UserPlusIcon } from '~/Components/Icons/user-plus.svg'
import { UserIcon } from '~/Components/Icons/user.svg'
import { Text } from '@kibamail/owly/text'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { SignoutForm } from './SignoutForm'
import { PageProps } from '~/types'
import { Link, usePage } from '@inertiajs/react'
import { ItemAvatar } from '../ItemAvatar'

interface WorkspacesDropdownMenuProps {
  rootId: string
  onCreateWorkspaceClick: () => void
}

export function WorkspacesDropdownMenu({
  rootId,
  onCreateWorkspaceClick,
}: WorkspacesDropdownMenuProps) {
  const pageProps = usePage<PageProps>()

  // Combine user's own workspaces and invited workspaces
  const allWorkspaces = pageProps?.props?.profile?.workspaces

  const activeWorkspace = allWorkspaces.find((workspace) =>
    pageProps?.url?.includes(`/${workspace.slug}`)
  )

  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          id={`${rootId}-dropdown-menu-trigger`}
          data-testid={`${rootId}-dropdown-menu-trigger`}
          className="grow flex items-center border transition ease-in-out border-(--border-tertiary) hover:bg-(--background-hover) focus:outline-none focus-within:border-(--border-focus) p-1 rounded-lg"
        >
          <span className="grow flex items-center">
            <ItemAvatar name={activeWorkspace?.name} size="md" />

            <Text className="kb-content-primary truncate capitalize">
              {activeWorkspace?.name || 'Select Workspace'}
            </Text>
          </span>

          <NavArrowDownIcon aria-hidden className="ml-1 w-4 h-4 kb-content-tertiary-inverse" />
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Content
        sideOffset={8}
        align="start"
        id={`${rootId}-dropdown-menu-content`}
        className="border workspaces-dropdown-menu kb-border-tertiary absolute rounded-xl p-1 shadow-[0px_16px_24px_-8px_var(--black-10)] kb-background-primary w-70 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 z-50"
      >
        <DropdownMenu.RadioGroup value={activeWorkspace?.id.toString()}>
          {pageProps?.props?.profile?.workspaces?.map((workspace) => (
            <DropdownMenu.RadioItem key={workspace.slug} value={workspace.slug} asChild>
              <Link
                href={`/w/${workspace.slug}`}
                data-testid={`${rootId}-switch-workspace-id-${workspace.slug}`}
                className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
              >
                <ItemAvatar name={workspace.name} size="sm" />
                <Text className="kb-content-secondary capitalize">{workspace.name}</Text>

                <DropdownMenu.ItemIndicator className="ml-auto">
                  <CheckIcon className="w-5 h-5 kb-content-secondary" />
                </DropdownMenu.ItemIndicator>
              </Link>
            </DropdownMenu.RadioItem>
          ))}
        </DropdownMenu.RadioGroup>

        <DropdownMenu.Item
          className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
          onSelect={onCreateWorkspaceClick}
        >
          <PlusIcon className="mr-1.5 w-5 h-5 kb-content-secondary" />
          <Text className="kb-content-secondary">New workspace</Text>
        </DropdownMenu.Item>

        <DropdownMenu.Separator className="my-1 h-px bg-(--black-5)" />

        <DropdownMenu.Item className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer">
          <UserPlusIcon className="mr-1.5 w-5 h-5 kb-content-tertiary" />
          <Text className="kb-content-secondary">Workspace members</Text>
        </DropdownMenu.Item>

        <DropdownMenu.Separator className="my-1 h-px bg-(--black-5)" />

        <DropdownMenu.Item className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer">
          <SettingsIcon className="mr-1.5 w-5 h-5 kb-content-tertiary" />
          <Text className="kb-content-secondary">Team settings</Text>
        </DropdownMenu.Item>

        <DropdownMenu.Item
          asChild
          className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
        >
          <a href={'/'}>
            <UserIcon className="mr-1.5 w-5 h-5 kb-content-tertiary" />
            <Text className="kb-content-secondary">Account settings</Text>
          </a>
        </DropdownMenu.Item>

        <DropdownMenu.Separator className="my-1 h-px bg-(--black-5)" />

        <SignoutForm />
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  )
}
