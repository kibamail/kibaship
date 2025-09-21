import { GithubConfig } from '#config/github'
import { BaseController } from '#controllers/Base/base_controller'
import SourceCodeProvider from '#models/source_code_provider'
import SourceCodeRepository from '#models/source_code_repository'
import { GithubService } from '#services/source-providers/github/github_service'
import { inject } from '@adonisjs/core'
import type { HttpContext } from '@adonisjs/core/http'
import app from '@adonisjs/core/services/app'

@inject()
export default class SourceProvidersController extends BaseController {
  constructor(protected githubService: GithubService) {
    super()
  }

  public async index(ctx: HttpContext) {
    const workspace = await this.workspace(ctx)

    const sourceCodeProviders = await SourceCodeProvider.query().where('workspace_id', workspace.id)

    return ctx.response.json(sourceCodeProviders)
  }

  public async show(ctx: HttpContext) {
    const repositories = await SourceCodeRepository.query()
      .where('sourceCodeProviderId', ctx.params.sourceCodeProviderId)
      .orderBy('lastUpdatedAt', 'desc')

    return ctx.response.json(repositories)
  }

  public async redirect(ctx: HttpContext) {
    const github = app.config.get<GithubConfig>('github')

    return ctx.response.redirect(`https://github.com/apps/${github.app.name}/installations/new`)
  }

  public async callback(ctx: HttpContext) {
    const query = ctx.request.qs()
    const workspace = await this.workspace(ctx)

    const installation = await this.githubService.installation(query.installation_id).installation()

    const sourceCodeProvider = await SourceCodeProvider.firstOrCreate(
      {
        providerId: installation.id.toString(),
        workspaceId: workspace.id,
      },
      {
        workspaceId: workspace.id,
        providerId: installation.id.toString(),
        name: (installation.account as { login: string })?.login,
        avatar: installation.account?.avatar_url,
        type:
          (installation?.account as { type: string })?.type === 'User' ? 'user' : 'organization',
      }
    )

    const repositories = await this.githubService.installation(query.installation_id).repositories()

    await SourceCodeRepository.createMany([
      ...repositories.map((repository) => ({
        ...repository,
        sourceCodeProviderId: sourceCodeProvider.id,
      })),
    ])

    return ctx.response.redirect(`/w/${workspace.slug}`)
  }
}
