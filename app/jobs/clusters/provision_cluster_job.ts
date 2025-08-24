import { Job } from '@rlanz/bull-queue'

interface ProvisionClusterJobPayload { }

export default class ProvisionClusterJob extends Job {
  // This is the path to the file that is used to create the job
  static get $$filepath() {
    return import.meta.url
  }

  /**
   * Base Entry point
   */
  async handle(payload: ProvisionClusterJobPayload) {
    // compile terraform templates for cluster
    // execute terraform apply and store terraform state remotely (this will be step by step, to provide visibility into the entire process)
    // the steps are:
    // 1. provision ssh keys
    // 2. provision private networking
    // 3. provision load balancers
    // 4. provision servers
    // 5. provision volumes for storage
    // next, run terraform commands to provision k8s cluster
    // this will execute a set of ansible scripts on top of the provisioned cluster once ready
    // next, run terraform commands on top of ready cluster to provision kibaship ready cluster

    // this will setup linstor, cert manager, ingress, policies, service accounts, etc and of course, the kibaship operator
    // 
  }

  /**
   * This is an optional method that gets called when the retries has exceeded and is marked failed.
   */
  async rescue(payload: ProvisionClusterJobPayload) { }
}