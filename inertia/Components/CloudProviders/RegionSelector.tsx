import type { CloudProviderRegion, CloudProviderType, PageProps } from '@/types'
import { usePage } from '@inertiajs/react'
import * as SelectField from '@kibamail/owly/select-field'

interface RegionSelectorProps {
  providerType: CloudProviderType
  selectedRegion?: string
  onRegionChange: (regionSlug: string) => void
  placeholder?: string
  disabled?: boolean
  groupByContinent?: boolean
}

/**
 * RegionSelector component that displays available regions for a specific cloud provider.
 * Uses the globally shared cloud provider regions data from Inertia middleware.
 * Regions are grouped by continent for better organization.
 */
export function RegionSelector({
  providerType,
  selectedRegion,
  onRegionChange,
  placeholder = 'Select a region',
  disabled = false,
  groupByContinent = true,
}: RegionSelectorProps) {
  const { cloudProviderRegions } = usePage<PageProps>().props

  const regionsByContinent = cloudProviderRegions[providerType] || {}
  const hasRegions = Object.keys(regionsByContinent).length > 0

  if (!hasRegions) {
    return (
      <SelectField.Root disabled>
        <SelectField.Trigger placeholder="No regions available" />
      </SelectField.Root>
    )
  }

  return (
    <SelectField.Root value={selectedRegion} onValueChange={onRegionChange} disabled={disabled}>
      <SelectField.Trigger placeholder={placeholder} />
      <SelectField.Content className="z-100">
        {groupByContinent
          ? Object.entries(regionsByContinent).map(([continent, regions]) => (
              <div key={continent}>
                <div className="px-2 py-1.5 text-xs font-semibold kb-content-tertiary uppercase tracking-wider">
                  {continent}
                </div>
                {regions.map((region: CloudProviderRegion) => (
                  <SelectField.Item key={region.slug} value={region.slug}>
                    <div className="flex items-center gap-2">
                      <img
                        src={region.flag}
                        alt={`${region.name} flag`}
                        className="w-4 h-3 object-cover flex-shrink-0"
                      />
                      <span>{region.name}</span>
                    </div>
                  </SelectField.Item>
                ))}
              </div>
            ))
          : // Flat list without continent grouping
            Object.values(regionsByContinent)
              .flat()
              .map((region) => (
                <SelectField.Item key={region.slug} value={region.slug}>
                  <div className="flex items-center gap-2">
                    <img
                      src={region.flag}
                      alt={`${region.name} flag`}
                      className="w-4 h-3 object-cover rounded-sm flex-shrink-0"
                    />
                    <span>{region.name}</span>
                  </div>
                </SelectField.Item>
              ))}
      </SelectField.Content>
    </SelectField.Root>
  )
}
