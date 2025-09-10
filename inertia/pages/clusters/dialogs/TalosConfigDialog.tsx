import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { useState } from 'react'

interface TalosConfigDialogProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (data: {ca: string, crt: string, key: string}) => void
}

export function TalosConfigDialog({ isOpen, onOpenChange, onSave }: TalosConfigDialogProps) {
  const [formData, setFormData] = useState({
    ca: '',
    crt: '',
    key: ''
  })

  const updateFormData = (field: keyof typeof formData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    // Here you would handle saving the Talos config
    console.log('Talos Config:', formData)
    
    // Call the onSave callback with form data
    onSave?.(formData)
    
    // Reset form and close dialog
    setFormData({ ca: '', crt: '', key: '' })
    onOpenChange(false)
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content className="!max-w-[600px]">
        <Dialog.Header>
          <Dialog.Title>Add Talos Config</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Add Talos configuration for your cluster node
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="text-owly-content-secondary text-sm leading-relaxed">
            Provide the Talos configuration files. These are used to authenticate and configure your Talos cluster.
          </Text>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="px-5 pb-5 space-y-4">
            <TextField.Root
              name="ca"
              value={formData.ca}
              onChange={(e) => updateFormData('ca', e.target.value)}
              placeholder="-----BEGIN CERTIFICATE-----"
              required
            >
              <TextField.Label>CA Certificate</TextField.Label>
            </TextField.Root>

            <TextField.Root
              name="crt"
              value={formData.crt}
              onChange={(e) => updateFormData('crt', e.target.value)}
              placeholder="-----BEGIN CERTIFICATE-----"
              required
            >
              <TextField.Label>Certificate</TextField.Label>
            </TextField.Root>

            <TextField.Root
              name="key"
              value={formData.key}
              onChange={(e) => updateFormData('key', e.target.value)}
              placeholder="-----BEGIN PRIVATE KEY-----"
              required
            >
              <TextField.Label>Private Key</TextField.Label>
            </TextField.Root>
          </div>

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild>
              <Button variant="secondary">Cancel</Button>
            </Dialog.Close>
            <Button 
              type="submit"
              disabled={!formData.ca || !formData.crt || !formData.key}
            >
              Save Talos Config
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}