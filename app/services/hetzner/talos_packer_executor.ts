import { ChildProcess } from '#utils/child_process'
import logger from '@adonisjs/core/services/logger'

/**
 * Minimal executor to run Packer against the generated Hetzner Talos template.
 *
 * It will run, in the working directory provided:
 *  - export HCLOUD_TOKEN=${TOKEN}
 *  - packer init .
 *  - packer build .
 *
 * For now, we ignore outputs and only stream logs to the app logger.
 */
export class TalosPackerExecutor {
  constructor(
    private workingDir: string,
    private token: string
  ) {}

  async run() {
    await new ChildProcess()
      .command('packer')
      .args(['init', '.'])
      .cwd(this.workingDir)
      .env({ HCLOUD_TOKEN: this.token })
      .onStdout((d) => logger.info(`[packer:init] ${d.trim()}`))
      .onStderr((d) => logger.error(`[packer:init] ${d.trim()}`))
      .onClose((code) => logger.info(`[packer:init] exited with code ${code}`))
      .onError((err) => logger.error(`[packer:init] error: ${err.message}`))
      .execute()

    // packer build .
    await new ChildProcess()
      .command('packer')
      .args(['build', '.'])
      .cwd(this.workingDir)
      .env({ HCLOUD_TOKEN: this.token })
      .onStdout((d) => logger.info(`[packer:build] ${d.trim()}`))
      .onStderr((d) => logger.error(`[packer:build] ${d.trim()}`))
      .onClose((code) => logger.info(`[packer:build] exited with code ${code}`))
      .onError((err) => logger.error(`[packer:build] error: ${err.message}`))
      .execute()
  }
}
