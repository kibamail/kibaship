import vine from '@vinejs/vine'

export const createByocClusterValidator = vine.compile(
  vine.object({
    location: vine.string().trim().minLength(1),
    talosConfig: vine.object({
      ca: vine.string().trim().minLength(1),
      crt: vine.string().trim().minLength(1),
      key: vine.string().trim().minLength(1),
    }),
  })
)

