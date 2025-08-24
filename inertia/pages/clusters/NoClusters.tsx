import { ClusterIcon } from '~/Components/Icons/cluster.svg'
import { CreateClusterDialog } from './CreateClusterDialog'
import { Button } from '@kibamail/owly'
import { Heading } from '@kibamail/owly/heading'
import { Text } from '@kibamail/owly/text'
import { useState } from 'react'
import { PlusIcon } from '~/Components/Icons/plus.svg'

export function NoWorkspaceCluster() {
  const [isModalOpen, setIsModalOpen] = useState(false)

  function onCreateCluster() {
    setIsModalOpen(true)
  }

  return (
    <>
      <div className="w-full h-full kb-background-hover flex flex-col items-center py-24 mt-12 border kb-border-tertiary rounded-lg px-6">
        <div className="flex flex-col items-center">
          <div className="w-24 h-24 rounded-xl flex items-center justify-center bg-white border kb-border-tertiary">
            <ClusterIcon className="w-18 h-18 kb-content-positive" />
          </div>

          <div className="mt-4 flex flex-col items-center max-w-lg">
            <Heading size="md" className="font-bold">
              Create your first cluster
            </Heading>

            <Text className="text-center kb-content-tertiary mt-4">
              You have not created any clusters in this workspace yet. A cluster is a group of
              servers that work together to run your applications. Once you create a cluster, you'll
              be able to deploy and manage your applications on it.
            </Text>
          </div>

          <div className="mt-6">
            <Button variant="primary" onClick={onCreateCluster}>
              <PlusIcon /> Create cluster
            </Button>
          </div>
        </div>
      </div>

      <CreateClusterDialog isOpen={isModalOpen} onOpenChange={setIsModalOpen} />
    </>
  )
}
