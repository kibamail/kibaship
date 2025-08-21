import { FooterMenuItems } from '~/Components/Dashboard/FooterMenuItems'
import { KibaIcon } from '~/Components/Icons/kiba.svg'
import { useState } from 'react'
import { SlashesIcon } from '../Icons/slashes.svg'
import { CreateProjectFlow } from './CreateProjectFlow'
import { CreateWorkspaceFlow } from './CreateWorkspaceFlow'
import { ProjectsDropdownMenu } from './ProjectsDropdownMenu'
import { SearchBoxTrigger } from './SearchBoxTrigger'
import { WorkspacesDropdownMenu } from './WorkspaceDropdownMenu'
import { Link, usePage } from '@inertiajs/react'
import { PageProps } from '~/types'

export function Topbar() {
  const [isCreateWorkspaceOpen, setIsCreateWorkspaceOpen] = useState(false)
  const [isCreateProjectOpen, setIsCreateProjectOpen] = useState(false)

  const { props } = usePage<PageProps>()

  const handleCreateWorkspaceClick = () => {
    setIsCreateWorkspaceOpen(true)
  }

  const handleCreateProjectClick = () => {
    setIsCreateProjectOpen(true)
  }

  return (
    <>
      <nav className="w-full lg:h-12 box-border p-2 flex items-center relative">
        <div className="flex items-center gap-2">
          <Link
            href={`/w/${props?.workspace?.slug}`}
            type="button"
            aria-label="Expand sidebar"
            className="kb-reset"
          >
            <KibaIcon className="w-8 h-8" />
          </Link>

          <SlashesIcon width={24} height={24} viewBox="0 0 24 24" />

          <WorkspacesDropdownMenu
            rootId="topbar-workspaces"
            onCreateWorkspaceClick={handleCreateWorkspaceClick}
          />

          {props?.projects?.length > 0 && (
            <>
              <SlashesIcon width={24} height={24} viewBox="0 0 24 24" />

              <ProjectsDropdownMenu
                rootId="topbar-projects"
                onCreateProjectClick={handleCreateProjectClick}
              />
            </>
          )}
        </div>

        <div className="max-w-md hidden lg:flex w-full absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 justify-center items-center">
          <SearchBoxTrigger />
        </div>

        <div className="ml-auto hidden lg:flex items-center gap-x-4">
          <FooterMenuItems />
        </div>
      </nav>

      <CreateWorkspaceFlow isOpen={isCreateWorkspaceOpen} onOpenChange={setIsCreateWorkspaceOpen} />
      <CreateProjectFlow isOpen={isCreateProjectOpen} onOpenChange={setIsCreateProjectOpen} />
    </>
  )
}
