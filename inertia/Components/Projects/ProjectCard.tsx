import { Text } from '@kibamail/owly/text'
import { TeamAvatar } from '../Dashboard/WorkspaceDropdownMenu'
import { PageProps, Project } from '~/types'
import { Link, usePage } from '@inertiajs/react'

interface ProjectCardProps {
  project: Project
}

export function ProjectCard({ project }: ProjectCardProps) {
  console.log({ project })
  const { props } = usePage<PageProps>()
  return (
    <Link
      href={`/w/${props.workspace.slug}/p/${project.id}`}
      className="w-full bg-white rounded-[11px] min-h-[150px] border border-owly-border-tertiary relative group cursor-pointer"
    >
      <div className="absolute w-[calc(100%+10px)] h-[calc(100%+10px)] rounded-[14px] transform translate-x-[-5px] translate-y-[-5px] bg-transparent transition-all ease-in-out border border-transparent group-hover:border-owly-border-tertiary"></div>
      <div className="p-4 z-1 flex flex-col justify-between h-full relative cursor-pointer rounded-[11px]">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <TeamAvatar
              name="kibaauth"
              className="!mr-0 !rounded-sm w-10 h-10 !text-lg font-semibold"
            />
            <Text className="text-owly-content-secondary font-semibold">{project?.name}</Text>
          </div>
        </div>

        <div className="gap-y-1.5 flex justify-between items-center">
          <div className="flex items-center gap-2">
            {/* source of colour svgs: https://nucleoapp.com/svg-flag-icons */}
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-6 h-6"
              width="32"
              height="32"
              viewBox="0 0 32 32"
            >
              <path fill="#cc2b1d" d="M1 11H31V21H1z"></path>
              <path d="M5,4H27c2.208,0,4,1.792,4,4v4H1v-4c0-2.208,1.792-4,4-4Z"></path>
              <path
                d="M5,20H27c2.208,0,4,1.792,4,4v4H1v-4c0-2.208,1.792-4,4-4Z"
                transform="rotate(180 16 24)"
                fill="#f8d147"
              ></path>
              <path
                d="M27,4H5c-2.209,0-4,1.791-4,4V24c0,2.209,1.791,4,4,4H27c2.209,0,4-1.791,4-4V8c0-2.209-1.791-4-4-4Zm3,20c0,1.654-1.346,3-3,3H5c-1.654,0-3-1.346-3-3V8c0-1.654,1.346-3,3-3H27c1.654,0,3,1.346,3,3V24Z"
                opacity=".15"
              ></path>
              <path
                d="M27,5H5c-1.657,0-3,1.343-3,3v1c0-1.657,1.343-3,3-3H27c1.657,0,3,1.343,3,3v-1c0-1.657-1.343-3-3-3Z"
                fill="#fff"
                opacity=".2"
              ></path>
            </svg>

            <Text className="text-owly-content-tertiary">{project?.cluster?.location}</Text>
          </div>

          <Text className="text-owly-content-tertiary text-sm">31 apps</Text>
        </div>
      </div>
    </Link>
  )
}
