import { GithubConfig } from '#config/github'
import app from '@adonisjs/core/services/app'
import { DateTime } from 'luxon'
import { App } from 'octokit'

export class GithubService {
  public app: App
  public config: GithubConfig

  constructor() {
    this.config = app.config.get('github')

    this.app = new App({
      appId: this.config.app.id,
      privateKey: this.config.app.privateKey,
    })
  }

  public installation(installationId: number) {
    return new GithubInstallationService(this, installationId)
  }
}

class GithubInstallationService {
  constructor(
    protected githubService: GithubService,
    protected installationId: number
  ) {}

  protected octokit() {
    return this.githubService.app.getInstallationOctokit(this.installationId)
  }

  public async installation() {
    const octokit = await this.octokit()

    const { data } = await octokit.request(`GET /app/installations/{installation_id}`, {
      installation_id: this.installationId,
    })

    return data
  }

  public async repositories() {
    const octokit = await this.octokit()

    const { data: repositories } = await octokit.request('GET /installation/repositories')

    return repositories.repositories.map((repo) => ({
      repository: repo.name,
      visibility: repo.visibility as 'public' | 'private',
      lastUpdatedAt: repo.updated_at ? DateTime.fromJSDate(new Date(repo.updated_at)) : null,
    }))
  }
}
