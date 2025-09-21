import edge from 'edge.js'
import app from '@adonisjs/core/services/app'
import drive from '@adonisjs/drive/services/main'
import { join } from 'node:path'
import { talosFactoryHashHetzner, talosVersion } from '#config/app'

export interface TalosPackerTemplateContext {
  cluster_talos_factory_hash_hetzner: string
  talos_version: string
  arch?: string
  server_type?: string
  server_location?: string
}

/**
 * Service to render the Hetzner Talos Packer template into the storage directory.
 *
 * It renders resources/views/cluster_providers/hetzner/talos.pkr.hcl.edge
 * to storage/packer/hetzner/<cloud_provider_uuid>/talos-image/<arch>/talos.pkr.hcl
 */
export class TalosPackerService {
  private edge = edge
  private disk = drive.use('fs')
  private baseKey: string

  constructor(private providerId: string) {
    this.baseKey = `packer/hetzner/${this.providerId}/talos-image`
  }

  private keyToPath(key: string): string {
    return join(app.makePath('storage'), key)
  }

  /** Mount Edge views root so we can render provider templates */
  async init() {
    const viewsPath = app.makePath('resources/views/cluster_providers/hetzner')
    this.edge.mount(viewsPath)
  }

  /**
   * Render the Packer template and write it to storage.
   * Returns the storage key, absolute path to the file, and working directory.
   */
  async buildTemplate(
    context?: Partial<TalosPackerTemplateContext>
  ): Promise<{ key: string; path: string; dir: string }> {
    await this.init()

    const ctx: TalosPackerTemplateContext = {
      cluster_talos_factory_hash_hetzner: talosFactoryHashHetzner,
      talos_version: talosVersion,
      ...context,
    }

    const content = await this.edge.render('cluster_providers/hetzner/talos.pkr.hcl', ctx)

    const arch = ctx.arch || 'amd64'
    const dirKey = `${this.baseKey}/${arch}`
    const fileKey = `${dirKey}/talos.pkr.hcl`

    await this.disk.put(fileKey, content)

    return {
      key: fileKey,
      path: this.keyToPath(fileKey),
      dir: this.keyToPath(dirKey),
    }
  }

  /**
   * Build templates for both architectures under provider-scoped persistent folders.
   * Returns array of per-arch outputs with working directories.
   */
  async buildBothArchitectures(): Promise<
    Array<{ arch: string; key: string; path: string; dir: string }>
  > {
    await this.init()

    const variants: Array<
      Required<Pick<TalosPackerTemplateContext, 'arch' | 'server_type' | 'server_location'>>
    > = [
      { arch: 'arm64', server_type: 'cax11', server_location: 'fsn1' },
      { arch: 'amd64', server_type: 'cx22', server_location: 'hel1' },
    ]

    const results: Array<{ arch: string; key: string; path: string; dir: string }> = []

    for (const variant of variants) {
      const ctx: TalosPackerTemplateContext = {
        cluster_talos_factory_hash_hetzner: talosFactoryHashHetzner,
        talos_version: talosVersion,
        arch: variant.arch,
        server_type: variant.server_type,
        server_location: variant.server_location,
      }

      const content = await this.edge.render('talos.pkr.hcl', ctx)

      const baseRoot = `packer/hetzner/${this.providerId}/talos-image`
      const dirKey = `${baseRoot}/${variant.arch}`
      const fileKey = `${dirKey}/talos.pkr.hcl`

      await this.disk.put(fileKey, content)

      results.push({
        arch: variant.arch,
        key: fileKey,
        path: this.keyToPath(fileKey),
        dir: this.keyToPath(dirKey),
      })
    }

    return results
  }
}
