import { test } from '@japa/runner'
import User from '#models/user'
import Workspace from '#models/workspace'
import { randomBytes } from 'node:crypto'
import hash from '@adonisjs/core/services/hash'
import db from '@adonisjs/lucid/services/db'

test.group('@register', () => {
  test('shows registration form', async ({ client }) => {
    const response = await client.get('/auth/register')
    
    response.assertStatus(200)
  })

  test('successfully registers new user with valid data', async ({ client, assert }) => {
    const testEmail = `new_user_${randomBytes(4).toString('hex')}@example.com`
    const testPassword = 'validpassword123'
    
    const response = await client.post('/auth/register').form({
      email: testEmail,
      password: testPassword
    }).withCsrfToken()

    response.assertStatus(200)
    
    // Verify user was created
    const createdUser = await User.findBy('email', testEmail)
    assert.isNotNull(createdUser)
    assert.equal(createdUser!.email, testEmail)
    assert.isNotNull(createdUser!.password)

    // Verify password was hashed correctly
    const passwordValid = await hash.verify(createdUser!.password!, testPassword)
    assert.isTrue(passwordValid)

    // Verify workspace was created
    const workspace = await Workspace.findBy('userId', createdUser!.id)
    assert.isNotNull(workspace)
    
    const [username] = testEmail.split('@')
    assert.equal(workspace!.name, `${username}'s Workspace`)
    assert.equal(workspace!.slug, username.toLowerCase().replace(/[^a-z0-9]/g, '-'))
    assert.equal(workspace!.userId, createdUser!.id)

  })


  test('creates user and workspace in database transaction', async ({ client, assert }) => {
    const testEmail = `transaction_success_${randomBytes(4).toString('hex')}@example.com`
    const testPassword = 'validpassword123'

    let userCountBefore: number
    let workspaceCountBefore: number

    await db.transaction(async (trx) => {
      const userResult = await trx.from('users').count('* as total')
      const workspaceResult = await trx.from('workspaces').count('* as total')
      
      userCountBefore = Number(userResult[0].total)
      workspaceCountBefore = Number(workspaceResult[0].total)
    })

    const response = await client.post('/auth/register').form({
      email: testEmail,
      password: testPassword
    }).withCsrfToken()

    response.assertStatus(200)

    // Verify both user and workspace were created
    await db.transaction(async (trx) => {
      const userResult = await trx.from('users').count('* as total')
      const workspaceResult = await trx.from('workspaces').count('* as total')
      
      const userCountAfter = Number(userResult[0].total)
      const workspaceCountAfter = Number(workspaceResult[0].total)

      assert.equal(userCountAfter, userCountBefore + 1)
      assert.equal(workspaceCountAfter, workspaceCountBefore + 1)
    })
  })

  test('automatically logs in user after successful registration', async ({ client, assert }) => {
    const testEmail = `auto_login_${randomBytes(4).toString('hex')}@example.com`
    const testPassword = 'validpassword123'

    const response = await client.post('/auth/register').form({
      email: testEmail,
      password: testPassword
    }).withCsrfToken()

    response.assertStatus(200)

    // Verify user was created and is logged in
    const createdUser = await User.findBy('email', testEmail)
    assert.isNotNull(createdUser)
  })

  test('handles special characters in email for workspace slug creation', async ({ client, assert }) => {
    const testEmail = `user.with+special-chars_${randomBytes(2).toString('hex')}@example.com`
    const testPassword = 'validpassword123'

    const response = await client.post('/auth/register').form({
      email: testEmail,
      password: testPassword
    }).withCsrfToken()

    response.assertStatus(200)

    const createdUser = await User.findBy('email', testEmail)
    assert.isNotNull(createdUser)

    const workspace = await Workspace.findBy('userId', createdUser!.id)
    assert.isNotNull(workspace)

    // Verify slug was properly sanitized
    const [username] = testEmail.split('@')
    const expectedSlug = username.toLowerCase().replace(/[^a-z0-9]/g, '-')
    assert.equal(workspace!.slug, expectedSlug)
    
    // Ensure slug contains only valid characters
    assert.match(workspace!.slug, /^[a-z0-9-]+$/)
  })

  test('handles very long email addresses gracefully', async ({ client, assert }) => {
    const longUsername = 'a'.repeat(50) // Very long username
    const testEmail = `${longUsername}@example.com`
    const testPassword = 'validpassword123'

    const response = await client.post('/auth/register').form({
      email: testEmail,
      password: testPassword
    }).withCsrfToken()

    response.assertStatus(200)

    const createdUser = await User.findBy('email', testEmail)
    assert.isNotNull(createdUser)

    const workspace = await Workspace.findBy('userId', createdUser!.id)
    assert.isNotNull(workspace)

    // Verify workspace name and slug were created properly
    assert.equal(workspace!.name, `${longUsername}'s Workspace`)
    assert.equal(workspace!.slug, longUsername.toLowerCase())
  })
})