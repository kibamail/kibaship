import { ChildProcess } from '#utils/child_process'

export interface TalosMetadata {
  created: string
  id: string
  namespace: string
  owner: string
  phase: string
  type: string
  updated: string
  version: number
}

export interface DiskSpec {
  bus_path: string
  cdrom: boolean
  dev_path: string
  io_size: number
  modalias?: string
  model?: string
  pretty_size: string
  readonly: boolean
  rotational?: boolean
  sector_size: number
  size: number
  sub_system: string
  symlinks: string[]
  transport?: string
}

export interface TalosDisk {
  metadata: TalosMetadata
  node: string
  spec: DiskSpec
}

export interface AddressSpec {
  address: string
  family: 'inet4' | 'inet6'
  flags: string
  linkIndex: number
  linkName: string
  priority: number
  scope: 'host' | 'link' | 'global'
}

export interface TalosAddress {
  metadata: TalosMetadata
  node: string
  spec: AddressSpec
}

export interface RouteSpec {
  dst: string
  family: 'inet4' | 'inet6'
  flags: string
  gateway: string
  outLinkIndex: number
  outLinkName: string
  priority: number
  protocol: string
  scope: string
  src: string
  table: string
  type: string
}

export interface TalosRoute {
  metadata: TalosMetadata
  node: string
  spec: RouteSpec
}

export interface TalosCtlOptions {
  nodes?: string[]
  insecure?: boolean
}



export class TalosCtl {
  private defaultOptions: TalosCtlOptions

  constructor(options: TalosCtlOptions = {}) {
    this.defaultOptions = {
      insecure: true,
      ...options,
    }
  }

  private async executeCommand<T>(command: string, args: string[]): Promise<[T[] | null, Error | null]> {
    const [result, error] = await new ChildProcess()
      .command(command)
      .args(args)
      .executeAsync()

    if (!result || error) {
      return [null, error]
    }

    let results: T[] = []

    try {
      results = parseMultipleJsonObjects<T>(result.stdout)
    } catch (error) {
      console.error(error)
      return [null, error as Error]
    }


    return [results, null]
  }

  private buildArgs(subcommand: string, resource: string, options: TalosCtlOptions = {}): string[] {
    const mergedOptions = { ...this.defaultOptions, ...options }
    const args: string[] = []

    if (mergedOptions.nodes && mergedOptions.nodes.length > 0) {
      args.push('--nodes', mergedOptions.nodes.join(','))
    }

    args.push(subcommand, resource)

    if (mergedOptions.insecure) {
      args.push('--insecure')
    }

    args.push('--output', 'json')

    return args
  }

  async getDisks(options: TalosCtlOptions = {}) {
    const args = this.buildArgs('get', 'disks', options)

    return this.executeCommand<TalosDisk>('talosctl', args)
  }

  async getAddresses(options: TalosCtlOptions = {}) {
    const args = this.buildArgs('get', 'addresses', options)

    return this.executeCommand<TalosAddress>('talosctl', args)
  }

  async getRoutes(options: TalosCtlOptions = {}) {
    const args = this.buildArgs('get', 'routes', options)

    return this.executeCommand<TalosRoute>('talosctl', args)
  }
}

function parseMultipleJsonObjects<T>(content: string): T[] {
  // Split by closing brace + newline + opening brace
  const parts = content.trim().split('}\n{');

  const objects = parts.map((part, index, array) => {
    // Restore the braces removed by split
    if (index === 0 && array.length > 1) {
      return part + '}';  // First object: add closing brace
    } else if (index === array.length - 1 && array.length > 1) {
      return '{' + part;  // Last object: add opening brace
    } else if (array.length > 1) {
      return '{' + part + '}';  // Middle objects: add both braces
    } else {
      return part;  // Single object: use as-is
    }
  }).map(obj => JSON.parse(obj.trim()));

  return objects;
}
