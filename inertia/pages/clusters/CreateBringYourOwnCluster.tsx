import { Button } from '@kibamail/owly/button'
import * as Select from '@kibamail/owly/select-field'
import * as TextField from '@kibamail/owly/text-field'

import { useForm, usePage } from '@inertiajs/react'
import type { PageProps } from '~/types'

interface CreateBringYourOwnClusterProps {
  onSubmit?: () => void
}

interface BringYourOwnClusterForm {
  location: string
  talosConfig: {
    ca: string
    crt: string
    key: string
  }
  type: 'byoc'
}

export function CreateBringYourOwnCluster({ onSubmit }: CreateBringYourOwnClusterProps) {
  const { workspace } = usePage<PageProps>().props
  const { data, setData, post, processing, reset } = useForm<BringYourOwnClusterForm>({
    location: '',
    talosConfig: { ca: '', crt: '', key: '' },
    type: 'byoc',
  })

  const REGIONS = [
    { group: 'North America', regions: [
      { label: 'US East (N. Virginia)', value: 'us-east-1', flag: 'us' },
      { label: 'US West (Oregon)', value: 'us-west-2', flag: 'us' },
      { label: 'Canada (Central)', value: 'ca-central-1', flag: 'ca' },
    ]},
    { group: 'Europe', regions: [
      { label: 'Ireland', value: 'eu-west-1', flag: 'ie' },
      { label: 'London', value: 'eu-west-2', flag: 'gb' },
      { label: 'Frankfurt', value: 'eu-central-1', flag: 'de' },
      { label: 'Stockholm', value: 'eu-north-1', flag: 'se' },
      { label: 'Helsinki', value: 'eu-north-2', flag: 'fi' },
    ]},
    { group: 'Asia Pacific', regions: [
      { label: 'Singapore', value: 'ap-southeast-1', flag: 'sg' },
      { label: 'Tokyo', value: 'ap-northeast-1', flag: 'jp' },
      { label: 'Sydney', value: 'ap-southeast-2', flag: 'au' },
    ]},
    { group: 'South America', regions: [
      { label: 'São Paulo', value: 'sa-east-1', flag: 'br' },
    ]},
    { group: 'Africa', regions: [
      { label: 'Cape Town', value: 'af-south-1', flag: 'za' },
    ]},
    { group: 'Middle East', regions: [
      { label: 'Bahrain', value: 'me-south-1', flag: 'bh' },
      { label: 'UAE', value: 'me-central-1', flag: 'ae' },
    ]},
  ]

  const handleSubmit = () => {
    post(`/w/${workspace.slug}/clusters/bring-your-own`, {
      onSuccess: () => {
        reset()
        onSubmit?.()
      },
    })
  }

  const canSubmit = Boolean(
    data.location && data.talosConfig.ca && data.talosConfig.crt && data.talosConfig.key
  )

  return (
    <div className="px-5">
      <div className="space-y-2">
        <Select.Root name="location" value={data.location} onValueChange={(v) => setData('location', v)}>
          <Select.Label>Region</Select.Label>
          <Select.Trigger placeholder="Select region" />
          <Select.Content className='z-50'>
            {REGIONS.map((group) => (
              <Select.Group key={group.group}>
                <Select.GroupLabel className='text-sm p-2'>{group.group}</Select.GroupLabel>
                {group.regions.map((r) => (
                  <Select.Item key={r.value} value={r.value}>
                    <span className="flex items-center gap-2">
                      <img src={`/flags/${r.flag}.svg`} alt="" className="w-4 h-4" />
                      <span>{r.label}</span>
                    </span>
                  </Select.Item>
                ))}
              </Select.Group>
            ))}
          </Select.Content>
        </Select.Root>
      </div>

      <div className="grid grid-cols-1 gap-4 mt-4">
        <TextField.Root
          name="talos.ca"
          placeholder="LS0BVUdBeXRsY0FNaEFHcVQxQ2JQakFU..."
          value={data.talosConfig.ca}
          onChange={(e) => setData('talosConfig', { ...data.talosConfig, ca: (e.target as HTMLInputElement).value })}
          required
          autoComplete="off"
          data-form-type="other"
          data-lpignore="true"
        >
          <TextField.Label>Talos ca certificate</TextField.Label>
        </TextField.Root>

        <TextField.Root
          name="talos.crt"
          placeholder="LS0BVUdBeXRsY0FNaEFHcVQxQ2JQakFU..."
          value={data.talosConfig.crt}
          onChange={(e) => setData('talosConfig', { ...data.talosConfig, crt: (e.target as HTMLInputElement).value })}
          required
          autoComplete="off"
          data-form-type="other"
          data-lpignore="true"
        >
          <TextField.Label>Talos client certificate</TextField.Label>
        </TextField.Root>

        <TextField.Root
          name="talos.key"
          placeholder="LS0BVUdBeXRsY0FNaEFHcVQxQ2JQakFU..."
          value={data.talosConfig.key}
          onChange={(e) => setData('talosConfig', { ...data.talosConfig, key: (e.target as HTMLInputElement).value })}
          required
          autoComplete="off"
          data-form-type="other"
          data-lpignore="true"
          type="password"
        >
          <TextField.Label>Talos client key</TextField.Label>
        </TextField.Root>
      </div>

      <div className="flex justify-end pt-2">
        <Button onClick={handleSubmit} loading={processing} disabled={!canSubmit || processing}>
          Create Cluster
        </Button>
      </div>


    </div>
  )
}