import { Badge, Button } from '@kibamail/owly'
import { Text } from '@kibamail/owly/text'
import classNames from 'classnames'
import { useState } from 'react'
import { ArrowUpCircleSolidIcon } from '~/Components/Icons/arrow-up-circle-solid.svg'
import { CheckIcon } from '~/Components/Icons/check.svg'
import { CircleIcon } from '~/Components/Icons/circle.svg'
import { ClockIcon } from '~/Components/Icons/clock.svg'
import { GitBranchIcon } from '~/Components/Icons/git-branch.svg'
import { GitCommitIcon } from '~/Components/Icons/git-commit.svg'
import { MoreHorizIcon } from '~/Components/Icons/more-horiz.svg'
import { NavArrowDownIcon } from '~/Components/Icons/nav-arrow-down.svg'
import Spinner from '~/Components/Icons/Spinner'

interface DeploymentCardProps {
  onSelected?: () => void
}

export function DeploymentCard({ onSelected }: DeploymentCardProps) {
  const [stateStatusExpanded, setStateStatusExpanded] = useState(false)

  return (
    <div className="w-full border border-owly-border-info rounded-md p-1 flex flex-col gap-1">
      <div
        role="button"
        onClick={onSelected}
        className="w-full cursor-pointer p-3 rounded-t-sm bg-owly-background-info/5 grid items-center grid-cols-2 lg:grid-cols-[repeat(4,1fr)_64px] gap-4 lg:gap-6"
      >
        <div className="flex flex-col gap-0.5">
          <Text className="text-owly-content-secondary !font-semibold text-left">85c345p3d</Text>

          <div className="flex items-center gap-2">
            <Text className="!text-sm text-owly-content-tertiary lowercase">Production</Text>
            <Badge size="sm" variant="info" className="!font-medium">
              <ArrowUpCircleSolidIcon />
              Current
            </Badge>
          </div>
        </div>

        <div className="flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <div className="size-2.5 rounded-full bg-owly-background-notice"></div>

            <Text>Building</Text>
          </div>

          <div className="flex gap-2">
            <ClockIcon className="w-4 h-4 mt-0.5 text-owly-content-tertiary" />

            <Text className="text-sm text-owly-content-tertiary">1m 33s</Text>
          </div>
        </div>

        <div className="flex flex-col gap-0.5">
          <div className="flex gap-1 items-center">
            <GitBranchIcon className="size-4 text-owly-content-tertiary" />

            <Text className="text-owly-content-secondary">main</Text>
          </div>

          <div className="flex gap-1 items-center">
            <GitCommitIcon className="size-5 text-owly-content-tertiary" />

            <Text className="text-owly-content-secondary">85c345p3d</Text>
          </div>
        </div>

        <Text className="text-owly-content-tertiary flex items-center">
          19 mins ago <span className="hidden lg:inline">by olumide</span>
          <img
            className="size-[22px] ml-2 rounded-full"
            src="https://github.com/bahdcoder.png"
            alt="olumide"
          />
        </Text>

        <div className="flex justify-end max-w-[64px]">
          <Button variant="tertiary">
            <MoreHorizIcon />
          </Button>
        </div>
      </div>

      <button
        onClick={() => setStateStatusExpanded((current) => !current)}
        className="w-full p-1.5 cursor-pointer rounded-b-sm bg-owly-background-info/5 flex justify-between items-center"
      >
        <Text className="text-owly-content-info flex items-center gap-1">
          <CheckIcon className="size-5" />
          Deployment in progress
        </Text>

        <div className="pr-4">
          <NavArrowDownIcon
            className={classNames('size-5 transition-transform ease-in-out', {
              'transform rotate-180': stateStatusExpanded,
            })}
          />
        </div>
      </button>

      <div
        className={classNames(
          'w-full flex flex-col overflow-hidden transition-all duration-300 ease-in-out',
          {
            'max-h-0 opacity-0 -mt-1': !stateStatusExpanded,
            'max-h-[200px] opacity-100 p-1.5': stateStatusExpanded,
          }
        )}
      >
        <div className="flex h-10 items-center justify-between">
          <Text className="text-owly-content-secondary gap-2 flex items-center">
            <CheckIcon className="size-5" />
            Initialised deployment
          </Text>
        </div>

        <div className="flex h-10 items-center justify-between">
          <Text className="text-owly-content-secondary gap-2 flex items-center">
            <Spinner className="size-4" />
            Building docker image
          </Text>

          <div className="pr-4">
            <Text className="text-owly-content-tertiary">1m 33s</Text>
          </div>
        </div>

        <div className="flex h-10 items-center justify-between">
          <Text className="text-owly-content-secondary gap-2 flex items-center">
            <CircleIcon className="size-4 text-owly-content-tertiary" />
            Upload artifacts
          </Text>

          <div className="pr-4">
            <Text className="text-owly-content-tertiary">not started</Text>
          </div>
        </div>
      </div>
    </div>
  )
}
