import Project from '#models/project'
import { test } from '@japa/runner'

test.group('Projects create', () => {
  test('example test', async ({ assert }) => {
    const project = await Project.create({})

    console.log({ project })
  })
})
