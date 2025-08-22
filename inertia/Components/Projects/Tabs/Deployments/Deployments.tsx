import { Button } from '@kibamail/owly'
import { Text } from '@kibamail/owly/text'
import { CloudUploadIcon } from '~/Components/Icons/cloud-upload.svg'
import { OpenNewWindowIcon } from '~/Components/Icons/open-new-window.svg'
import { DeploymentCard } from './DeploymentCard'

export function Deployments() {
  return (
    <>
      <div className="w-full lg:justify-between flex flex-col lg:flex-row lg:items-center mt-4 gap-4 lg:gap-0">
        <Text className="text-owly-content-secondary !text-xl !font-bold ">Latest deployments</Text>

        <div className="flex gap-2">
          <Button>
            <CloudUploadIcon className="!size-4" />
            Deploy
          </Button>
          <Button variant="secondary">
            <OpenNewWindowIcon />
            Visit
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 mt-6">
        <DeploymentCard />
        <DeploymentCard />
        <DeploymentCard />
      </div>
    </>
  )
}
