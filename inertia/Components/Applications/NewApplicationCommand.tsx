import { Command } from 'cmdk'

import { MySQLIcon } from '~/Components/Icons/mysql.svg'
import { DockerIcon } from '~/Components/Icons/docker.svg'
import { GitIcon } from '~/Components/Icons/git.svg'
import { PostgresIcon } from '~/Components/Icons/postgres.svg'
import { useEffect, useState } from 'react'
import { GitHubIcon } from '../Icons/github.svg'
import { GitLabIcon } from '../Icons/gitlab.svg'
import { BitbucketIcon } from '../Icons/bitbucket.svg'
import { NavArrowRightIcon } from '../Icons/nav-arrow-right.svg'
import { JSX } from 'react/jsx-runtime'
import classNames from 'classnames'
import { OpenNewWindowIcon } from '../Icons/open-new-window.svg'
import { Button } from '@kibamail/owly'

interface NewApplicationCommandProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

enum CommandType {
  MYSQL = 'mysql',
  DOCKER = 'docker_image',
  POSTGRES = 'postgres',
  GIT = 'git',
  CONNECT_GITHUB = 'connect_github',
  CONNECT_GITLAB = 'connect_gitlab',
  CONNECT_BITBUCKET = 'connect_bitbucket',
}

type Command = {
  id: CommandType
  label: string
  href?: string
  icon: JSX.Element
  commands?: Command[]
}

export function NewApplicationCommand({ open, onOpenChange }: NewApplicationCommandProps) {
  const [pages, setPages] = useState<CommandType[]>([])
  const [search, setSearch] = useState('')

  const commands: Command[] = [
    {
      id: CommandType.GIT,
      label: 'Deploy git repository',
      icon: <GitIcon className="size-5" />,
      commands: [
        {
          id: CommandType.CONNECT_GITHUB,
          label: 'Connect github app',
          icon: <GitHubIcon className="size-5" />,
          href: 'https://github.com',
        },
        {
          id: CommandType.CONNECT_GITLAB,
          label: 'Connect gitlab app',
          icon: <GitLabIcon className="size-5" />,
          href: 'https://gitlab.com',
        },
        {
          id: CommandType.CONNECT_BITBUCKET,
          label: 'Connect bitbucket app',
          icon: <BitbucketIcon className="size-5" />,
          href: 'https://bitbucket.org',
        },
      ],
    },
    {
      id: CommandType.MYSQL,
      label: 'Deploy mysql',
      icon: <MySQLIcon className="size-5" />,
    },
    {
      id: CommandType.POSTGRES,
      label: 'Deploy postgresql',
      icon: <PostgresIcon className="size-5" />,
    },
    {
      id: CommandType.DOCKER,
      label: 'Deploy docker image',
      icon: <DockerIcon className="size-5" />,
    },
  ]

  const gitCommands = [
    {
      id: CommandType.CONNECT_GITHUB,
      label: 'Connect github app',
      icon: <GitHubIcon className="size-5" />,
      href: 'https://github.com',
    },
    {
      id: CommandType.CONNECT_GITLAB,
      label: 'Connect gitlab app',
      icon: <GitLabIcon className="size-5" />,
      href: 'https://gitlab.com',
    },
    {
      id: CommandType.CONNECT_BITBUCKET,
      label: 'Connect bitbucket app',
      icon: <BitbucketIcon className="size-5" />,
      href: 'https://bitbucket.org',
    },
  ]

  const page = pages[pages.length - 1]

  const commandsWithSubItems = [CommandType.GIT, CommandType.DOCKER]

  useEffect(() => {
    setSearch('')
  }, [pages])

  useEffect(() => {
    if (!open) {
      setPages([])
      setSearch('')
    }
  }, [open])

  const itemClassNames = classNames(
    'h-11 w-full flex items-center px-2 hover:bg-[var(--background-hover)] transition ease-linear cursor-pointer rounded-lg kb-content-secondary'
  )

  return (
    <>
      <Command.Dialog
        open={open}
        onOpenChange={onOpenChange}
        className="absolute top-0 w-full h-screen bg-[rgba(17,17,17,0.2)]"
        onKeyDown={(event) => {
          if (event.key === 'Escape' || (event.key === 'Backspace' && !search)) {
            event.preventDefault()
            setPages((pages) => pages.slice(0, -1))
          }
        }}
      >
        <div className="flex flex-col bg-white shadow-lg max-w-xl mx-auto mt-64 rounded-xl">
          <div className="w-full flex items-center h-15 kb-background-secondary rounded-t-2xl px-6">
            {page && (
              <Button
                variant="secondary"
                size="xs"
                className="flex-shrink-0"
                onClick={() => setPages((pages) => pages.slice(0, -1))}
              >
                <NavArrowRightIcon className="transform rotate-180" />{' '}
              </Button>
            )}
            <Command.Input
              className={classNames(
                ' w-full text-sm flex items-center font-sans placeholder:font-sans kb-content-secondary focus:outline-none',
                {
                  'pl-4': page,
                }
              )}
              placeholder="What would you like to deploy?"
              value={search}
              onValueChange={setSearch}
            />
          </div>
          <Command.List className="p-2">
            <Command.Empty className="w-full py-6 kb-content-tertiary text-sm text-center max-w-sm mx-auto">
              We probably don't deploy this way yet. Ping us with feedback if you think we should.
            </Command.Empty>

            {!page &&
              commands.map((command) => (
                <Command.Item
                  key={command.id}
                  value={command.id}
                  onSelect={(value) => setPages((current) => [...current, value as CommandType])}
                  className={itemClassNames}
                >
                  <div className="flex items-center w-full justify-between">
                    <div className="flex items-center gap-3">
                      {command.icon}
                      <span className="kb-content-secondary font-sans text-sm">
                        {command.label}
                      </span>
                    </div>

                    {commandsWithSubItems.includes(command.id) ? (
                      <NavArrowRightIcon className="size-5 kb-content-tertiary" />
                    ) : null}
                  </div>
                </Command.Item>
              ))}

            {page === CommandType.GIT && (
              <>
                {gitCommands.map((command) => (
                  <Command.Item key={command.id} value={command.id} className={itemClassNames}>
                    <a
                      href={command?.href}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center w-full justify-between"
                    >
                      <div className="flex items-center gap-3">
                        {command.icon}
                        <span className="kb-content-secondary font-sans text-sm">
                          {command.label}
                        </span>
                      </div>

                      <OpenNewWindowIcon className="size-4 kb-content-tertiary" />
                    </a>
                  </Command.Item>
                ))}
              </>
            )}
          </Command.List>
        </div>
      </Command.Dialog>
    </>
  )
}
