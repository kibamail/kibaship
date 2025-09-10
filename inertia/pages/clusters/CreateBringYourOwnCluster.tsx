import { Button } from '@kibamail/owly/button'
import { Text } from '@kibamail/owly/text'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { useState } from 'react'
import { useForm, usePage } from '@inertiajs/react'
import type { PageProps } from '~/types'
import { AddNodeDialog } from './dialogs/AddNodeDialog'
import { TalosConfigDialog } from './dialogs/TalosConfigDialog'
import { KubeconfigDialog } from './dialogs/KubeconfigDialog'

interface CreateBringYourOwnClusterProps {
  onSubmit?: () => void
}

interface BringYourOwnClusterForm {
  nodes: Array<{
    id: string
    type: 'controlplane' | 'worker'
    publicIp: string
    privateIp: string
    diskName?: string
  }>
  talosConfig: {
    ca: string
    crt: string
    key: string
  } | null
  kubeconfig: {
    clientKeyData: string
    clientCertificateData: string
    certificateAuthorityData: string
  } | null
  type: 'bring_your_own'
}

export function CreateBringYourOwnCluster({ onSubmit }: CreateBringYourOwnClusterProps) {
  const { workspace } = usePage<PageProps>().props
  const { data, setData, post, processing, reset } = useForm<BringYourOwnClusterForm>({
    nodes: [],
    talosConfig: null,
    kubeconfig: null,
    type: 'bring_your_own'
  })
  
  const [isAddNodeDialogOpen, setIsAddNodeDialogOpen] = useState(false)
  const [isTalosConfigDialogOpen, setIsTalosConfigDialogOpen] = useState(false)
  const [isKubeconfigDialogOpen, setIsKubeconfigDialogOpen] = useState(false)

  const handleAddNode = (nodeData: Omit<BringYourOwnClusterForm['nodes'][number], 'id'>) => {
    const newNode = {
      id: Math.random().toString(36).substring(2, 11).toString(),
      ...nodeData
    }
    setData('nodes', [...data.nodes, newNode])
  }

  const handleTalosConfigSave = (configData: BringYourOwnClusterForm['talosConfig']) => {
    setData('talosConfig', configData)
  }

  const handleKubeconfigSave = (configData: BringYourOwnClusterForm['kubeconfig']) => {
    setData('kubeconfig', configData)
  }

  const handleSubmit = () => {
    post(`/w/${workspace.slug}/clusters/bring-your-own`, {
      onSuccess: () => {
        reset()
        onSubmit?.()
      },
    })
  }

  const canSubmit = data.nodes.length > 0 && data.talosConfig !== null && data.kubeconfig !== null

  return (
    <div className="px-5 pb-5 space-y-4">
      <div className="w-full flex flex-col p-3 rounded-md bg-owly-background-secondary border border-owly-border-tertiary">
        <Text className="text-left text-owly-content-tertiary !text-sm">
          Add your cluster nodes to get started. You can add both control plane and worker nodes.
        </Text>

        <div className="my-5 flex justify-center">
          <Button 
            variant="secondary" 
            onClick={() => setIsAddNodeDialogOpen(true)}
          >
            <PlusIcon className="!size-4 mr-2" />
            Add cluster node
          </Button>
        </div>
      </div>

      {data.nodes.length > 0 && (
        <div className="space-y-3">
          <Text className="font-medium">Added Nodes ({data.nodes.length})</Text>
          {data.nodes.map((node) => (
            <div 
              key={node.id} 
              className="p-3 border border-owly-border-tertiary rounded-md bg-owly-background-primary"
            >
              <div className="flex justify-between items-start">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className={`px-2 py-0.5 text-xs rounded-full ${
                      node.type === 'controlplane' 
                        ? 'bg-blue-100 text-blue-800' 
                        : 'bg-green-100 text-green-800'
                    }`}>
                      {node.type}
                    </span>
                  </div>
                  <Text className="text-sm">
                    Public: {node.publicIp} | Private: {node.privateIp}
                  </Text>
                  {node.diskName && (
                    <Text className="text-sm text-owly-content-tertiary">
                      Disk: {node.diskName}
                    </Text>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Talos Configuration */}
      <div className="w-full flex flex-col p-3 rounded-md bg-owly-background-hover border border-owly-border-tertiary">
        <Text className="text-left text-owly-content-tertiary !text-sm">
          Add your Talos configuration for cluster authentication and management.
        </Text>

        <div className="my-5 flex justify-center">
          <Button 
            variant="secondary" 
            onClick={() => setIsTalosConfigDialogOpen(true)}
          >
            {data.talosConfig ? (
              <>✓ Talos Config Added</>
            ) : (
              <>
                <PlusIcon className="!size-4 mr-2" />
                Add talos config
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Kubeconfig */}
      <div className="w-full flex flex-col p-3 rounded-md bg-owly-background-hover border border-owly-border-tertiary">
        <Text className="text-left text-owly-content-tertiary !text-sm">
          Add your kubeconfig data to authenticate with the Kubernetes cluster.
        </Text>

        <div className="my-5 flex justify-center">
          <Button 
            variant="secondary" 
            onClick={() => setIsKubeconfigDialogOpen(true)}
          >
            {data.kubeconfig ? (
              <>✓ Kubeconfig Added</>
            ) : (
              <>
                <PlusIcon className="!size-4 mr-2" />
                Add kubeconfig
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Submit Button */}
      {canSubmit && (
        <div className="flex justify-end pt-4">
          <Button 
            onClick={handleSubmit}
            loading={processing}
            disabled={processing}
          >
            Create Cluster
          </Button>
        </div>
      )}

      <AddNodeDialog
        isOpen={isAddNodeDialogOpen}
        onOpenChange={setIsAddNodeDialogOpen}
        onAddNode={handleAddNode}
      />

      <TalosConfigDialog
        isOpen={isTalosConfigDialogOpen}
        onOpenChange={setIsTalosConfigDialogOpen}
        onSave={handleTalosConfigSave}
      />

      <KubeconfigDialog
        isOpen={isKubeconfigDialogOpen}
        onOpenChange={setIsKubeconfigDialogOpen}
        onSave={handleKubeconfigSave}
      />
    </div>
  )
}