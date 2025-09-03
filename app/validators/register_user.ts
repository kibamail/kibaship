import vine from '@vinejs/vine'
import { FieldContext } from '@vinejs/vine/types'
import User from '#models/user'

const emailUniqueRule = vine.createRule(async (value: unknown, _: any, field: FieldContext) => {
  if (typeof value !== 'string') {
    return
  }

  const user = await User.query().where('email', value).first()

  if (user) {
    field.report('The {{ field }} is already taken', 'unique', field)
  }
})

export const registerUserValidator = vine.compile(
  vine.object({
    email: vine
      .string()
      .email()
      .use(emailUniqueRule()),
    password: vine
      .string()
      .minLength(8)
  })
)