import { OauthConfig } from '#config/oauth'
import User from '#models/user'
import { OauthService } from '#services/auth/oauth_service'
import app from '@adonisjs/core/services/app'
import redis from '@adonisjs/redis/services/main'
import encryption from '@adonisjs/core/services/encryption'
import { test } from '@japa/runner'
import { randomBytes } from 'node:crypto'

test.group('@auth', () => {
  test('can successfully redirect a user to authentication app', async ({ client, assert }) => {
    const response = await client.get('/auth/redirect').redirects(0)

    response.assertStatus(302)

    const redirectTo = response.header('location')

    const oauthConfig = app.config.get<OauthConfig>('oauth')

    assert.include(redirectTo, oauthConfig.oauth.clientBaseUrl)
    assert.include(redirectTo, oauthConfig.oauth.clientId)
  })

  test('auth callback redirects to home page is callback code is not present', async ({
    client,
    assert,
  }) => {
    const response = await client.get('/auth/callback').redirects(0)

    response.assertStatus(302)

    const redirectTo = response.header('location')

    assert.include(redirectTo, '/')
  })

  test('auth callback redirects to home page if failed to get a valid access token', async ({
    client,
    assert,
  }) => {
    class FakeOauthService extends OauthService {
      api() {
        return {
          auth() {
            return {
              accessToken() {
                return [
                  null,
                  {
                    cause: new Error('Invalid access token.'),
                  },
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['api']>
      }
    }

    app.container.swap(OauthService, function () {
      return new FakeOauthService()
    })
    const response = await client.get('/auth/callback?code=1234567890').redirects(0)

    response.assertStatus(302)

    const redirectTo = response.header('location')

    assert.include(
      redirectTo,
      `/?error=${encodeURIComponent('Failed to authenticate. Please try again.')}`
    )

    app.container.restoreAll()
  })

  test('auth callback redirects to home page if failed to get user profile from access token', async ({
    client,
    assert,
  }) => {
    class FakeOauthService extends OauthService {
      api() {
        return {
          auth() {
            return {
              accessToken() {
                return [
                  {
                    access_token: randomBytes(64),
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['api']>
      }
      accessToken() {
        return {
          user() {
            return {
              profile() {
                return [
                  null,
                  {
                    cause: new Error('Failed to fetch profile.'),
                  },
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['accessToken']>
      }
    }

    app.container.swap(OauthService, function () {
      return new FakeOauthService()
    })

    const response = await client.get('/auth/callback?code=1234567890').redirects(0)

    response.assertStatus(302)

    const redirectTo = response.header('location')

    assert.include(
      redirectTo,
      `/?error=${encodeURIComponent('Failed to get your profile information. Please try again.')}`
    )

    app.container.restoreAll()
  })

  test('auth callback successfully logs in existing user and redirects to workspace', async ({
    client,
    assert,
  }) => {
    const uniqueOauthId = `oauth_${randomBytes(8).toString('hex')}`
    const uniqueEmail = `updated_${randomBytes(4).toString('hex')}@example.com`
    const existingUser = await User.create({
      email: `old_${randomBytes(4).toString('hex')}@example.com`,
      oauthId: uniqueOauthId,
    })

    class FakeOauthService extends OauthService {
      api() {
        return {
          auth() {
            return {
              accessToken() {
                return [
                  {
                    access_token: 'valid_token_123',
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['api']>
      }

      accessToken() {
        return {
          user() {
            return {
              profile() {
                return [
                  {
                    id: uniqueOauthId,
                    email: uniqueEmail,
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['accessToken']>
      }
    }

    app.container.swap(OauthService, function () {
      return new FakeOauthService()
    })

    const response = await client.get('/auth/callback?code=1234567890').redirects(0)

    response.assertStatus(302)
    const redirectTo = response.header('location')
    assert.equal(redirectTo, '/w')

    await existingUser.refresh()
    assert.equal(existingUser.email, uniqueEmail)

    app.container.restoreAll()
  })

  test('auth callback successfully creates new user, caches profile, creates workspace and redirects', async ({
    client,
    assert,
  }) => {
    const newOauthId = `new_oauth_${randomBytes(8).toString('hex')}`
    const newEmail = `new_${randomBytes(4).toString('hex')}@example.com`

    class FakeOauthService extends OauthService {
      api() {
        return {
          auth() {
            return {
              accessToken() {
                return [
                  {
                    access_token: 'new_user_token_123',
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['api']>
      }

      accessToken() {
        return {
          user() {
            return {
              profile() {
                return [
                  {
                    id: newOauthId,
                    email: newEmail,
                  },
                  null,
                ]
              },
            }
          },
          workspaces() {
            return {
              create() {
                return [
                  {
                    id: 'workspace123',
                    name: `${newEmail.split('@')[0]} ${newEmail.split('@')[1]}'s Workspace`,
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['accessToken']>
      }
    }

    app.container.swap(OauthService, function () {
      return new FakeOauthService()
    })

    const response = await client.get('/auth/callback?code=1234567890').redirects(0)

    response.assertStatus(302)
    const redirectTo = response.header('location')
    assert.equal(redirectTo, '/w')

    const createdUser = await User.findBy('oauthId', newOauthId)
    assert.isNotNull(createdUser)
    assert.equal(createdUser!.email, newEmail)

    const cachedProfile = await redis.get(`users:${createdUser!.id}`)
    assert.isNotNull(cachedProfile)
    const parsedProfile = JSON.parse(cachedProfile!)
    assert.equal(parsedProfile.id, newOauthId)
    assert.equal(parsedProfile.email, newEmail)

    const cachedToken = await redis.get(`oauth_tokens:${createdUser!.id}`)
    assert.isNotNull(cachedToken)
    const decryptedToken = encryption.decrypt(cachedToken!)
    assert.equal(decryptedToken, 'new_user_token_123')

    app.container.restoreAll()
  })

  test('auth callback handles workspace creation failure for new user', async ({
    client,
    assert,
  }) => {
    const newOauthId = `workspace_fail_${randomBytes(8).toString('hex')}`
    const newEmail = `workspace_fail_${randomBytes(4).toString('hex')}@example.com`

    class FakeOauthService extends OauthService {
      api() {
        return {
          auth() {
            return {
              accessToken() {
                return [
                  {
                    access_token: 'workspace_fail_token',
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['api']>
      }

      accessToken() {
        return {
          user() {
            return {
              profile() {
                return [
                  {
                    id: newOauthId,
                    email: newEmail,
                  },
                  null,
                ]
              },
            }
          },
          workspaces() {
            return {
              create() {
                return [
                  null,
                  {
                    cause: new Error('Workspace creation failed'),
                  },
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['accessToken']>
      }
    }

    app.container.swap(OauthService, function () {
      return new FakeOauthService()
    })

    const response = await client.get('/auth/callback?code=1234567890').redirects(0)

    response.assertStatus(302)
    const redirectTo = response.header('location')
    assert.equal(redirectTo, '/')

    const createdUser = await User.findBy('oauthId', newOauthId)
    assert.isNotNull(createdUser)
    assert.equal(createdUser!.email, newEmail)

    const cachedProfile = await redis.get(`users:${createdUser!.id}`)
    assert.isNotNull(cachedProfile)
    const parsedProfile = JSON.parse(cachedProfile!)
    assert.equal(parsedProfile.id, newOauthId)
    assert.equal(parsedProfile.email, newEmail)

    const cachedToken = await redis.get(`oauth_tokens:${createdUser!.id}`)
    assert.isNotNull(cachedToken)
    const decryptedToken = encryption.decrypt(cachedToken!)
    assert.equal(decryptedToken, 'workspace_fail_token')

    app.container.restoreAll()
  })

  test('auth callback handles second profile fetch failure after workspace creation', async ({
    client,
    assert,
  }) => {
    const newOauthId = `profile_fail_${randomBytes(8).toString('hex')}`
    const newEmail = `profile_fail_${randomBytes(4).toString('hex')}@example.com`
    let profileCallCount = 0

    class FakeOauthService extends OauthService {
      api() {
        return {
          auth() {
            return {
              accessToken() {
                return [
                  {
                    access_token: 'profile_fail_token',
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['api']>
      }

      accessToken() {
        return {
          user() {
            return {
              profile() {
                profileCallCount++
                if (profileCallCount === 1) {
                  return [
                    {
                      id: newOauthId,
                      email: newEmail,
                    },
                    null,
                  ]
                } else {
                  return [
                    null,
                    {
                      cause: new Error('Second profile fetch failed'),
                    },
                  ]
                }
              },
            }
          },
          workspaces() {
            return {
              create() {
                return [
                  {
                    id: 'workspace123',
                    name: `${newEmail.split('@')[0]} ${newEmail.split('@')[1]}'s Workspace`,
                  },
                  null,
                ]
              },
            }
          },
        } as unknown as ReturnType<OauthService['accessToken']>
      }
    }

    app.container.swap(OauthService, function () {
      return new FakeOauthService()
    })

    const response = await client.get('/auth/callback?code=1234567890').redirects(0)

    response.assertStatus(302)
    const redirectTo = response.header('location')
    assert.equal(redirectTo, '/')

    const createdUser = await User.findBy('oauthId', newOauthId)
    assert.isNotNull(createdUser)
    assert.equal(createdUser!.email, newEmail)

    const cachedProfile = await redis.get(`users:${createdUser!.id}`)
    assert.isNotNull(cachedProfile)
    const parsedProfile = JSON.parse(cachedProfile!)
    assert.equal(parsedProfile.id, newOauthId)
    assert.equal(parsedProfile.email, newEmail)

    const cachedToken = await redis.get(`oauth_tokens:${createdUser!.id}`)
    assert.isNotNull(cachedToken)
    const decryptedToken = encryption.decrypt(cachedToken!)
    assert.equal(decryptedToken, 'profile_fail_token')

    app.container.restoreAll()
  })
})
