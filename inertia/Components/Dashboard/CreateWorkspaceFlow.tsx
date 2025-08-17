import { useForm } from '@inertiajs/react'
import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import type { FormEventHandler } from 'react'

interface CreateWorkspaceFlowProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateWorkspaceFlow({ isOpen, onOpenChange }: CreateWorkspaceFlowProps) {
  const { data, setData, post, processing, errors, reset } = useForm({
    name: '',
  })

  const submit: FormEventHandler = (event) => {
    event.preventDefault()

    post('/w/workspaces', {
      onSuccess(response) {
        console.log({ response })
        reset()
        onOpenChange(false)
        window.location.reload()
      },
    })
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>Create a new workspace</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>Create a new workspace</Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="kb-content-secondary text-sm leading-relaxed">
            Workspaces help you organize your projects and team members. Each workspace has its own
            projects, environments, and team members, keeping your work separate and organized.
          </Text>
        </div>

        <form onSubmit={submit}>
          <div className="px-5 pb-5">
            <TextField.Root
              placeholder="e.g. Marketing Team"
              name="name"
              value={data.name}
              onChange={(e) => setData('name', e.target.value)}
              required
            >
              <TextField.Label>Workspace name</TextField.Label>
              {errors.name && <TextField.Error>{errors.name}</TextField.Error>}
            </TextField.Root>
          </div>

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild disabled={processing}>
              <Button variant="secondary">Close</Button>
            </Dialog.Close>
            <Button type="submit" loading={processing}>
              Create workspace
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}
