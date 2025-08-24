import { CreateCloudProviderDialog } from './CreateCloudProviderDialog'
import { AWSIcon } from '~/Components/Icons/aws.svg'
import { CloudWaterdropIcon } from '~/Components/Icons/cloud-waterdrop.svg'
import { DigitalOceanIcon } from '~/Components/Icons/digital-ocean.svg'
import { GoogleCloudIcon } from '~/Components/Icons/google-cloud.svg'
import { HetznerIcon } from '~/Components/Icons/hetzner.svg'
import { LeaseWebIcon } from '~/Components/Icons/leaseweb.svg'
import { LinodeIcon } from '~/Components/Icons/linode.svg'
import { OVHIcon } from '~/Components/Icons/ovh.svg'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { VultrIcon } from '~/Components/Icons/vultr.svg'
import type {
  CloudProviderInfo,
  CloudProviderRegion,
  CloudProviderServerType,
  CloudProviderType,
  PageProps,
} from '~/types'
import { usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly'
import { Heading } from '@kibamail/owly/heading'
import { Text } from '@kibamail/owly/text'
import { useState } from 'react'

interface CloudProviderWithIcon extends CloudProviderInfo {
  icon: React.ComponentType<{ className?: string }>
}

// Icon mapping for cloud providers
const providerIcons: Record<CloudProviderType, React.ComponentType<{ className?: string }>> = {
  aws: AWSIcon,
  hetzner: HetznerIcon,
  leaseweb: LeaseWebIcon,
  google_cloud: GoogleCloudIcon,
  digital_ocean: DigitalOceanIcon,
  linode: LinodeIcon,
  vultr: VultrIcon,
  ovh: OVHIcon,
}

export function NoCloudProviders() {
  const { props } = usePage<
    PageProps & {
      regions: CloudProviderRegion[]
      serverTypes: CloudProviderServerType[]
      providers: CloudProviderInfo[]
    }
  >()
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [selectedProviderType, setSelectedProviderType] = useState<CloudProviderType | undefined>(
    undefined
  )

  const handleConnectProvider = (providerType: CloudProviderType) => {
    setSelectedProviderType(providerType)
    setIsDialogOpen(true)
  }

  const handleDialogOpenChange = (open: boolean) => {
    setIsDialogOpen(open)
    if (!open) {
      setSelectedProviderType(undefined)
    }
  }

  const cloudProvidersWithIcons: CloudProviderWithIcon[] = props.providers
    .map((provider) => ({
      ...provider,
      icon: providerIcons[provider.type],
    }))
    .sort((a, b) => {
      if (a.implemented && !b.implemented) return -1
      if (!a.implemented && b.implemented) return 1
      return 0
    })

  return (
    <>
      <div className="w-full h-full kb-background-hover flex flex-col items-center py-24 mt-12 border kb-border-tertiary rounded-lg px-6">
        <div className="flex flex-col items-center">
          <div className="w-24 h-24 rounded-xl flex items-center justify-center bg-white border kb-border-tertiary">
            <CloudWaterdropIcon className="w-18 h-18 kb-content-positive" />
          </div>

          <div className="mt-4 flex flex-col items-center max-w-lg">
            <Heading size="md" className="font-bold">
              Connect a cloud provider
            </Heading>

            <Text className="text-center kb-content-tertiary mt-4">
              You have not connected any cloud providers to this workspace yet. Once you do, you'll
              be able to provision your first cluster. You may connect multiple cloud providers to a
              single workspace.
            </Text>
          </div>

          <div className="w-full mt-6 flex flex-col gap-4 max-w-lg mx-auto">
            {cloudProvidersWithIcons.map((provider) => {
              const IconComponent = provider.icon
              return (
                <div
                  key={provider.type}
                  className="w-full flex items-center justify-between rounded-md border kb-border-tertiary p-2.5 bg-white"
                >
                  <div className="flex items-center gap-2">
                    <IconComponent className="w-6 h-6" />
                    <Text>{provider.name}</Text>
                    {!provider.implemented && (
                      <Text
                        size="xs"
                        className="text-xs px-2 py-0.5 lowercase kb-content-disabled font-bold kb-background-hover border kb-border-tertiary rounded-full"
                      >
                        Coming Soon
                      </Text>
                    )}
                  </div>

                  <Button
                    size="sm"
                    className="pr-1"
                    variant="secondary"
                    disabled={!provider.implemented}
                    onClick={() => handleConnectProvider(provider.type)}
                  >
                    <PlusIcon />
                    Connect provider
                  </Button>
                </div>
              )
            })}
          </div>
        </div>
      </div>

      <CreateCloudProviderDialog
        isOpen={isDialogOpen}
        onOpenChange={handleDialogOpenChange}
        preselectedProviderType={selectedProviderType}
      />
    </>
  )
}
