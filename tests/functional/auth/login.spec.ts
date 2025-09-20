import { test } from '@japa/runner'
import { createRegisteredUser } from '#tests/helpers/test_helpers'

test.group('@login', () => {
  test('shows login form', async ({ client }) => {
    const response = await client.get('/auth/login')
    response.assertStatus(200)
  })

  test('successfully logs in with valid credentials and is redirected to /w', async ({ client }) => {
    const { user } = await createRegisteredUser()

    const response = await client
      .post('/auth/login')
      .form({ email: user.email, password: 'testpassword123' })
      .withCsrfToken()
      .redirects(0)

    response.assertStatus(302)
    response.assertHeader('location', '/w')
  })

  test('rejects invalid credentials and redirects back to /auth/login', async ({ client }) => {
    const { user } = await createRegisteredUser()

    const response = await client
      .post('/auth/login')
      .form({ email: user.email, password: 'wrongpassword' })
      .withCsrfToken()
      .redirects(0)

    response.assertStatus(302)
    response.assertHeader('location', '/auth/login')
  })
})

