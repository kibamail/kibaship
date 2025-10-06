import type { CloudProviderType } from '~/types'
import { AWSIcon } from '~/Components/Icons/aws.svg'
import { DigitalOceanIcon } from '~/Components/Icons/digital-ocean.svg'
import { GoogleCloudIcon } from '~/Components/Icons/google-cloud.svg'
import { HetznerIcon } from '~/Components/Icons/hetzner.svg'
import { K8sIcon } from '~/Components/Icons/k8s.svg'
import { LeaseWebIcon } from '~/Components/Icons/leaseweb.svg'
import { LinodeIcon } from '~/Components/Icons/linode.svg'
import { OVHIcon } from '~/Components/Icons/ovh.svg'
import { VultrIcon } from '~/Components/Icons/vultr.svg'

export const providerIcons: Record<CloudProviderType, React.ComponentType<{ className?: string }>> = {
  aws: AWSIcon,
  hetzner: HetznerIcon,
  hetzner_robot: HetznerIcon,
  leaseweb: LeaseWebIcon,
  google_cloud: GoogleCloudIcon,
  digital_ocean: DigitalOceanIcon,
  linode: LinodeIcon,
  vultr: VultrIcon,
  ovh: OVHIcon,
  byoc: K8sIcon,
}
