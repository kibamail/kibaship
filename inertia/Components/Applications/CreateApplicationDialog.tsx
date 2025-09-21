import { useState } from 'react'
import { useForm, usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import * as Select from '@kibamail/owly/select-field'
import { Text } from '@kibamail/owly/text'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import type { FormEventHandler } from 'react'
import { PageProps } from '~/types'

// Application type configuration components (empty for now)
import { GitRepositoryConfig } from './configs/GitRepositoryConfig'
import { MysqlConfig } from './configs/MysqlConfig'
import { PostgresqlConfig } from './configs/PostgresqlConfig'
import { DockerImageConfig } from './configs/DockerImageConfig'
import { MysqlClusterConfig } from './configs/MysqlClusterConfig'
import { PostgresqlClusterConfig } from './configs/PostgresqlClusterConfig'
import { ValkeyConfig } from './configs/ValkeyConfig'
import { ValkeyClusterConfig } from './configs/ValkeyClusterConfig'

// Icons for application types
import { GitBranchIcon } from '~/Components/Icons/git-branch.svg'
import { MySQLIcon } from '~/Components/Icons/mysql.svg'
import { PostgresIcon } from '~/Components/Icons/postgres.svg'
import { DockerIcon } from '~/Components/Icons/docker.svg'
import { ServerIcon } from '~/Components/Icons/server.svg'
import { ClusterIcon } from '~/Components/Icons/cluster.svg'

export type ApplicationType =
  | 'git_repository'
  | 'mysql'
  | 'postgresql'
  | 'docker_image'
  | 'mysql_cluster'
  | 'postgresql_cluster'
  | 'valkey'
  | 'valkey_cluster'

interface CreateApplicationDialogProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
}

interface ApplicationTypeOption {
  value: ApplicationType
  label: string
  icon: React.ComponentType<{ className?: string }>
  description: string
}

const applicationTypeOptions: ApplicationTypeOption[] = [
  {
    value: 'git_repository',
    label: 'Git Repository',
    icon: GitBranchIcon,
    description: 'Deploy applications from Git repositories',
  },
  {
    value: 'mysql',
    label: 'MySQL',
    icon: MySQLIcon,
    description: 'Single MySQL database instance',
  },
  {
    value: 'postgresql',
    label: 'PostgreSQL',
    icon: PostgresIcon,
    description: 'Single PostgreSQL database instance',
  },
  {
    value: 'docker_image',
    label: 'Docker Image',
    icon: DockerIcon,
    description: 'Deploy from Docker container images',
  },
  {
    value: 'mysql_cluster',
    label: 'MySQL Cluster',
    icon: ClusterIcon,
    description: 'High-availability MySQL cluster',
  },
  {
    value: 'postgresql_cluster',
    label: 'PostgreSQL Cluster',
    icon: ClusterIcon,
    description: 'High-availability PostgreSQL cluster',
  },
  {
    value: 'valkey',
    label: 'Valkey',
    icon: ServerIcon,
    description: 'Single Valkey cache instance',
  },
  {
    value: 'valkey_cluster',
    label: 'Valkey Cluster',
    icon: ClusterIcon,
    description: 'High-availability Valkey cluster',
  },
]

export function CreateApplicationDialog({ isOpen, onOpenChange }: CreateApplicationDialogProps) {
  const {
    props: { workspace },
  } = usePage<PageProps>()
  const [selectedApplicationType, setSelectedApplicationType] = useState<ApplicationType | ''>('')

  const { data, setData, post, processing, errors, reset } = useForm({
    name: '',
    type: '' as ApplicationType | '',
    clusterId: '',
  })

  const submit: FormEventHandler = (event) => {
    event.preventDefault()

    // TODO: Replace with actual endpoint when ready
    console.log('Form submitted:', {
      name: data.name,
      type: data.type,
      clusterId: data.clusterId,
    })

    // post(`/w/${workspace?.slug}/applications`, {
    //   onSuccess() {
    //     reset()
    //     onOpenChange(false)
    //   },
    // })
  }

  const handleApplicationTypeChange = (value: string) => {
    const appType = value as ApplicationType
    setSelectedApplicationType(appType)
    setData('type', appType)
  }

  const renderApplicationTypeConfig = () => {
    switch (selectedApplicationType) {
      case 'git_repository':
        return <GitRepositoryConfig />
      case 'mysql':
        return <MysqlConfig />
      case 'postgresql':
        return <PostgresqlConfig />
      case 'docker_image':
        return <DockerImageConfig />
      case 'mysql_cluster':
        return <MysqlClusterConfig />
      case 'postgresql_cluster':
        return <PostgresqlClusterConfig />
      case 'valkey':
        return <ValkeyConfig />
      case 'valkey_cluster':
        return <ValkeyClusterConfig />
      default:
        return null
    }
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>Create Application</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Create a new application by selecting the type and configuring its settings.
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="kb-content-secondary text-sm leading-relaxed">
            Deploy and manage applications on your clusters. Choose from various application types
            including databases, caches, and custom applications from Git repositories or Docker
            images.
          </Text>
        </div>

        <form onSubmit={submit}>
          <div className="px-5 pb-5 space-y-4">
            {/* Application Type Selection */}
            <Select.Root name="type" value={data.type} onValueChange={handleApplicationTypeChange}>
              <Select.Label>Application Type</Select.Label>
              <Select.Trigger placeholder="Select application type" />
              <Select.Content className="z-50 relative w-full max-h-64 overflow-y-auto">
                {applicationTypeOptions.map((option) => {
                  const IconComponent = option.icon

                  return (
                    <Select.Item
                      key={option.value}
                      value={option.value}
                      className="w-full flex items-center gap-2"
                    >
                      <IconComponent className="w-4 h-4 text-owly-content-secondary" />
                      <span>{option.label}</span>
                    </Select.Item>
                  )
                })}
              </Select.Content>
              {errors.type && <Select.Error>{errors.type}</Select.Error>}
            </Select.Root>

            {/* Cluster Selection */}
            <Select.Root
              name="clusterId"
              value={data.clusterId}
              onValueChange={(value) => setData('clusterId', value)}
            >
              <Select.Label>Target Cluster</Select.Label>
              <Select.Trigger placeholder="Select cluster" />
              <Select.Content className="z-50 relative w-full">
                {/* TODO: Populate with actual clusters when data is available */}
                <Select.Item value="clu" disabled>
                  <span className="text-owly-content-tertiary">No clusters available</span>
                </Select.Item>
              </Select.Content>
              {errors.clusterId && <Select.Error>{errors.clusterId}</Select.Error>}
            </Select.Root>

            {/* Application Type Specific Configuration */}
            {selectedApplicationType && (
              <div className="pt-4 border-t border-owly-border-tertiary">
                <Text className="text-sm font-medium mb-3 text-owly-content-secondary">
                  Configuration
                </Text>
                {renderApplicationTypeConfig()}
              </div>
            )}
          </div>

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild disabled={processing}>
              <Button variant="secondary">Cancel</Button>
            </Dialog.Close>

            <Button type="submit" loading={processing} disabled={!data.type || !data.clusterId}>
              Create Application
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}
