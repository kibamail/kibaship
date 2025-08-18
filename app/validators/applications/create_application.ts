import vine from '@vinejs/vine'

export const createApplicationValidator = vine.compile(
  vine.object({
    type: vine.enum(['mysql', 'postgres', 'git', 'docker_image']),
    gitConfiguration: vine
      .object({
        sourceCodeRepositoryId: vine.string().uuid(),
      })
      .optional()
      .requiredWhen('type', '=', 'git'),
    dockerImageConfiguration: vine
      .object({
        image: vine.string().url(),
      })
      .optional()
      .requiredWhen('type', '=', 'docker_image'),
    projectId: vine.string().uuid().optional(),
  })
)
