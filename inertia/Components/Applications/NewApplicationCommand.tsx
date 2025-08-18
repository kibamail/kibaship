import { Command } from 'cmdk'

import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
dayjs.extend(relativeTime)

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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useForm, usePage } from '@inertiajs/react'
import { PageProps, SourceCodeProvider, SourceCodeRepository } from '~/types'
import { axios } from '~/app/axios'
import Spinner from '../Icons/Spinner'
import { AxiosError } from 'axios'

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
  REPOSITORY = 'repository',
  COMPLETE = 'complete',
}

type Command = {
  id: CommandType
  key?: string
  label: string | JSX.Element
  href?: string
  onClick?: () => void
  icon: JSX.Element
  commands?: Command[]
}

export function NewApplicationCommand({ open, onOpenChange }: NewApplicationCommandProps) {
  const [pages, setPages] = useState<CommandType[]>([])
  const [search, setSearch] = useState('')
  const [selectedProvider, setSelectedProvider] = useState<SourceCodeProvider | null>(null)
  const [selectedRepository, setSelectedRepository] = useState<SourceCodeRepository | null>(null)

  const form = useForm({
    type: 'git',
    gitConfiguration: {
      sourceCodeRepositoryId: '',
    },
  })

  const page = pages[pages.length - 1]

  const providersQuery = useQuery<SourceCodeProvider[]>({
    queryKey: ['connections/source-code-providers'],
    async queryFn() {
      const response = await axios.get('/connections/source-code-providers')
      return response.data
    },
    enabled: page === CommandType.GIT,
    refetchOnWindowFocus: true,
  })

  const repositoriesQuery = useQuery<SourceCodeRepository[]>({
    queryKey: ['connections/source-code-providers', selectedProvider?.id],
    async queryFn() {
      const response = await axios.get(`/connections/source-code-providers/${selectedProvider?.id}`)
      return response.data
    },
    enabled: page === CommandType.REPOSITORY && Boolean(selectedProvider),
  })

  const createApplicationMutation = useMutation<
    unknown,
    AxiosError,
    {
      sourceCodeRepositoryId: string
    }
  >({
    async mutationFn({ sourceCodeRepositoryId }) {
      const response = await axios.post('/w/applications', {
        type: 'git',
        gitConfiguration: {
          sourceCodeRepositoryId,
        },
      })
      return response.data
    },
  })

  const commands: Command[] = [
    {
      id: CommandType.GIT,
      label: 'Deploy git repository',
      icon: <GitIcon className="size-5" />,
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

  const hasAtLeastOneGithubConnection = providersQuery.data?.some(
    (provider) => provider.provider === 'github'
  )

  const gitCommands: Command[] = [
    ...(providersQuery.data?.map((provider) => ({
      id: CommandType.REPOSITORY,
      label: (
        <span>
          <span className="capitalize">{provider.name}</span> repositories
        </span>
      ),
      icon: <img src={provider?.avatar} alt={provider?.name} className="size-6 rounded-md" />,
      onClick() {
        setSelectedProvider(provider)
        setPages((current) => [...current, CommandType.REPOSITORY])
      },
    })) || []),
    {
      id: CommandType.CONNECT_GITHUB,
      label: hasAtLeastOneGithubConnection
        ? 'Configure github app installation'
        : 'Connect github app',
      icon: <GitHubIcon className="size-5" />,
      href: '/connections/github/redirect',
    },
    {
      id: CommandType.CONNECT_GITLAB,
      label: 'Connect gitlab app',
      icon: <GitLabIcon className="size-5" />,
      href: '/connections/gitlab/redirect',
    },
    {
      id: CommandType.CONNECT_BITBUCKET,
      label: 'Connect bitbucket app',
      icon: <BitbucketIcon className="size-5" />,
      href: '/connections/bitbucket/redirect',
    },
  ]

  const repositories = repositoriesQuery?.data?.map((repository) => ({
    id: CommandType.COMPLETE,
    label: (
      <div className="flex items-center gap-0.5">
        <span>{selectedProvider?.name}</span>
        <div className="h-3.5 mx-0.5 w-px bg-(--border-primary) transform rotate-20"></div>
        <span>{repository.repository}</span>

        <GitHubIcon className="size-3 ml-2 opacity-45" />

        <span className="text-sm kb-content-tertiary ml-2">
          {repository.lastUpdatedAt ? dayjs(repository.lastUpdatedAt).fromNow() : 'Never'}
        </span>
      </div>
    ),
    key: repository?.id,
    icon: (
      <img
        className="size-6 rounded-md"
        src={selectedProvider?.avatar}
        alt={selectedProvider?.name}
      />
    ),
    onClick() {
      setSelectedRepository(repository)
      form.setData('gitConfiguration.sourceCodeRepositoryId', repository.id)

      setTimeout(() => {
        form.post('/w/applications')
      }, 0)
    },
  }))

  const commandsWithSubItems = [CommandType.GIT, CommandType.DOCKER, CommandType.REPOSITORY]

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
          <div className="w-full flex items-center h-15 kb-background-secondary rounded-t-2xl px-4">
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
          <Command.List className="p-2 max-h-80 overflow-y-auto">
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
                  onClick={command?.onClick}
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
                {providersQuery.isLoading ? (
                  <div className="w-full flex items-center justify-center py-4">
                    <Spinner className="w-5 h-5" />
                  </div>
                ) : null}
                {gitCommands.map((command) => {
                  const Container = command?.href ? 'a' : 'button'

                  return (
                    <Command.Item key={command.id} value={command.id} className={itemClassNames}>
                      <Container
                        href={command?.href}
                        onClick={command?.onClick}
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

                        {command?.href ? (
                          <OpenNewWindowIcon className="size-4 kb-content-tertiary" />
                        ) : null}
                        {commandsWithSubItems.includes(command.id) ? (
                          <NavArrowRightIcon className="size-5 kb-content-tertiary" />
                        ) : null}
                      </Container>
                    </Command.Item>
                  )
                })}
              </>
            )}

            {page === CommandType.REPOSITORY && (
              <>
                {repositoriesQuery.isLoading ? (
                  <div className="w-full flex items-center justify-center py-4">
                    <Spinner className="w-5 h-5" />
                  </div>
                ) : null}

                {repositories?.map((repository) => {
                  return (
                    <Command.Item key={repository.key} className={itemClassNames}>
                      <button
                        className="flex items-center w-full cursor-pointer h-full justify-between"
                        onClick={repository.onClick}
                        disabled={form?.processing}
                      >
                        <div className="flex items-center gap-3">
                          {repository.icon}
                          <span className="kb-content-secondary font-sans text-sm">
                            {repository.label}
                          </span>
                        </div>

                        {form?.processing &&
                        form?.data?.gitConfiguration?.sourceCodeRepositoryId === repository?.key ? (
                          <Spinner className="size-5 kb-content-tertiary" />
                        ) : (
                          <NavArrowRightIcon className="size-4 kb-content-tertiary" />
                        )}
                      </button>
                    </Command.Item>
                  )
                })}
              </>
            )}
          </Command.List>
        </div>
      </Command.Dialog>
    </>
  )
}
