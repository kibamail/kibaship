import type { CloudProviderType } from '@/types';
import { Heading } from '@kibamail/owly/heading';
import * as SelectField from '@kibamail/owly/select-field';
import { Text } from '@kibamail/owly/text';
import { useState } from 'react';
import { RegionSelector, useAllCloudProviderRegions } from './RegionSelector';

/**
 * Example component demonstrating how to use cloud provider regions data.
 * This shows how frontend components can access the globally shared regions data.
 */
export function CloudProviderRegionsExample() {
  const [selectedProvider, setSelectedProvider] = useState<CloudProviderType>('hetzner');
  const [selectedRegion, setSelectedRegion] = useState<string>('');

  const allRegions = useAllCloudProviderRegions();

  const providerOptions = [
    { value: 'aws', label: 'Amazon Web Services' },
    { value: 'hetzner', label: 'Hetzner Cloud' },
    { value: 'digital_ocean', label: 'DigitalOcean' },
    { value: 'google_cloud', label: 'Google Cloud Platform' },
    { value: 'vultr', label: 'Vultr' },
    { value: 'linode', label: 'Linode' },
    { value: 'leaseweb', label: 'LeaseWeb' },
  ] as const;

  const handleProviderChange = (provider: string) => {
    setSelectedProvider(provider as CloudProviderType);
    setSelectedRegion(''); // Reset region when provider changes
  };

  return (
    <div className="max-w-2xl mx-auto p-6 space-y-6">
      <div>
        <Heading size="lg" className="mb-2">
          Cloud Provider Regions Example
        </Heading>
        <Text className="kb-content-tertiary">
          This example demonstrates how frontend components can access the globally shared cloud
          provider regions data through Inertia middleware.
        </Text>
      </div>

      <div className="space-y-4">
        <div>
          <label htmlFor="provider-select" className="block text-sm font-medium mb-2">
            Select Cloud Provider
          </label>
          <SelectField.Root value={selectedProvider} onValueChange={handleProviderChange}>
            <SelectField.Trigger placeholder="Select a provider" />
            <SelectField.Content>
              {providerOptions.map((option) => (
                <SelectField.Item key={option.value} value={option.value}>
                  {option.label}
                </SelectField.Item>
              ))}
            </SelectField.Content>
          </SelectField.Root>
        </div>

        <div>
          <label htmlFor="region-select" className="block text-sm font-medium mb-2">
            Select Region
          </label>
          <RegionSelector
            providerType={selectedProvider}
            selectedRegion={selectedRegion}
            onRegionChange={setSelectedRegion}
            placeholder="Choose a region"
          />
        </div>

        {selectedProvider && selectedRegion && (
          <div className="p-4 bg-green-50 border border-green-200 rounded-lg">
            <Text className="text-green-800">
              <strong>Selected:</strong> {selectedProvider} in region {selectedRegion}
            </Text>
          </div>
        )}
      </div>

      <div className="mt-8">
        <Heading size="md" className="mb-4">
          Available Regions Summary
        </Heading>
        <div className="space-y-2">
          {Object.entries(allRegions).map(([provider, regionsByContinent]) => {
            const totalRegions = Object.values(regionsByContinent).flat().length;
            const continentCount = Object.keys(regionsByContinent).length;

            return (
              <div key={provider} className="p-3 bg-gray-50 rounded-lg">
                <div className="flex justify-between items-center mb-2">
                  <Text className="font-medium capitalize">{provider.replace('_', ' ')}</Text>
                  <Text className="kb-content-tertiary">
                    {totalRegions} regions across {continentCount} continents
                  </Text>
                </div>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(regionsByContinent).map(([continent, regions]) => (
                    <span
                      key={continent}
                      className="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded-md"
                    >
                      {continent}: {regions.length}
                    </span>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
