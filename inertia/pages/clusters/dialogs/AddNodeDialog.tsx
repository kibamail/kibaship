import { Button } from '@kibamail/owly/button'
import {Checkbox} from '@kibamail/owly/checkbox'
import * as Dialog from '@kibamail/owly/dialog'
import * as Select from '@kibamail/owly/select-field'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { useState } from 'react'

interface AddNodeDialogProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
  onAddNode: (nodeData: {
    type: 'controlplane' | 'worker'
    publicIp: string
    privateIp: string
    diskName?: string
  }) => void
}

export function AddNodeDialog({ isOpen, onOpenChange, onAddNode }: AddNodeDialogProps) {
  const [formData, setFormData] = useState({
    nodeType: 'worker' as 'controlplane' | 'worker',
    publicIp: '',
    privateIp: '',
    diskName: '',
    storageOnControlPlanes: false
  })

  const updateFormData = (field: keyof typeof formData, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    const nodeData = {
      type: formData.nodeType,
      publicIp: formData.publicIp,
      privateIp: formData.privateIp,
      diskName: (formData.nodeType === 'worker' || formData.storageOnControlPlanes) ? formData.diskName : undefined
    }
    
    onAddNode(nodeData)
    
    // Reset form
    setFormData({
      nodeType: 'worker',
      publicIp: '',
      privateIp: '',
      diskName: '',
      storageOnControlPlanes: false
    })
    onOpenChange(false)
  }

  const showDiskField = formData.nodeType === 'worker' || (formData.nodeType === 'controlplane' && formData.storageOnControlPlanes)

  return (
    <>
      <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
        <Dialog.Content className="!max-w-[500px]">
          <Dialog.Header>
            <Dialog.Title>Add cluster node</Dialog.Title>
            <VisuallyHidden>
              <Dialog.Description>
                Add a new node to your cluster
              </Dialog.Description>
            </VisuallyHidden>
          </Dialog.Header>

          <div className="px-5 pt-2 pb-4">
            <Text className="text-owly-content-secondary text-sm leading-relaxed">
              Configure the details for your cluster node. You can add Talos config and kubeconfig after adding the node.
            </Text>
          </div>

          <form onSubmit={handleSubmit}>
            <div className="px-5 pb-5 space-y-4">
              <Select.Root
                name="type"
                value={formData.nodeType}
                onValueChange={(value) => updateFormData('nodeType', value as 'controlplane' | 'worker')}
              >
                <Select.Label>Role</Select.Label>
                <Select.Trigger placeholder="Select node type" />
                <Select.Content className='z-[100]'>
                  <Select.Item value="controlplane">Control plane</Select.Item>
                  <Select.Item value="worker">Worker</Select.Item>
                </Select.Content>
              </Select.Root>

              <TextField.Root
                name="publicIp"
                value={formData.publicIp}
                onChange={(e) => updateFormData('publicIp', e.target.value)}
                placeholder="192.168.1.100"
                required
              >
                <TextField.Label>Public IPv4 Address</TextField.Label>
              </TextField.Root>

              <TextField.Root
                name="privateIp"
                value={formData.privateIp}
                onChange={(e) => updateFormData('privateIp', e.target.value)}
                placeholder="10.0.0.100"
                required
              >
                <TextField.Label>Private IPv4 Address</TextField.Label>
              </TextField.Root>

              {showDiskField && (
                <TextField.Root
                  name="diskName"
                  value={formData.diskName}
                  onChange={(e) => updateFormData('diskName', e.target.value)}
                  placeholder="/dev/sdb"
                >
                  <TextField.Label>Disk Name (optional)</TextField.Label>
                  <TextField.Hint>
                    Specify the disk to use for storage on this node. Add this if this node will be used for storage.
                  </TextField.Hint>
                </TextField.Root>
              )}

            </div>

            <Dialog.Footer className="flex justify-between">
              <Dialog.Close asChild>
                <Button variant="secondary">Cancel</Button>
              </Dialog.Close>
              <Button 
                type="submit"
                disabled={!formData.publicIp || !formData.privateIp}
              >
                Add Node
              </Button>
            </Dialog.Footer>
          </form>
        </Dialog.Content>
      </Dialog.Root>
    </>
  )
}