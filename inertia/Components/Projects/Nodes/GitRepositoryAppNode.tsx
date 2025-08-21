import { Link } from '@inertiajs/react'
import { Node, NodeProps } from '@xyflow/react'
import { GitHubIcon } from '~/Components/Icons/github.svg'
import { useProjectPageContext } from '~/pages/projects/projects-context'

export type GitRepositoryAppNode = Node<{ getApplicationUrl?: () => string }>

export function GitRepositoryAppNode({
  id,
  data: { getApplicationUrl },
}: NodeProps<GitRepositoryAppNode>) {
  const { applicationsMap } = useProjectPageContext()

  const application = applicationsMap[id]

  return (
    <Link
      href={getApplicationUrl?.() as string}
      className="bg-white block w-[320px] rounded-[11px] min-h-[120px] border border-owly-border-tertiary relative group cursor-pointer hover:border-owly-border-brand transition ease-in-out"
    >
      <div className="w-full border-b border-owly-border-tertiary">
        <div className="px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GitHubIcon className="size-3 text-owly-content-tertiary" />
            <span className="kb-content-tertiary font-sans !text-xs">{application?.name}</span>
          </div>
        </div>
      </div>
    </Link>
  )
}
