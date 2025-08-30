import { DateTime } from 'luxon'
import { BaseModel, beforeCreate, column } from '@adonisjs/lucid/orm'
import { randomUUID } from 'node:crypto'
import encryption from '@adonisjs/core/services/encryption'

export default class CloudProvider extends BaseModel {
  @column({ isPrimary: true })
  declare id: string

  @column()
  declare name: string

  @column()
  declare type: 'aws' | 'hetzner' | 'google_cloud' | 'digital_ocean' | 'leaseweb' | 'linode' | 'vultr' | 'ovh'

  @column()
  declare workspaceId: string

  @column({
    prepare: value => encryption.encrypt(JSON.stringify(value)),
    consume: value => JSON.parse(encryption.decrypt(value) || '{}'),
    serialize: __dirnamevalue => null,
  })
  declare credentials: Partial<{
    // Common fields
    token: string
    api_key: string

    // AWS specific
    access_key_id: string
    secret_access_key: string

    // Google Cloud specific
    project_id: string
    service_account_key: string

    // OVH specific
    endpoint: string
    application_key: string
    application_secret: string
    consumer_key: string

    access_token: string
    refresh_token: string
  }>

  @column.dateTime({ autoCreate: true })
  declare createdAt: DateTime

  @column.dateTime({ autoCreate: true, autoUpdate: true })
  declare updatedAt: DateTime

  public credentialsPath() {
    return `${this.workspaceId}/${this.type}/${this.id}`
  }

  /**
   * Get credentials formatted for Terraform variables
   */
  public getTerraformCredentials(): Record<string, string> {
    switch (this.type) {
      case 'hetzner':
        return {
          hcloud_token: this.credentials.token || ''
        }
      case 'aws':
        return {
          aws_access_key_id: this.credentials.access_key_id || '',
          aws_secret_access_key: this.credentials.secret_access_key || ''
        }
      case 'digital_ocean':
        return {
          do_token: this.credentials.access_token || ''
        }
      case 'linode':
        return {
          linode_token: this.credentials.token || ''
        }
      case 'vultr':
        return {
          vultr_api_key: this.credentials.api_key || ''
        }
      case 'google_cloud':
        return {
          gcp_project_id: this.credentials.project_id || '',
          gcp_service_account_key: this.credentials.service_account_key || ''
        }
      case 'leaseweb':
        return {
          leaseweb_api_key: this.credentials.api_key || ''
        }
      case 'ovh':
        return {
          ovh_endpoint: this.credentials.endpoint || '',
          ovh_application_key: this.credentials.application_key || '',
          ovh_application_secret: this.credentials.application_secret || '',
          ovh_consumer_key: this.credentials.consumer_key || ''
        }
      default:
        return {}
    }
  }

  /**
   * Get network zone for a given location
   */
  public getNetworkZone(location: string): string {
    switch (this.type) {
      case 'hetzner':
        return this.getHetznerNetworkZone(location)
      case 'aws':
        return location
      case 'digital_ocean':
        return this.getDigitalOceanNetworkZone(location)
      case 'linode':
        return this.getLinodeNetworkZone(location)
      case 'vultr':
        return location
      case 'google_cloud':
        return location
      default:
        return location
    }
  }

  /**
   * Get Hetzner network zone from location
   */
  private getHetznerNetworkZone(location: string): string {
    const locationToZone: Record<string, string> = {
      'nbg1': 'eu-central',    // Nuremberg
      'fsn1': 'eu-central',    // Falkenstein
      'hel1': 'eu-central',    // Helsinki
      'ash': 'us-east',        // Ashburn
      'hil': 'us-west',        // Hillsboro
      'sin': 'ap-southeast',   // Singapore
    }
    return locationToZone[location] || 'eu-central'
  }

  /**
   * Get DigitalOcean network zone from location
   */
  private getDigitalOceanNetworkZone(location: string): string {
    const locationToZone: Record<string, string> = {
      'nyc1': 'us-east',
      'nyc2': 'us-east',
      'nyc3': 'us-east',
      'sfo1': 'us-west',
      'sfo2': 'us-west',
      'sfo3': 'us-west',
      'ams2': 'eu-west',
      'ams3': 'eu-west',
      'sgp1': 'ap-southeast',
      'lon1': 'eu-west',
      'fra1': 'eu-central',
      'tor1': 'us-east',
      'blr1': 'ap-south',
    }
    return locationToZone[location] || 'us-east'
  }

  /**
   * Get Linode network zone from location
   */
  private getLinodeNetworkZone(location: string): string {
    const locationToZone: Record<string, string> = {
      'us-east': 'us-east',
      'us-central': 'us-central',
      'us-west': 'us-west',
      'us-southeast': 'us-southeast',
      'ca-central': 'ca-central',
      'eu-west': 'eu-west',
      'eu-central': 'eu-central',
      'ap-south': 'ap-south',
      'ap-west': 'ap-west',
      'ap-southeast': 'ap-southeast',
      'ap-northeast': 'ap-northeast',
    }
    return locationToZone[location] || 'us-east'
  }

  @beforeCreate()
  public static async generateId(cluster: CloudProvider) {
    cluster.id = randomUUID()
  }
}