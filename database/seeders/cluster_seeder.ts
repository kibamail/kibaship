import Cluster from '#models/cluster'
import { BaseSeeder } from '@adonisjs/lucid/seeders'

export default class extends BaseSeeder {
  async run() {
    await Cluster.firstOrCreate(
      {
        subdomainIdentifier: 'staging.kibaship.app',
      },
      {
        controlPlaneEndpoint: 'https://kube.staging.kibaship.com',
        location: 'eu-central-helsinki',
        subdomainIdentifier: 'staging.kibaship.app',
      }
    )
  }
}
