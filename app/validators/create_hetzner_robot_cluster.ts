import vine from '@vinejs/vine'
import { FieldContext } from '@vinejs/vine/types'
import db from '@adonisjs/lucid/services/db'
import { cache } from '#services/cache/cache'
import CloudProvider from '#models/cloud_provider'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'

async function validateCloudProvider(value: unknown, _options: undefined, field: FieldContext) {
  if (!value) {
    return
  }

  const cloudProvider = await db
    .from('cloud_providers')
    .where('id', value as string)
    .first()

  if (!cloudProvider) {
    field.report('The selected cloud provider does not exist', 'exists', field)
  }
}

async function validateRobotCloudProvider(
  value: unknown,
  _options: undefined,
  field: FieldContext
) {
  if (!value) {
    return
  }

  const cloudProvider = await db
    .from('cloud_providers')
    .where('id', value as string)
    .first()

  if (!cloudProvider) {
    field.report('The selected cloud provider does not exist', 'exists', field)
    return
  }

  if (cloudProvider.type !== CloudProviderDefinitions.HETZNER) {
    field.report(
      'The cloud provider must be of type Hetzner (not Hetzner Robot)',
      'invalid_provider_type',
      field
    )
    return
  }
}

async function validateRobotServerNumbers(
  value: unknown,
  _options: undefined,
  field: FieldContext
) {
  if (!Array.isArray(value)) {
    field.report('The {{ field }} field must be an array', 'array', field)
    return
  }

  if (value.length < 3) {
    field.report('You must select at least 3 servers', 'min_servers', field)
    return
  }

  // Get cloud provider id from the request data
  const cloudProviderId = (field.data as any).cloud_provider_id
  if (!cloudProviderId) {
    return
  }

  const cloudProvider = await CloudProvider.find(cloudProviderId)
  if (!cloudProvider) {
    return
  }

  // Check cache for servers
  const cacheKey = `provider:${cloudProvider.id}`
  const cachedServers = (await cache('hetzner-robot').item(cacheKey).read()) as Array<{
    server_number: number
  }> | null

  if (!cachedServers) {
    field.report('Server data not found in cache. Please refresh the page.', 'cache_miss', field)
    return
  }

  // Validate all server numbers exist in cache
  const serverNumbers = cachedServers.map((s) => s.server_number)
  for (const serverNumber of value) {
    if (!serverNumbers.includes(serverNumber)) {
      field.report(
        `Server ${serverNumber} not found in your Hetzner Robot account`,
        'invalid_server',
        field
      )
      return
    }
  }
}

async function validateRobotVSwitchId(value: unknown, _options: undefined, field: FieldContext) {
  if (!value) {
    field.report('The {{ field }} field is required', 'required', field)
    return
  }

  // Allow "create_new" as a special value
  if (value === 'create_new') {
    return
  }

  // Get cloud provider id from the request data
  const cloudProviderId = (field.data as any).cloud_provider_id
  if (!cloudProviderId) {
    return
  }

  const cloudProvider = await CloudProvider.find(cloudProviderId)
  if (!cloudProvider) {
    return
  }

  // Check cache for vswitches
  const cacheKey = `vswitches:provider:${cloudProvider.id}`
  const cachedVswitches = (await cache('hetzner-robot').item(cacheKey).read()) as Array<{
    id: number
  }> | null

  if (!cachedVswitches) {
    field.report('vSwitch data not found in cache. Please refresh the page.', 'cache_miss', field)
    return
  }

  // Validate vswitch exists in cache
  const vswitchIds = cachedVswitches.map((v) => v.id)
  if (!vswitchIds.includes(Number(value))) {
    field.report(
      'The selected vSwitch does not exist in your Hetzner Robot account',
      'invalid_vswitch',
      field
    )
    return
  }
}

export const cloudProviderExistsRule = vine.createRule(validateCloudProvider)
export const robotCloudProviderRule = vine.createRule(validateRobotCloudProvider)
export const robotServerNumbersRule = vine.createRule(validateRobotServerNumbers)
export const robotVSwitchIdRule = vine.createRule(validateRobotVSwitchId)

export const createHetznerRobotClusterValidator = vine.compile(
  vine.object({
    subdomain_identifier: vine
      .string()
      .regex(
        /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$/
      )
      .trim()
      .minLength(1)
      .maxLength(255),
    cloud_provider_id: vine.string().uuid().use(cloudProviderExistsRule()),
    robot_cloud_provider_id: vine.string().uuid().use(robotCloudProviderRule()),
    region: vine.string().trim().minLength(1).maxLength(255),
    robot_server_numbers: vine.array(vine.number()).use(robotServerNumbersRule()),
    robot_vswitch_id: vine.number().use(robotVSwitchIdRule()).optional(),
  })
)
