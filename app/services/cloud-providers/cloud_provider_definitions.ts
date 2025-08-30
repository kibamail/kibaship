import { locations as hetznerLocations } from './constants/hetzner/locations.js'
import { serverTypes as hetznerServerTypes } from './constants/hetzner/server_types.js'
import { regions as digitalOceanRegions } from './constants/digital_ocean/regions.js'
import { sizes as digitalOceanSizes } from './constants/digital_ocean/droplet_sizes.js'

export type CloudProviderType =
    | 'aws'
    | 'hetzner'
    | 'leaseweb'
    | 'google_cloud'
    | 'digital_ocean'
    | 'linode'
    | 'vultr'
    | 'ovh'

export interface CredentialField {
    name: string
    label: string
    type: 'text' | 'password' | 'textarea'
    placeholder: string
    required: boolean
}

export interface Region {
    name: string
    slug: string
    flag: string
    availableServerTypes?: Record<string, boolean>
}

export interface RegionsByContinent {
    [continent: string]: Region[]
}

export interface RegionServerTypes {
    [regionSlug: string]: Record<string, boolean>
}

export interface ServerType {
    cpu: number
    ram: number
    disk: number
    name: string
}

export interface ServerTypes {
    [typeSlug: string]: ServerType
}

export interface CloudProviderOption {
    value: CloudProviderType
    label: string
}

export interface CloudProviderData {
    type: CloudProviderType
    name: string
    implemented: boolean
    description: string
    documentationLink: string
    credentialFields: CredentialField[]
}

export class CloudProviderDefinitions {
    static readonly AWS: CloudProviderType = 'aws'
    static readonly HETZNER: CloudProviderType = 'hetzner'
    static readonly LEASEWEB: CloudProviderType = 'leaseweb'
    static readonly GOOGLE_CLOUD: CloudProviderType = 'google_cloud'
    static readonly DIGITAL_OCEAN: CloudProviderType = 'digital_ocean'
    static readonly LINODE: CloudProviderType = 'linode'
    static readonly VULTR: CloudProviderType = 'vultr'
    static readonly OVH: CloudProviderType = 'ovh'

    private static transformHetznerLocationsToRegions(): RegionsByContinent {
        const continentMap: Record<string, string> = {
            'eu-central': 'Europe',
            'us-east': 'North America',
            'us-west': 'North America',
            'ap-southeast': 'Asia Pacific',
        }

        const regionsByContinent: RegionsByContinent = {}

        hetznerLocations.forEach((location) => {
            const continent = continentMap[location.network_zone] || 'Other'

            if (!regionsByContinent[continent]) {
                regionsByContinent[continent] = []
            }

            const countryFlags: Record<string, string> = {
                DE: '/flags/de.svg',
                FI: '/flags/fi.svg',
                US: '/flags/us.svg',
                SG: '/flags/sg.svg',
            }

            // Find all server types available in this location
            const availableServerTypes: Record<string, boolean> = {}
            hetznerServerTypes.forEach(serverType => {
                const isAvailable = serverType.prices.some(price => price.location === location.name)
                availableServerTypes[serverType.name] = isAvailable
            })

            regionsByContinent[continent].push({
                name: `${location.city}, ${location.country === 'DE' ? 'Germany' : location.country === 'FI' ? 'Finland' : location.country === 'US' ? 'USA' : location.country === 'SG' ? 'Singapore' : location.country}`,
                slug: location.name,
                flag: countryFlags[location.country] || '/flags/unknown.svg',
                availableServerTypes
            })
        })

        return regionsByContinent
    }

    private static transformDigitalOceanRegionsToRegions(): RegionsByContinent {
        const continentMap: Record<string, string> = {
            nyc1: 'North America',
            nyc2: 'North America',
            nyc3: 'North America',
            sfo1: 'North America',
            sfo2: 'North America',
            sfo3: 'North America',
            tor1: 'North America',
            atl1: 'North America',
            ams2: 'Europe',
            ams3: 'Europe',
            lon1: 'Europe',
            fra1: 'Europe',
            sgp1: 'Asia Pacific',
            blr1: 'Asia Pacific',
            syd1: 'Asia Pacific',
        }

        const regionsByContinent: RegionsByContinent = {}

        digitalOceanRegions
            .filter((region) => region.available)
            .forEach((region) => {
                const continent = continentMap[region.slug] || 'Other'

                if (!regionsByContinent[continent]) {
                    regionsByContinent[continent] = []
                }

                const flagMap: Record<string, string> = {
                    nyc1: '/flags/us.svg',
                    nyc2: '/flags/us.svg',
                    nyc3: '/flags/us.svg',
                    sfo1: '/flags/us.svg',
                    sfo2: '/flags/us.svg',
                    sfo3: '/flags/us.svg',
                    atl1: '/flags/us.svg',
                    tor1: '/flags/ca.svg',
                    ams2: '/flags/nl.svg',
                    ams3: '/flags/nl.svg',
                    lon1: '/flags/gb.svg',
                    fra1: '/flags/de.svg',
                    sgp1: '/flags/sg.svg',
                    blr1: '/flags/in.svg',
                    syd1: '/flags/au.svg',
                }

                // Find all droplet sizes available in this region
                const availableServerTypes: Record<string, boolean> = {}
                digitalOceanSizes.forEach(size => {
                    availableServerTypes[size.slug] = size.regions.includes(region.slug)
                })

                regionsByContinent[continent].push({
                    name: region.name,
                    slug: region.slug,
                    flag: flagMap[region.slug] || '/flags/unknown.svg',
                    availableServerTypes
                })
            })

        return regionsByContinent
    }

    private static transformHetznerServerTypes(): ServerTypes {
        const serverTypes: ServerTypes = {}

        hetznerServerTypes.forEach((serverType) => {
            const ramInGB = serverType.memory
            const diskInGB = serverType.disk
            const cpuCount = serverType.cores

            serverTypes[serverType.name] = {
                cpu: cpuCount,
                ram: ramInGB,
                disk: diskInGB,
                name: `${serverType.name.toUpperCase()} - ${cpuCount} vCPU, ${ramInGB}GB RAM, ${diskInGB}GB SSD`,
            }
        })

        return serverTypes
    }

    private static transformDigitalOceanSizes(): ServerTypes {
        const serverTypes: ServerTypes = {}

        digitalOceanSizes.forEach((size) => {
            const ramInGB = Math.round(size.memory / 1024)
            const diskInGB = size.disk
            const cpuCount = size.vcpus

            serverTypes[size.slug] = {
                cpu: cpuCount,
                ram: ramInGB,
                disk: diskInGB,
                name: `${size.description} - ${cpuCount} vCPU, ${ramInGB}GB RAM, ${diskInGB}GB SSD`,
            }
        })

        return serverTypes
    }

    private static transformHetznerRegionServerTypes(): RegionServerTypes {
        const regionServerTypes: RegionServerTypes = {}

        hetznerLocations.forEach(location => {
            const availableServerTypes: Record<string, boolean> = {}
            hetznerServerTypes.forEach(serverType => {
                const isAvailable = serverType.prices.some(price => price.location === location.name)
                availableServerTypes[serverType.name] = isAvailable
            })
            regionServerTypes[location.name] = availableServerTypes
        })

        return regionServerTypes
    }

    private static transformDigitalOceanRegionServerTypes(): RegionServerTypes {
        const regionServerTypes: RegionServerTypes = {}

        digitalOceanRegions.filter(region => region.available).forEach(region => {
            const availableServerTypes: Record<string, boolean> = {}
            digitalOceanSizes.forEach(size => {
                availableServerTypes[size.slug] = size.regions.includes(region.slug)
            })
            regionServerTypes[region.slug] = availableServerTypes
        })

        return regionServerTypes
    }

    static values(): CloudProviderType[] {
        return [
            this.AWS,
            this.HETZNER,
            this.GOOGLE_CLOUD,
            this.DIGITAL_OCEAN,
            // this.LEASEWEB,
            // this.LINODE,
            // this.VULTR,
            // this.OVH,
        ]
    }

    static label(type: CloudProviderType): string {
        switch (type) {
            case this.AWS:
                return 'Amazon web services'
            case this.HETZNER:
                return 'Hetzner cloud'
            case this.LEASEWEB:
                return 'Lease web'
            case this.GOOGLE_CLOUD:
                return 'Google cloud platform'
            case this.DIGITAL_OCEAN:
                return 'Digital ocean'
            case this.LINODE:
                return 'Linode'
            case this.VULTR:
                return 'Vultr'
            case this.OVH:
                return 'OVH'
            default:
                throw new Error(`Unknown cloud provider type: ${type}`)
        }
    }

    static description(type: CloudProviderType): string {
        switch (type) {
            case this.AWS:
                return 'Create an IAM user with programmatic access in your AWS console.'
            case this.HETZNER:
                return 'Generate an API token in your Hetzner Cloud console under Security > API Tokens.'
            case this.DIGITAL_OCEAN:
                return 'Create a personal access token in your DigitalOcean control panel under API > Tokens.'
            case this.GOOGLE_CLOUD:
                return 'Create a service account and download the JSON key file from Google Cloud Console.'
            case this.LEASEWEB:
                return 'Generate an API key in your LeaseWeb customer portal under API Management.'
            case this.LINODE:
                return 'Create a personal access token in your Linode Cloud Manager under My Profile > API Tokens.'
            case this.VULTR:
                return 'Generate an API key in your Vultr customer portal under Account > API.'
            case this.OVH:
                return 'Create API credentials in your OVH control panel under Advanced > API Management.'
            default:
                throw new Error(`Unknown cloud provider type: ${type}`)
        }
    }

    static documentationLink(type: CloudProviderType): string {
        switch (type) {
            case this.AWS:
                return 'https://kibaship.com/docs/providers/aws'
            case this.HETZNER:
                return 'https://kibaship.com/docs/providers/hetzner'
            case this.DIGITAL_OCEAN:
                return 'https://kibaship.com/docs/providers/digitalocean'
            case this.GOOGLE_CLOUD:
                return 'https://kibaship.com/docs/providers/gcp'
            case this.LEASEWEB:
                return 'https://kibaship.com/docs/providers/leaseweb'
            case this.LINODE:
                return 'https://kibaship.com/docs/providers/linode'
            case this.VULTR:
                return 'https://kibaship.com/docs/providers/vultr'
            case this.OVH:
                return 'https://kibaship.com/docs/providers/ovh'
            default:
                throw new Error(`Unknown cloud provider type: ${type}`)
        }
    }

    static credentialFields(type: CloudProviderType): CredentialField[] {
        switch (type) {
            case this.AWS:
                return [
                    {
                        name: 'access_key',
                        label: 'Access key ID',
                        type: 'text',
                        placeholder: 'AKIAIOSFODNN7EXAMPLE',
                        required: true,
                    },
                    {
                        name: 'secret_key',
                        label: 'Secret access key',
                        type: 'password',
                        placeholder: 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
                        required: true,
                    },
                ]
            case this.HETZNER:
                return [
                    {
                        name: 'token',
                        label: 'API token',
                        type: 'password',
                        placeholder: 'Enter your hetzner cloud api token',
                        required: true,
                    },
                ]
            case this.DIGITAL_OCEAN:
                return [
                    {
                        name: 'token',
                        label: 'Personal access token',
                        type: 'password',
                        placeholder: 'Enter your DigitalOcean personal access token',
                        required: true,
                    },
                ]
            case this.GOOGLE_CLOUD:
                return [
                    {
                        name: 'service_account_key',
                        label: 'Service account key (JSON)',
                        type: 'textarea',
                        placeholder: 'Paste your service account JSON key here',
                        required: true,
                    },
                ]
            case this.LEASEWEB:
                return [
                    {
                        name: 'api_key',
                        label: 'API Key',
                        type: 'password',
                        placeholder: 'Enter your LeaseWeb API key',
                        required: true,
                    },
                ]
            case this.LINODE:
                return [
                    {
                        name: 'token',
                        label: 'Personal access token',
                        type: 'password',
                        placeholder: 'Enter your linode personal access token',
                        required: true,
                    },
                ]
            case this.VULTR:
                return [
                    {
                        name: 'api_key',
                        label: 'API key',
                        type: 'password',
                        placeholder: 'Enter your vultr api key',
                        required: true,
                    },
                ]
            case this.OVH:
                return [
                    {
                        name: 'application_key',
                        label: 'Application Key',
                        type: 'text',
                        placeholder: 'Enter your ovh application key',
                        required: true,
                    },
                    {
                        name: 'application_secret',
                        label: 'Application Secret',
                        type: 'password',
                        placeholder: 'Enter your ovh application secret',
                        required: true,
                    },
                    {
                        name: 'consumer_key',
                        label: 'Consumer Key',
                        type: 'password',
                        placeholder: 'Enter your ovh consumer key',
                        required: true,
                    },
                ]
            default:
                throw new Error(`Unknown cloud provider type: ${type}`)
        }
    }

    static regions(type: CloudProviderType): RegionsByContinent {
        switch (type) {
            case this.AWS:
                return {
                    'North America': [
                        { name: 'US East (N. Virginia)', slug: 'us-east-1', flag: '/flags/us.svg', availableServerTypes: {} },
                        { name: 'US East (Ohio)', slug: 'us-east-2', flag: '/flags/us.svg', availableServerTypes: {} },
                        { name: 'US West (N. California)', slug: 'us-west-1', flag: '/flags/us.svg', availableServerTypes: {} },
                        { name: 'US West (Oregon)', slug: 'us-west-2', flag: '/flags/us.svg', availableServerTypes: {} },
                        { name: 'Canada (Central)', slug: 'ca-central-1', flag: '/flags/ca.svg', availableServerTypes: {} },
                        { name: 'Canada West (Calgary)', slug: 'ca-west-1', flag: '/flags/ca.svg', availableServerTypes: {} },
                        { name: 'Mexico (Central)', slug: 'mx-central-1', flag: '/flags/mx.svg', availableServerTypes: {} },
                    ],
                    'South America': [
                        { name: 'South America (São Paulo)', slug: 'sa-east-1', flag: '/flags/br.svg' },
                    ],
                    'Europe': [
                        { name: 'Europe (Ireland)', slug: 'eu-west-1', flag: '/flags/ie.svg' },
                        { name: 'Europe (London)', slug: 'eu-west-2', flag: '/flags/gb.svg' },
                        { name: 'Europe (Paris)', slug: 'eu-west-3', flag: '/flags/fr.svg' },
                        { name: 'Europe (Frankfurt)', slug: 'eu-central-1', flag: '/flags/de.svg' },
                        { name: 'Europe (Stockholm)', slug: 'eu-north-1', flag: '/flags/se.svg' },
                        { name: 'Europe (Milan)', slug: 'eu-south-1', flag: '/flags/it.svg' },
                        { name: 'Europe (Spain)', slug: 'eu-south-2', flag: '/flags/es.svg' },
                        { name: 'Europe (Zurich)', slug: 'eu-central-2', flag: '/flags/ch.svg' },
                    ],
                    'Asia Pacific': [
                        { name: 'Asia Pacific (Mumbai)', slug: 'ap-south-1', flag: '/flags/in.svg' },
                        { name: 'Asia Pacific (Hyderabad)', slug: 'ap-south-2', flag: '/flags/in.svg' },
                        { name: 'Asia Pacific (Singapore)', slug: 'ap-southeast-1', flag: '/flags/sg.svg' },
                        { name: 'Asia Pacific (Jakarta)', slug: 'ap-southeast-3', flag: '/flags/id.svg' },
                        { name: 'Asia Pacific (Sydney)', slug: 'ap-southeast-2', flag: '/flags/au.svg' },
                        { name: 'Asia Pacific (Melbourne)', slug: 'ap-southeast-4', flag: '/flags/au.svg' },
                        { name: 'Asia Pacific (Tokyo)', slug: 'ap-northeast-1', flag: '/flags/jp.svg' },
                        { name: 'Asia Pacific (Osaka)', slug: 'ap-northeast-3', flag: '/flags/jp.svg' },
                        { name: 'Asia Pacific (Seoul)', slug: 'ap-northeast-2', flag: '/flags/kr.svg' },
                        { name: 'Asia Pacific (Hong Kong)', slug: 'ap-east-1', flag: '/flags/hk.svg' },
                        { name: 'Asia Pacific (Taiwan)', slug: 'ap-northeast-4', flag: '/flags/tw.svg' },
                        { name: 'Asia Pacific (Thailand)', slug: 'ap-southeast-5', flag: '/flags/th.svg' },
                        { name: 'Asia Pacific (Malaysia)', slug: 'ap-southeast-6', flag: '/flags/my.svg' },
                    ],
                    'Middle East': [
                        { name: 'Middle East (Bahrain)', slug: 'me-south-1', flag: '/flags/bh.svg' },
                        { name: 'Middle East (UAE)', slug: 'me-central-1', flag: '/flags/ae.svg' },
                        { name: 'Israel (Tel Aviv)', slug: 'il-central-1', flag: '/flags/il.svg' },
                    ],
                    'Africa': [{ name: 'Africa (Cape Town)', slug: 'af-south-1', flag: '/flags/za.svg' }],
                }
            case this.HETZNER:
                return this.transformHetznerLocationsToRegions()
            case this.DIGITAL_OCEAN:
                return this.transformDigitalOceanRegionsToRegions()
            case this.GOOGLE_CLOUD:
                return {
                    'North America': [
                        { name: 'Iowa (us-central1)', slug: 'us-central1', flag: '/flags/us.svg' },
                        { name: 'South Carolina (us-east1)', slug: 'us-east1', flag: '/flags/us.svg' },
                        { name: 'N. Virginia (us-east4)', slug: 'us-east4', flag: '/flags/us.svg' },
                        { name: 'Columbus (us-east5)', slug: 'us-east5', flag: '/flags/us.svg' },
                        { name: 'Dallas (us-south1)', slug: 'us-south1', flag: '/flags/us.svg' },
                        { name: 'Oregon (us-west1)', slug: 'us-west1', flag: '/flags/us.svg' },
                        { name: 'Los Angeles (us-west2)', slug: 'us-west2', flag: '/flags/us.svg' },
                        { name: 'Salt Lake City (us-west3)', slug: 'us-west3', flag: '/flags/us.svg' },
                        { name: 'Las Vegas (us-west4)', slug: 'us-west4', flag: '/flags/us.svg' },
                        {
                            name: 'Montreal (northamerica-northeast1)',
                            slug: 'northamerica-northeast1',
                            flag: '/flags/ca.svg',
                        },
                        {
                            name: 'Toronto (northamerica-northeast2)',
                            slug: 'northamerica-northeast2',
                            flag: '/flags/ca.svg',
                        },
                    ],
                    'South America': [
                        {
                            name: 'São Paulo (southamerica-east1)',
                            slug: 'southamerica-east1',
                            flag: '/flags/br.svg',
                        },
                        {
                            name: 'Santiago (southamerica-west1)',
                            slug: 'southamerica-west1',
                            flag: '/flags/cl.svg',
                        },
                    ],
                    'Europe': [
                        { name: 'Belgium (europe-west1)', slug: 'europe-west1', flag: '/flags/be.svg' },
                        { name: 'London (europe-west2)', slug: 'europe-west2', flag: '/flags/gb.svg' },
                        { name: 'Frankfurt (europe-west3)', slug: 'europe-west3', flag: '/flags/de.svg' },
                        { name: 'Netherlands (europe-west4)', slug: 'europe-west4', flag: '/flags/nl.svg' },
                        { name: 'Zurich (europe-west6)', slug: 'europe-west6', flag: '/flags/ch.svg' },
                        { name: 'Milan (europe-west8)', slug: 'europe-west8', flag: '/flags/it.svg' },
                        { name: 'Paris (europe-west9)', slug: 'europe-west9', flag: '/flags/fr.svg' },
                        { name: 'Berlin (europe-west10)', slug: 'europe-west10', flag: '/flags/de.svg' },
                        { name: 'Turin (europe-west12)', slug: 'europe-west12', flag: '/flags/it.svg' },
                        { name: 'Finland (europe-north1)', slug: 'europe-north1', flag: '/flags/fi.svg' },
                        { name: 'Warsaw (europe-central2)', slug: 'europe-central2', flag: '/flags/pl.svg' },
                        {
                            name: 'Madrid (europe-southwest1)',
                            slug: 'europe-southwest1',
                            flag: '/flags/es.svg',
                        },
                    ],
                    'Asia Pacific': [
                        { name: 'Mumbai (asia-south1)', slug: 'asia-south1', flag: '/flags/in.svg' },
                        { name: 'Delhi (asia-south2)', slug: 'asia-south2', flag: '/flags/in.svg' },
                        { name: 'Singapore (asia-southeast1)', slug: 'asia-southeast1', flag: '/flags/sg.svg' },
                        { name: 'Jakarta (asia-southeast2)', slug: 'asia-southeast2', flag: '/flags/id.svg' },
                        { name: 'Hong Kong (asia-east2)', slug: 'asia-east2', flag: '/flags/hk.svg' },
                        { name: 'Taiwan (asia-east1)', slug: 'asia-east1', flag: '/flags/tw.svg' },
                        { name: 'Tokyo (asia-northeast1)', slug: 'asia-northeast1', flag: '/flags/jp.svg' },
                        { name: 'Osaka (asia-northeast2)', slug: 'asia-northeast2', flag: '/flags/jp.svg' },
                        { name: 'Seoul (asia-northeast3)', slug: 'asia-northeast3', flag: '/flags/kr.svg' },
                        {
                            name: 'Sydney (australia-southeast1)',
                            slug: 'australia-southeast1',
                            flag: '/flags/au.svg',
                        },
                        {
                            name: 'Melbourne (australia-southeast2)',
                            slug: 'australia-southeast2',
                            flag: '/flags/au.svg',
                        },
                    ],
                    'Middle East': [
                        { name: 'Tel Aviv (me-west1)', slug: 'me-west1', flag: '/flags/il.svg' },
                        { name: 'Doha (me-central1)', slug: 'me-central1', flag: '/flags/qa.svg' },
                        { name: 'Dammam (me-central2)', slug: 'me-central2', flag: '/flags/sa.svg' },
                    ],
                    'Africa': [
                        { name: 'Johannesburg (africa-south1)', slug: 'africa-south1', flag: '/flags/za.svg' },
                    ],
                }
            case this.VULTR:
                return {
                    'North America': [
                        { name: 'Atlanta, GA', slug: 'atl', flag: '/flags/us.svg' },
                        { name: 'Chicago, IL', slug: 'ord', flag: '/flags/us.svg' },
                        { name: 'Dallas, TX', slug: 'dfw', flag: '/flags/us.svg' },
                        { name: 'Honolulu, HI', slug: 'hnl', flag: '/flags/us.svg' },
                        { name: 'Los Angeles, CA', slug: 'lax', flag: '/flags/us.svg' },
                        { name: 'Miami, FL', slug: 'mia', flag: '/flags/us.svg' },
                        { name: 'New York Area', slug: 'ewr', flag: '/flags/us.svg' },
                        { name: 'San Francisco Bay Area, CA', slug: 'sjc', flag: '/flags/us.svg' },
                        { name: 'Seattle, WA', slug: 'sea', flag: '/flags/us.svg' },
                        { name: 'Toronto, Canada', slug: 'yto', flag: '/flags/ca.svg' },
                        { name: 'Mexico City, Mexico', slug: 'mex', flag: '/flags/mx.svg' },
                    ],
                    'South America': [
                        { name: 'São Paulo, Brazil', slug: 'sao', flag: '/flags/br.svg' },
                        { name: 'Santiago, Chile', slug: 'scl', flag: '/flags/cl.svg' },
                    ],
                    'Europe': [
                        { name: 'Amsterdam, Netherlands', slug: 'ams', flag: '/flags/nl.svg' },
                        { name: 'Frankfurt, Germany', slug: 'fra', flag: '/flags/de.svg' },
                        { name: 'London, United Kingdom', slug: 'lhr', flag: '/flags/gb.svg' },
                        { name: 'Madrid, Spain', slug: 'mad', flag: '/flags/es.svg' },
                        { name: 'Manchester, United Kingdom', slug: 'man', flag: '/flags/gb.svg' },
                        { name: 'Paris, France', slug: 'cdg', flag: '/flags/fr.svg' },
                        { name: 'Stockholm, Sweden', slug: 'arn', flag: '/flags/se.svg' },
                        { name: 'Warsaw, Poland', slug: 'waw', flag: '/flags/pl.svg' },
                    ],
                    'Asia Pacific': [
                        { name: 'Tokyo, Japan', slug: 'nrt', flag: '/flags/jp.svg' },
                        { name: 'Osaka, Japan', slug: 'itm', flag: '/flags/jp.svg' },
                        { name: 'Seoul, South Korea', slug: 'icn', flag: '/flags/kr.svg' },
                        { name: 'Singapore', slug: 'sgp', flag: '/flags/sg.svg' },
                        { name: 'Mumbai, India', slug: 'bom', flag: '/flags/in.svg' },
                        { name: 'Delhi NCR, India', slug: 'del', flag: '/flags/in.svg' },
                        { name: 'Bangalore, India', slug: 'blr', flag: '/flags/in.svg' },
                        { name: 'Sydney, Australia', slug: 'syd', flag: '/flags/au.svg' },
                        { name: 'Melbourne, Australia', slug: 'mel', flag: '/flags/au.svg' },
                    ],
                    'Middle East': [{ name: 'Tel Aviv-Yafo, Israel', slug: 'tlv', flag: '/flags/il.svg' }],
                    'Africa': [{ name: 'Johannesburg, South Africa', slug: 'jnb', flag: '/flags/za.svg' }],
                }
            case this.LINODE:
                return {
                    'North America': [
                        { name: 'Newark, NJ', slug: 'us-east', flag: '/flags/us.svg' },
                        { name: 'Atlanta, GA', slug: 'us-southeast', flag: '/flags/us.svg' },
                        { name: 'Dallas, TX', slug: 'us-central', flag: '/flags/us.svg' },
                        { name: 'Fremont, CA', slug: 'us-west', flag: '/flags/us.svg' },
                        { name: 'Chicago, IL', slug: 'us-ord', flag: '/flags/us.svg' },
                        { name: 'Los Angeles, CA', slug: 'us-lax', flag: '/flags/us.svg' },
                        { name: 'Miami, FL', slug: 'us-mia', flag: '/flags/us.svg' },
                        { name: 'Seattle, WA', slug: 'us-sea', flag: '/flags/us.svg' },
                        { name: 'Washington, D.C.', slug: 'us-iad', flag: '/flags/us.svg' },
                        { name: 'Toronto, Canada', slug: 'ca-central', flag: '/flags/ca.svg' },
                    ],
                    'South America': [{ name: 'São Paulo, Brazil', slug: 'br-gru', flag: '/flags/br.svg' }],
                    'Europe': [
                        { name: 'London, UK', slug: 'eu-west', flag: '/flags/gb.svg' },
                        { name: 'Frankfurt, Germany', slug: 'eu-central', flag: '/flags/de.svg' },
                        { name: 'Amsterdam, Netherlands', slug: 'nl-ams', flag: '/flags/nl.svg' },
                        { name: 'Stockholm, Sweden', slug: 'se-sto', flag: '/flags/se.svg' },
                        { name: 'Paris, France', slug: 'fr-par', flag: '/flags/fr.svg' },
                        { name: 'Milan, Italy', slug: 'it-mil', flag: '/flags/it.svg' },
                        { name: 'Madrid, Spain', slug: 'es-mad', flag: '/flags/es.svg' },
                    ],
                    'Asia Pacific': [
                        { name: 'Mumbai, India', slug: 'ap-west', flag: '/flags/in.svg' },
                        { name: 'Chennai, India', slug: 'in-maa', flag: '/flags/in.svg' },
                        { name: 'Singapore', slug: 'ap-south', flag: '/flags/sg.svg' },
                        { name: 'Tokyo, Japan', slug: 'ap-northeast', flag: '/flags/jp.svg' },
                        { name: 'Osaka, Japan', slug: 'jp-osa', flag: '/flags/jp.svg' },
                        { name: 'Jakarta, Indonesia', slug: 'id-cgk', flag: '/flags/id.svg' },
                        { name: 'Sydney, Australia', slug: 'ap-southeast', flag: '/flags/au.svg' },
                        { name: 'Melbourne, Australia', slug: 'au-mel', flag: '/flags/au.svg' },
                    ],
                }
            case this.LEASEWEB:
                return {
                    'North America': [
                        { name: 'Washington, D.C., USA', slug: 'wdc-02', flag: '/flags/us.svg' },
                        { name: 'San Francisco, CA, USA', slug: 'sfo-01', flag: '/flags/us.svg' },
                        { name: 'Montreal, Canada', slug: 'yul-01', flag: '/flags/ca.svg' },
                        { name: 'Miami, FL, USA', slug: 'mia-01', flag: '/flags/us.svg' },
                    ],
                    'Europe': [
                        { name: 'Amsterdam, Netherlands', slug: 'ams-01', flag: '/flags/nl.svg' },
                        { name: 'Frankfurt, Germany', slug: 'fra-01', flag: '/flags/de.svg' },
                        { name: 'London, United Kingdom', slug: 'lon-01', flag: '/flags/gb.svg' },
                    ],
                    'Asia Pacific': [
                        { name: 'Singapore', slug: 'sin-01', flag: '/flags/sg.svg' },
                        { name: 'Tokyo, Japan', slug: 'tyo-10', flag: '/flags/jp.svg' },
                    ],
                }
            case this.OVH:
                return {
                    'North America': [
                        { name: 'Beauharnois, Canada', slug: 'bhs', flag: '/flags/ca.svg' },
                        { name: 'Hillsboro, USA', slug: 'us-west-or-1', flag: '/flags/us.svg' },
                        { name: 'Vint Hill, USA', slug: 'us-east-va-1', flag: '/flags/us.svg' },
                    ],
                    'Europe': [
                        { name: 'Gravelines, France', slug: 'gra', flag: '/flags/fr.svg' },
                        { name: 'Strasbourg, France', slug: 'sbg', flag: '/flags/fr.svg' },
                        { name: 'Frankfurt, Germany', slug: 'de', flag: '/flags/de.svg' },
                        { name: 'London, United Kingdom', slug: 'uk', flag: '/flags/gb.svg' },
                        { name: 'Warsaw, Poland', slug: 'waw', flag: '/flags/pl.svg' },
                    ],
                    'Asia Pacific': [
                        { name: 'Singapore', slug: 'sgp', flag: '/flags/sg.svg' },
                        { name: 'Sydney, Australia', slug: 'syd', flag: '/flags/au.svg' },
                    ],
                }
            default:
                throw new Error(`Unknown cloud provider type: ${type}`)
        }
    }

    static regionServerTypes(type: CloudProviderType): RegionServerTypes {
        switch (type) {
            case this.HETZNER:
                return this.transformHetznerRegionServerTypes()
            case this.DIGITAL_OCEAN:
                return this.transformDigitalOceanRegionServerTypes()
            default:
                return {}
        }
    }

    static serverTypes(type: CloudProviderType): ServerTypes {
        switch (type) {
            case this.HETZNER:
                return this.transformHetznerServerTypes()
            case this.AWS:
                return {
                    't3.medium': { cpu: 2, ram: 4, disk: 20, name: 't3.medium - 2 vCPU, 4GB RAM, 20GB EBS' },
                    't3.large': { cpu: 2, ram: 8, disk: 20, name: 't3.large - 2 vCPU, 8GB RAM, 20GB EBS' },
                    't3.xlarge': {
                        cpu: 4,
                        ram: 16,
                        disk: 20,
                        name: 't3.xlarge - 4 vCPU, 16GB RAM, 20GB EBS',
                    },
                    't3.2xlarge': {
                        cpu: 8,
                        ram: 32,
                        disk: 20,
                        name: 't3.2xlarge - 8 vCPU, 32GB RAM, 20GB EBS',
                    },
                    'm5.large': { cpu: 2, ram: 8, disk: 20, name: 'm5.large - 2 vCPU, 8GB RAM, 20GB EBS' },
                    'm5.xlarge': {
                        cpu: 4,
                        ram: 16,
                        disk: 20,
                        name: 'm5.xlarge - 4 vCPU, 16GB RAM, 20GB EBS',
                    },
                    'm5.2xlarge': {
                        cpu: 8,
                        ram: 32,
                        disk: 20,
                        name: 'm5.2xlarge - 8 vCPU, 32GB RAM, 20GB EBS',
                    },
                    'm5.4xlarge': {
                        cpu: 16,
                        ram: 64,
                        disk: 20,
                        name: 'm5.4xlarge - 16 vCPU, 64GB RAM, 20GB EBS',
                    },
                }
            case this.DIGITAL_OCEAN:
                return this.transformDigitalOceanSizes()
            default:
                return {}
        }
    }

    static options(): CloudProviderOption[] {
        return this.values().map((value) => ({
            value,
            label: this.label(value),
        }))
    }

    static implemented(): CloudProviderType[] {
        return [this.HETZNER, this.DIGITAL_OCEAN]
    }

    static flatRegions(type: CloudProviderType): Region[] {
        const regionsByContinent = this.regions(type)
        const flatRegions: Region[] = []

        Object.values(regionsByContinent).forEach((regions) => {
            flatRegions.push(...regions)
        })

        return flatRegions
    }

    static getValidRegionSlugs(type: CloudProviderType): string[] {
        return this.flatRegions(type).map((region) => region.slug)
    }

    static getValidServerTypes(type: CloudProviderType): string[] {
        return Object.keys(this.serverTypes(type))
    }

    static getServerSpecs(type: CloudProviderType, serverType: string): ServerType | null {
        const serverTypes = this.serverTypes(type)
        return serverTypes[serverType] || null
    }

    static allProviders(): CloudProviderData[] {
        const implementedTypes = this.implemented()

        return this.values().map((type) => ({
            type,
            name: this.label(type),
            implemented: implementedTypes.includes(type),
            description: this.description(type),
            documentationLink: this.documentationLink(type),
            credentialFields: this.credentialFields(type),
        }))
    }

    static allRegions(): Record<CloudProviderType, RegionsByContinent> {
        const regions: Record<string, RegionsByContinent> = {}

        this.values().forEach((type) => {
            regions[type] = this.regions(type)
        })

        return regions as Record<CloudProviderType, RegionsByContinent>
    }

    static allServerTypes(): Record<CloudProviderType, ServerTypes> {
        const serverTypes: Record<string, ServerTypes> = {}

        this.values().forEach((type) => {
            serverTypes[type] = this.serverTypes(type)
        })

        return serverTypes as Record<CloudProviderType, ServerTypes>
    }
}
