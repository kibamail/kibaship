import { usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import * as Select from '@kibamail/owly/select-field'
import { Text } from '@kibamail/owly/text'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { useEffect, useState } from 'react'
import type { PageProps, CloudProvider } from '~/types'
import Spinner from '~/Components/Icons/Spinner'

interface HetznerRobotServer {
  server_ip: string
  server_ipv6_net?: string
  server_number: number
  server_name: string
  product: string
  dc: string
  traffic: string
  status: 'ready' | 'in process'
  cancelled: boolean
  paid_until: string
  ip: string[]
  subnet?: Array<{ ip?: string; mask?: string }> | null
}

interface CreateHetznerRobotServerDialogProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
  cloudProviderId: string
}

export function CreateHetznerRobotServerDialog({
  isOpen,
  onOpenChange,
  cloudProviderId,
}: CreateHetznerRobotServerDialogProps) {
  const { props } = usePage<PageProps>()
  const [servers, setServers] = useState<HetznerRobotServer[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedServer, setSelectedServer] = useState<string>('')

  useEffect(() => {
    if (isOpen && cloudProviderId) {
      setLoading(true)

      fetch(`/connections/cloud-providers/${cloudProviderId}/servers`)
        .then((response) => response.json())
        .then((data) => {
          setServers(data)
          setLoading(false)
        })
        .catch((error) => {
          console.error('Failed to fetch servers:', error)
          setLoading(false)
        })
    }
  }, [isOpen, cloudProviderId])

  const handleServerChange = (value: string) => {
    setSelectedServer(value)
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>Create Hetzner Robot Server</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>
              Select a Hetzner Robot dedicated server to provision
            </Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="kb-content-secondary text-sm leading-relaxed">
            Select a dedicated server from your Hetzner Robot account to provision and manage with
            Kibaship.
          </Text>
        </div>

        <div className="px-5 pb-5 space-y-4">
          <Select.Root
            name="server"
            value={selectedServer}
            onValueChange={handleServerChange}
            disabled={loading}
          >
            <Select.Label>Server</Select.Label>
            <Select.Trigger
              placeholder={loading ? 'Loading servers...' : 'Select a server'}
            />
            <Select.Content className="z-50 relative w-full">
              {servers.map((server) => (
                <Select.Item key={server.server_number} value={server.server_number.toString()}>
                  <div className="flex items-center justify-between w-full">
                    <div className="flex flex-col">
                      <span className="font-medium">{server.server_name}</span>
                      <span className="text-xs kb-content-tertiary">
                        {server.product} • {server.dc} • {server.server_ip}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span
                        className={`text-xs px-2 py-0.5 rounded-full ${
                          server.status === 'ready'
                            ? 'kb-content-positive kb-background-positive'
                            : 'kb-content-notice kb-background-notice'
                        }`}
                      >
                        {server.status}
                      </span>
                    </div>
                  </div>
                </Select.Item>
              ))}
            </Select.Content>
          </Select.Root>

          {loading && (
            <div className="flex items-center gap-2 py-2">
              <Spinner className="w-4 h-4 kb-content-tertiary" />
              <Text className="kb-content-tertiary text-sm">
                Fetching servers from your Hetzner Robot account...
              </Text>
            </div>
          )}
        </div>

        <Dialog.Footer className="flex justify-between">
          <Dialog.Close asChild disabled={loading}>
            <Button variant="secondary">Cancel</Button>
          </Dialog.Close>
          <Button type="button" disabled={!selectedServer || loading}>
            Provision Server
          </Button>
        </Dialog.Footer>
      </Dialog.Content>
    </Dialog.Root>
  )
}
