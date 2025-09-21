import { Job } from '@rlanz/bull-queue'
import CloudProvider from '#models/cloud_provider'
import { DateTime } from 'luxon'
import logger from '@adonisjs/core/services/logger'
import { TalosPackerService } from '#services/hetzner/talos_packer_service'
import { TalosPackerExecutor } from '#services/hetzner/talos_packer_executor'
import { HetznerService } from '#services/hetzner/hetzner_service'
import { talosVersion } from '#config/app'

interface ProvisionHetznerTalosImageJobPayload {
  cloudProviderId: string
}

export default class ProvisionHetznerTalosImageJob extends Job {
  static get $$filepath() {
    return import.meta.url
  }

  async handle(payload: ProvisionHetznerTalosImageJobPayload) {
    const provider = await CloudProvider.find(payload.cloudProviderId)

    if (!provider || provider.type !== 'hetzner') {
      logger.error(
        'ProvisionHetznerTalosImageJob: Cloud provider not found or not hetzner',
        payload
      )
      return
    }

    provider.providerImageProvisioningStartedAt = DateTime.now()
    provider.providerImageProvisioningErrorAt = null
    provider.providerImageProvisioningCompletedAt = null

    await provider.save()

    const service = new TalosPackerService(provider.id)
    const builds = await service.buildBothArchitectures()

    const token = provider.credentials?.token || ''

    if (!token) {
      throw new Error('Hetzner token is missing in cloud provider credentials')
    }

    const [existingImages, error] = await new HetznerService(token).images().get('snapshot')

    if (error) {
      throw new Error('Failed to get account images.')
    }

    for (const build of builds) {
      const buildExists = existingImages.images.find(
        (image) =>
          image.labels['os'] === `kibaship-talos-v-${talosVersion}` &&
          image.labels['arch'] === build.arch
      )

      if (buildExists) {
        continue
      }

      await new TalosPackerExecutor(build.dir, token).run()
    }

    const [images, imagesError] = await new HetznerService(token).images().get('snapshot')

    if (imagesError) {
      throw new Error('Failed to get account images after uploading snapshots')
    }

    const arm64Image = images.images.find(
      (image) =>
        image.labels['os'] === `kibaship-talos-v-${talosVersion}` &&
        image.labels['arch'] === 'arm64'
    )
    const amd64Image = images.images.find(
      (image) =>
        image.labels['os'] === `kibaship-talos-v-${talosVersion}` &&
        image.labels['arch'] === 'amd64'
    )

    if (!arm64Image || !amd64Image) {
      throw new Error('Failed to find uploaded images after uploading snapshots')
    }

    provider.providerImageProvisioningCompletedAt = DateTime.now()
    provider.providerImageAmd64 = amd64Image.id.toString()
    provider.providerImageArm64 = arm64Image.id.toString()

    await provider.save()

    logger.info('ProvisionHetznerTalosImageJob: Completed successfully', {
      providerId: provider.id,
    })
  }

  async rescue(payload: ProvisionHetznerTalosImageJobPayload) {
    const provider = await CloudProvider.findOrFail(payload.cloudProviderId)
    provider.providerImageProvisioningErrorAt = DateTime.now()

    await provider.save()
  }
}
