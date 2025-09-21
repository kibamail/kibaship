import { useForm, usePage } from '@inertiajs/react'
import { Button } from '@kibamail/owly/button'
import * as Dialog from '@kibamail/owly/dialog'
import { Text } from '@kibamail/owly/text'
import * as TextField from '@kibamail/owly/text-field'
import { VisuallyHidden } from '@radix-ui/react-visually-hidden'
import type { FormEventHandler } from 'react'
import { PageProps } from '~/types'

interface CreateProjectFlowProps {
  isOpen: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateProjectFlow({ isOpen, onOpenChange }: CreateProjectFlowProps) {
  const {
    props: { workspace, clusters },
  } = usePage<PageProps>()

  const { data, setData, post, processing, errors, reset } = useForm({
    name: '',
  })

  const submit: FormEventHandler = (event) => {
    event.preventDefault()

    post(`/w/${workspace?.slug}/projects`, {
      onSuccess() {
        reset()
        onOpenChange(false)
      },
    })
  }

  return (
    <Dialog.Root open={isOpen} onOpenChange={onOpenChange}>
      <Dialog.Content>
        <Dialog.Header>
          <Dialog.Title>Create a new project</Dialog.Title>
          <VisuallyHidden>
            <Dialog.Description>Create a new project</Dialog.Description>
          </VisuallyHidden>
        </Dialog.Header>

        <div className="px-5 pt-2 pb-4">
          <Text className="kb-content-secondary text-sm leading-relaxed">
            Applications help you organize your application environments, deployments,
            infrastructure and resources. Each project will automatically include staging and
            production environments.
          </Text>
        </div>

        <form onSubmit={submit}>
          <div className="px-5 pb-5 space-y-4">
            <TextField.Root
              placeholder="e.g. Marketing Website"
              name="name"
              value={data.name}
              onChange={(e) => setData('name', e.target.value)}
              required
            >
              <TextField.Label>Project name</TextField.Label>
              {errors.name && <TextField.Error>{errors.name}</TextField.Error>}
            </TextField.Root>
          </div>

          <Dialog.Footer className="flex justify-between">
            <Dialog.Close asChild disabled={processing}>
              <Button variant="secondary">Close</Button>
            </Dialog.Close>
            <Button type="submit" loading={processing}>
              Create project
            </Button>
          </Dialog.Footer>
        </form>
      </Dialog.Content>
    </Dialog.Root>
  )
}
