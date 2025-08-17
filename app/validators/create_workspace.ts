import vine from '@vinejs/vine'

export const createWorkspaceValidator = vine.compile(
  vine.object({
    name: vine.string().alphaNumeric().minLength(2).maxLength(24),
  })
)
