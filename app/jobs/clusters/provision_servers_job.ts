import { Job } from '@rlanz/bull-queue'

interface ProvisionServersJobPayload {}

export default class ProvisionServersJob extends Job {
  // This is the path to the file that is used to create the job
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionServersJobPayload) {}

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(payload: ProvisionServersJobPayload) {}
}