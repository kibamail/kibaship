import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { useState } from 'react'

interface KubeconfigDialogProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
  onSave?: (data: {clientKeyData: string, clientCertificateData: string, certificateAuthorityData: string}) => void
}

export function KubeconfigDialog({ isOpen, onOpenChange, onSave }: KubeconfigDialogProps) {
  const [formData, setFormData] = useState({
    clientKeyData: '',
    clientCertificateData: '',
    certificateAuthorityData: ''
  })

  const updateFormData = (field: keyof typeof formData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    // Here you would handle saving the kubeconfig
    console.log('Kubeconfig:', formData)
    
    // Call the onSave callback with form data
    onSave?.(formData)
    
    // Reset form and close dialog
    setFormData({
      clientKeyData: '',
      clientCertificateData: '',
      certificateAuthorityData: ''
    })
    onOpenChange(false)
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content className="!max-w-[600px]">
        <Dialog.Header>
          <Dialog.Title>Add Kubeconfig</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Add kubeconfig data for your cluster
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="text-owly-content-secondary text-sm leading-relaxed">
            Provide the kubeconfig data to authenticate with your Kubernetes cluster. These are base64-encoded values from your kubeconfig file.
          </Text>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="px-5 pb-5 space-y-4">
            <TextField.Root
              name="clientKeyData"
              value={formData.clientKeyData}
              onChange={(e) => updateFormData('clientKeyData', e.target.value)}
              placeholder="LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQ..."
              required
            >
              <TextField.Label>Client Key Data</TextField.Label>
              <TextField.Hint>Base64-encoded client private key</TextField.Hint>
            </TextField.Root>

            <TextField.Root
              name="clientCertificateData"
              value={formData.clientCertificateData}
              onChange={(e) => updateFormData('clientCertificateData', e.target.value)}
              placeholder="LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURJ..."
              required
            >
              <TextField.Label>Client Certificate Data</TextField.Label>
              <TextField.Hint>Base64-encoded client certificate</TextField.Hint>
            </TextField.Root>

            <TextField.Root
              name="certificateAuthorityData"
              value={formData.certificateAuthorityData}
              onChange={(e) => updateFormData('certificateAuthorityData', e.target.value)}
              placeholder="LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNJ..."
              required
            >
              <TextField.Label>Certificate Authority Data</TextField.Label>
              <TextField.Hint>Base64-encoded CA certificate</TextField.Hint>
            </TextField.Root>
          </div>

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild>
              <Button variant="secondary">Cancel</Button>
            </Dialog.Close>
            <Button 
              type="submit"
              disabled={!formData.clientKeyData || !formData.clientCertificateData || !formData.certificateAuthorityData}
            >
              Save Kubeconfig
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}