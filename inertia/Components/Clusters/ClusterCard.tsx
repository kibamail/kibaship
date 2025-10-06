import { Text } from '@kibamail/owly/text'
import { Cluster } from '~/types'
import classNames from 'classnames'
import { OptionsDropdown } from '../OptionsDropdown'
import { SettingsIcon } from '../Icons/settings.svg'
import { TrashIcon } from '../Icons/trash.svg'
import { getStatusIcon } from './ClusterProvisioningStep'
import { ItemAvatar } from '../ItemAvatar'
import { providerIcons } from '~/lib/providerIcons'

interface ProjectCardProps {
  cluster: Cluster
  onClusterSelected?: (cluster: Cluster) => void
  onClusterDeletion?: (cluster: Cluster) => void
}

export function ClusterCard({ cluster, onClusterSelected, onClusterDeletion }: ProjectCardProps) {
  const statusClassNames = (() => {
    if (cluster.provisioningStatus === 'ready') {
      return 'text-owly-content-positive'
    }

    if (cluster.deletedAt) {
      return 'text-owly-content-negative'
    }

    if (cluster.status === 'provisioning') {
      return 'text-owly-content-notice'
    }

    return ''
  })()

  const totalControlPlanes = cluster.nodes?.filter((node) => node.type === 'master')?.length || 0
  const totalWorkerNodes = cluster.nodes?.filter((node) => node.type === 'worker')?.length || 0

  function onCardClick(event: React.MouseEvent) {
    const dropdownMenuContent = document.querySelector('#cluster-card-dropdown-menu-content')

    if (dropdownMenuContent?.contains(event.target as Node)) {
      return
    }

    onClusterSelected?.(cluster)
  }

  const CloudProviderIcon = providerIcons[cluster?.cloudProvider?.type]

  return (
    <>
      <div
        onClick={onCardClick}
        className="w-full bg-white rounded-[11px] h-[130px] border border-owly-border-tertiary relative group cursor-pointer"
      >
        <div className="absolute w-[calc(100%+10px)] h-[calc(100%+10px)] rounded-[14px] transform translate-x-[-5px] translate-y-[-5px] bg-transparent transition-all ease-in-out border border-transparent group-hover:border-owly-border-tertiary"></div>
        <div className="p-4 z-1 flex flex-col justify-between h-full relative cursor-pointer rounded-[11px]">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <ItemAvatar
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

              <OptionsDropdown
                id="cluster-card-dropdown-menu-content"
                items={[
                  { icon: SettingsIcon, name: 'Settings', onClick: () => {} },
                  {
                    icon: TrashIcon,
                    name: 'Destroy',
                    onClick: () => onClusterDeletion?.(cluster),
                    type: 'destructive',
                  },
                ]}
              />
            </div>
          </div>

          <div className="gap-y-1.5 flex justify-between items-center">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <img
                  src={`${cluster.region.flag}`}
                  alt={cluster?.region.name}
                  className="w-6 h-4 object-cover flex-shrink-0 shadow"
                />

                <Text className="text-owly-content-tertiary">{cluster?.region.name}</Text>
              </div>

              {CloudProviderIcon ? <CloudProviderIcon className="w-6 h-6" /> : null}
            </div>

            <Text className="text-owly-content-tertiary text-sm">
              {totalControlPlanes} control planes, {totalWorkerNodes} worker nodes
            </Text>
          </div>
        </div>
      </div>
    </>
  )
}
