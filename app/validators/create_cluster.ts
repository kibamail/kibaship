import vine from '@vinejs/vine'
import { FieldContext } from '@vinejs/vine/types'
import db from '@adonisjs/lucid/services/db'

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

function validateControlPlaneNodes(value: unknown, _options: undefined, field: FieldContext) {
  if (typeof value !== 'number') {
    field.report('The {{ field }} field must be a number', 'number', field)
    return
  }

  const allowedValues = [1, 3, 5]
  if (!allowedValues.includes(value)) {
    field.report('The {{ field }} field must be 1, 3, or 5', 'invalid_control_plane_count', field)
    return
  }
}

export const cloudProviderExistsRule = vine.createRule(validateCloudProvider)
export const controlPlaneNodesRule = vine.createRule(validateControlPlaneNodes)

export const createClusterValidator = vine.compile(
  vine.object({
    subdomain_identifier: vine.string().regex(
      /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$/
    ).trim().minLength(1).maxLength(255),
    cloud_provider_id: vine
      .string()
      .uuid()
      .use(cloudProviderExistsRule()),
    region: vine.string().trim().minLength(1),
    control_plane_nodes_count: vine
      .number()
      .use(controlPlaneNodesRule()),
    worker_nodes_count: vine
      .number()
      .min(1)
      .max(100),
    server_type: vine.string().trim().minLength(1),
    control_planes_volume_size: vine
      .number()
      .min(10)
      .max(500),
    workers_volume_size: vine
      .number()
      .min(10)
      .max(500),
  })
)
