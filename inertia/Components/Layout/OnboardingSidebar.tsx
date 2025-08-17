import { Heading } from '@kibamail/owly/heading'
import { Progress } from '@kibamail/owly/progress'
import { Text } from '@kibamail/owly/text'
import cn from 'classnames'
import { CheckIcon } from '../Icons/check.svg'
import { CloudWaterdropIcon } from '../Icons/cloud-waterdrop.svg'
import { ClusterIcon } from '../Icons/cluster.svg'
import { CodeRepositoryIcon } from '../Icons/code-repository.svg'
import { ProjectLaunchIcon } from '../Icons/project-launch.svg'
import { WorkspaceIcon } from '../Icons/workspace.svg'

interface OnboardingStepProps {
  completed?: boolean
  title: string
  description: string
  Icon: React.ComponentType<{ className?: string }>
}

function OnboardingStep({ completed, title, description, Icon }: OnboardingStepProps) {
  return (
    <div className="flex items-start gap-4 p-3 bg-white border kb-border-tertiary rounded-lg">
      <div className="w-10 h-10 rounded-full shrink-0 border kb-border-tertiary flex justify-center items-center kb-background-hover">
        <Icon className="w-6 h-6 kb-content-tertiary" />
      </div>
      <div className="-mt-1">
        <Heading
          size="xs"
          className={cn({
            'line-through kb-content-tertiary': completed,
          })}
        >
          {title}
        </Heading>
        <Text size="sm" className="kb-content-tertiary">
          {description}
        </Text>
      </div>

      <div
        className={cn(
          'w-8 h-8 rounded-full shrink-0 border kb-border-tertiary flex justify-center items-center',
          {
            'bg-white': !completed,
            'kb-background-positive': completed,
          }
        )}
      >
        <CheckIcon
          className={cn('w-6 h-6', {
            'text-white': completed,
            'kb-content-tertiary': !completed,
          })}
        />
      </div>
    </div>
  )
}

export function OnboardingSidebar() {
  const { cloudProvidersCount, sourceCodeConnectionsCount, clustersCount } = {
    cloudProvidersCount: 0,
    sourceCodeConnectionsCount: 0,
    clustersCount: 0,
  }

  const onboardingSteps = [
    {
      title: 'Create your first workspace',
      description: 'A workspace provides a great way to group your projects and team members.',
      completed: true,
      Icon: WorkspaceIcon,
    },
    {
      title: 'Connect a cloud provider',
      description: 'Connect the cloud provider on which you want to provision your infrastructure.',
      completed: cloudProvidersCount > 0,
      Icon: CloudWaterdropIcon,
    },
    {
      title: 'Create your first cluster',
      description: 'Set up a server cluster to run your applications, workloads and services.',
      completed: clustersCount > 0,
      Icon: ClusterIcon,
    },
    {
      title: 'Link a source code provider',
      description: 'Link the source code provider from which you want to deploy your applications.',
      completed: sourceCodeConnectionsCount > 0,
      Icon: CodeRepositoryIcon,
    },
    {
      title: 'Launch your first project',
      description: 'Create your first project and deploy your application.',
      completed: false,
      Icon: ProjectLaunchIcon,
    },
  ]

  const percentageCompleted = Math.floor(
    (onboardingSteps.filter((step) => step.completed).length / onboardingSteps.length) * 100
  )

  return (
    <div className="w-full flex flex-col">
      <Heading size="sm">Welcome to Kibaops, Olamide</Heading>

      <Text className="mt-2 kb-content-tertiary">
        To start deploying your applications, you'll need to connect a cloud provider, link a source
        code provider, create a cluster, and launch your first project.
      </Text>

      <div className="mt-4">
        <Progress value={percentageCompleted} variant="success" />
        <Text size="sm" className="kb-content-tertiary flex justify-end mt-0.5 font-semibold">
          {percentageCompleted}% complete
        </Text>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-4">
        {onboardingSteps.map((step) => (
          <OnboardingStep key={step.title} {...step} />
        ))}
      </div>
    </div>
  )
}
