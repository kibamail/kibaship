import { Text } from '@kibamail/owly/text'
import { TeamAvatar } from '../Dashboard/WorkspaceDropdownMenu'
import { Cluster } from '~/types'
import Spinner from '../Icons/Spinner'
import classNames from 'classnames'
import { useState } from 'react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { MoreHorizIcon } from '../Icons/more-horiz.svg'
import { SettingsIcon } from '../Icons/settings.svg'
import { TrashIcon } from '../Icons/trash.svg'
import { Button } from '@kibamail/owly/button'
import { router } from '@inertiajs/react'
import * as Dialog from '@kibamail/owly/dialog'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { getStatusIcon } from './ClusterProvisioningStep'

interface ProjectCardProps {
  cluster: Cluster
  onClusterSelected?: (cluster: Cluster) => void
  onClusterDeletion?: (cluster: Cluster) => void
}

export function ClusterCard({ cluster, onClusterSelected, onClusterDeletion }: ProjectCardProps) {
  const statusClassNames = (() => {
    if (cluster.status === 'provisioning') {
      return 'text-owly-content-notice'
    }

    return ''
  })()

  function onCardClick(event: React.MouseEvent) {
    const dropdownMenuContent = document.querySelector('#cluster-card-dropdown-menu-content')

    if (dropdownMenuContent?.contains(event.target as Node)) {
      return
    }

    onClusterSelected?.(cluster)
  }

  return (
    <>
      <div
        onClick={onCardClick}
        className="w-full bg-white rounded-[11px] min-h-[150px] border border-owly-border-tertiary relative group cursor-pointer"
      >
        <div className="absolute w-[calc(100%+10px)] h-[calc(100%+10px)] rounded-[14px] transform translate-x-[-5px] translate-y-[-5px] bg-transparent transition-all ease-in-out border border-transparent group-hover:border-owly-border-tertiary"></div>
        <div className="p-4 z-1 flex flex-col justify-between h-full relative cursor-pointer rounded-[11px]">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <TeamAvatar
                name={cluster?.subdomainIdentifier}
                className="!mr-0 !rounded-sm w-10 h-10 !text-lg font-semibold"
              />
              <Text className="text-owly-content-secondary font-semibold">
                {cluster?.subdomainIdentifier}
              </Text>
            </div>

            <div className="flex items-center gap-1">
              {getStatusIcon(cluster.provisioningStatus)}

              <Text className={classNames('capitalize !text-sm !font-medium', statusClassNames)}>
                {cluster.provisioningStatus}
              </Text>

              <DropdownMenu.Root modal={false}>
                <DropdownMenu.Trigger asChild>
                  <Button variant="tertiary" onClick={(event) => event.stopPropagation()}>
                    <MoreHorizIcon className="w-4 h-4 text-owly-content-tertiary" />
                  </Button>
                </DropdownMenu.Trigger>

                <DropdownMenu.Content
                  sideOffset={8}
                  align="end"
                  id="cluster-card-dropdown-menu-content"
                  className="border kb-border-tertiary rounded-xl p-1 shadow-[0px_16px_24px_-8px_var(--black-10)] kb-background-primary w-48 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 z-50"
                >
                  <DropdownMenu.Item className="p-2 flex items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer">
                    <SettingsIcon className="mr-1.5 w-4 h-4 kb-content-tertiary" />
                    <Text className="kb-content-secondary">Settings</Text>
                  </DropdownMenu.Item>

                  <DropdownMenu.Separator className="my-1 h-px bg-(--black-5)" />

                  <DropdownMenu.Item
                    asChild
                    onClick={() => onClusterDeletion?.(cluster)}
                    className="p-2 flex items-center hover:bg-owly-background-negative/20 rounded-lg cursor-pointer w-full"
                  >
                    <button>
                      <TrashIcon className="mr-1.5 w-4 h-4 text-owly-content-negative" />
                      <Text className="text-owly-content-negative">Destroy</Text>
                    </button>
                  </DropdownMenu.Item>
                </DropdownMenu.Content>
              </DropdownMenu.Root>
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

              <Text className="text-owly-content-tertiary">{cluster?.location}</Text>
            </div>

            <Text className="text-owly-content-tertiary text-sm">
              3 control planes, 2 worker nodes
            </Text>
          </div>
        </div>
      </div>
    </>
  )
}
