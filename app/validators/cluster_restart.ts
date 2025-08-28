import vine from '@vinejs/vine'

export const clusterRestartValidator = vine.compile(
  vine.object({
    type: vine.enum(['start', 'failed'])
  })
)
