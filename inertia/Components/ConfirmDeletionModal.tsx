import { useState } from 'react'
import { Button } from '@kibamail/owly'
import * as Dialog from '@kibamail/owly/dialog'
import { Heading } from '@kibamail/owly/heading'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import { WarningCircleSolidIcon } from './Icons/warning-circle-solid.svg'

export function ConfirmDeletionModal({
  open,
  onOpenChange,
  title,
  description,
  onConfirm,
  loading = false,
  confirmText = 'Delete',
  requireTextMatch,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  onConfirm: () => void
  loading?: boolean
  confirmText?: string
  requireTextMatch?: string
}) {
  const [input, setInput] = useState('')
  const requireMatch = typeof requireTextMatch === 'string' && requireTextMatch.length > 0
  const disabled = loading || (requireMatch && input !== requireTextMatch)

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>{title}</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>{title}</Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-6 w-full py-6 flex flex-col gap-4">
          <div className="flex flex-col gap-2 justify-center items-center">
            <div className="h-12 w-12 rounded-md flex items-center bg-owly-border-negative/10 justify-center border border-owly-border-negative">
              <WarningCircleSolidIcon className="text-owly-content-negative" />
            </div>
            <Heading size="sm" className="text-owly-content-primary">
              {title}
            </Heading>
          </div>

          <Text className="text-owly-content-tertiary text-center">{description}</Text>

          {requireMatch && (
            <TextField.Root onChange={(e) => setInput(e.target.value)}>
              <TextField.Label>Type "{requireTextMatch}" to confirm</TextField.Label>
            </TextField.Root>
          )}
        </div>

        <Dialog.Footer className="flex justify-between items-center">
          <Dialog.Close asChild>
            <Button variant="secondary">Cancel</Button>
          </Dialog.Close>

          <Button variant="destructive" onClick={onConfirm} disabled={disabled} loading={loading}>
            {confirmText}
          </Button>
        </Dialog.Footer>
      </Dialog.Content>
    </Dialog.Root>
  )
}
