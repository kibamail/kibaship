import React, { useState } from 'react'
import { CopyIcon } from './Icons/copy.svg'
import { Button } from '@kibamail/owly'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import * as Dialog from '@kibamail/owly/dialog'
import { CheckIcon } from './Icons/check.svg'
type DialogConfirmationElement = React.ElementRef<'div'>

export const Confirmation = React.forwardRef<
  DialogConfirmationElement,
  React.ComponentPropsWithoutRef<'div'> & {
    item?: string
    value?: string
    processing?: boolean
    onDelete?: () => void
  }
>((props, forwardedRef) => {
  const [confirmation, setConfirmation] = useState('')
  const [copied, setCopied] = useState(false)
  const { className, item, value, processing, onDelete, ...confirmationProps } = props

  function onCopyClicked() {
    navigator.clipboard?.writeText(value || '')

    setCopied(true)

    setTimeout(() => {
      setCopied(false)
    }, 2000)
  }

  return (
    <>
      <div ref={forwardedRef} className="flex flex-col gap-4 p-6" {...confirmationProps}>
        <div className="w-full bg-owly-background-secondary border border-owly-border-tertiary px-3 py-2 rounded-md flex items-center justify-between">
          <Text className="!font-mono text-center flex-grow">{value}</Text>

          <Button variant="secondary" size="sm" onClick={copied ? undefined : onCopyClicked}>
            {copied ? <CheckIcon /> : <CopyIcon />}
          </Button>
        </div>
        <TextField.Root onChange={(event) => setConfirmation(event.target.value)}>
          <TextField.Label>Enter the name of this {item} to enable deletion</TextField.Label>
        </TextField.Root>
      </div>

      <Dialog.Footer className="flex justify-between items-center">
        <Dialog.Close asChild>
          <Button variant="secondary">Cancel</Button>
        </Dialog.Close>

        <Button
          variant="destructive"
          onClick={() => onDelete?.()}
          disabled={confirmation !== value}
          loading={processing}
        >
          Destroy cluster
        </Button>
      </Dialog.Footer>
    </>
  )
})

export * from '@kibamail/owly/dialog'