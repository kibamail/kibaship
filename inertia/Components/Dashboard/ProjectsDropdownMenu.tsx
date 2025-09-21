import { CheckIcon } from '~/Components/Icons/check.svg'
import { NavArrowDownIcon } from '~/Components/Icons/nav-arrow-down.svg'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { Text } from '@kibamail/owly/text'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import cn from 'classnames'
import { Link, usePage } from '@inertiajs/react'
import { PageProps } from '~/types'

interface ProjectsDropdownMenuProps {
  rootId: string
  onCreateProjectClick?: () => void
}

export function ProjectsDropdownMenu({ rootId, onCreateProjectClick }: ProjectsDropdownMenuProps) {
  const {
    props: { activeProject, projects, workspace },
  } = usePage<PageProps>()

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
            <ProjectAvatar name={activeProject?.name} size="md" />

            <Text className="kb-content-primary truncate">
              {activeProject?.name || 'Select a project'}
            </Text>
          </span>

          <NavArrowDownIcon aria-hidden className="ml-1 w-4 h-4 kb-content-tertiary-inverse" />
        </button>
      </DropdownMenu.Trigger>

      <DropdownMenu.Content
        sideOffset={8}
        align="start"
        id={`${rootId}-dropdown-menu-content`}
        className="border projects-dropdown-menu kb-border-tertiary absolute rounded-xl p-1 shadow-[0px_16px_24px_-8px_var(--black-10)] kb-background-primary w-70 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 z-50"
      >
        <DropdownMenu.RadioGroup value={activeProject?.id.toString()}>
          {projects.map((project) => (
            <DropdownMenu.RadioItem key={project.id} value={project.id.toString()} asChild>
              <Link
                data-testid={`${rootId}-switch-project-id-${project.id}`}
                href={`/w/${workspace?.slug}/p/${project.id}`}
                className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
              >
                <ProjectAvatar name={project.name} size="sm" />
                <Text className="kb-content-secondary capitalize">{project.name}</Text>

                <DropdownMenu.ItemIndicator className="ml-auto">
                  <CheckIcon className="w-5 h-5 kb-content-secondary" />
                </DropdownMenu.ItemIndicator>
              </Link>
            </DropdownMenu.RadioItem>
          ))}
        </DropdownMenu.RadioGroup>

        <DropdownMenu.Item
          className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
          onSelect={() => onCreateProjectClick?.()}
        >
          <PlusIcon className="mr-1.5 w-5 h-5 kb-content-tertiary" />
          <Text>Add new project</Text>
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  )
}

interface ProjectAvatarProps {
  size: 'sm' | 'md'
  name?: string
}

function ProjectAvatar({ size, name }: ProjectAvatarProps) {
  return (
    <span
      className={cn(
        'mr-1.5 text-sm shadow-[0px_0px_0px_1px_rgba(0,0,0,0.10)_inset] kb-background-info rounded-lg flex items-center justify-center kb-content-primary-inverse uppercase',
        {
          'w-5 h-5': size === 'sm',
          'w-6 h-6': size === 'md',
        },
        getProjectAvatarBackgroundColor(name?.[0] ?? '')
      )}
    >
      {name?.[0]}
    </span>
  )
}

function getProjectAvatarBackgroundColor(firstCharacter: string) {
  const colors = [
    'kb-background-info',
    'kb-background-positive',
    'kb-background-negative',
    'kb-background-warning',
    'kb-background-highlight',
  ]

  const asciiValue = firstCharacter.charCodeAt(0)
  const index = (asciiValue - 97) % colors.length

  return colors[index] ?? colors?.[0]
}
