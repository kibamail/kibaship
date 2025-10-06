import { exec } from 'node:child_process'
import { promisify } from 'node:util'
import logger from '@adonisjs/core/services/logger'

const execAsync = promisify(exec)

/**
 * Talos Link Status Interface
 */
export interface TalosLinkStatus {
  metadata: {
    id: string
    namespace: string
    type: string
    version: number
  }
  spec: {
    index: number
    linkState: boolean
    operationalState: string
    kind: string
    type: string
    flags: string
    hardwareAddr: string
    broadcastAddr: string
    mtu: number
    driver?: string
    speedMbit?: number
    duplex?: string
  }
}

/**
 * Talos Address Status Interface
 */
export interface TalosAddressStatus {
  metadata: {
    id: string
    namespace: string
    type: string
    version: number
  }
  spec: {
    address: string
    linkName: string
    linkIndex: number
    family: string
    scope: string
    flags: string
  }
}

/**
 * Talos Route Status Interface
 */
export interface TalosRouteStatus {
  metadata: {
    id: string
    namespace: string
    type: string
    version: number
  }
  spec: {
    family: string
    dst: string
    src: string
    gateway: string
    outLinkName: string
    outLinkIndex: number
    table: string
    priority: number
    scope: string
    type: string
    flags: string
    protocol: string
  }
}

/**
 * Network Detection Result
 */
export interface TalosNetworkDetection {
  nodeIp: string
  links: TalosLinkStatus[]
  addresses: TalosAddressStatus[]
  routes: TalosRouteStatus[]
  publicInterface: {
    name: string
    gateway: string
    ipAddress: string
  } | null
}

export class TalosBareMetalNetworkDetectionService {
  /**
   * Detect network configuration on a Talos node
   */
  async detect(nodeIp: string): Promise<TalosNetworkDetection> {
    logger.info(`Starting network detection for node ${nodeIp}`)

    try {
      // Run all commands in parallel
      const [linksResult, addressesResult, routesResult] = await Promise.all([
        this.getLinks(nodeIp),
        this.getAddresses(nodeIp),
        this.getRoutes(nodeIp),
      ])

      // Find public interface details
      const publicInterface = this.findPublicInterface(
        linksResult,
        addressesResult,
        routesResult,
        nodeIp
      )

      return {
        nodeIp,
        links: linksResult,
        addresses: addressesResult,
        routes: routesResult,
        publicInterface,
      }
    } catch (error) {
      logger.error(`Network detection failed for node ${nodeIp}:`, error)
      throw error
    }
  }

  /**
   * Get network links from Talos node
   */
  private async getLinks(nodeIp: string): Promise<TalosLinkStatus[]> {
    const command = `talosctl -n ${nodeIp} get links --insecure -o json | jq -s '.'`
    const { stdout } = await execAsync(command)

    const links = JSON.parse(stdout.trim())

    logger.info(`Found ${links.length} network links on node ${nodeIp}`)
    return links
  }

  /**
   * Get network addresses from Talos node
   */
  private async getAddresses(nodeIp: string): Promise<TalosAddressStatus[]> {
    const command = `talosctl -n ${nodeIp} get addresses --insecure -o json | jq -s '.'`
    const { stdout } = await execAsync(command)

    const addresses = JSON.parse(stdout.trim())

    logger.info(`Found ${addresses.length} network addresses on node ${nodeIp}`)
    return addresses
  }

  /**
   * Get routes from Talos node
   */
  private async getRoutes(nodeIp: string): Promise<TalosRouteStatus[]> {
    const command = `talosctl -n ${nodeIp} get routes --insecure -o json | jq -s '.'`
    const { stdout } = await execAsync(command)

    const routes = JSON.parse(stdout.trim())

    logger.info(`Found ${routes.length} routes on node ${nodeIp}`)
    return routes
  }

  /**
   * Find the public interface name and gateway
   */
  private findPublicInterface(
    links: TalosLinkStatus[],
    addresses: TalosAddressStatus[],
    routes: TalosRouteStatus[],
    nodeIp: string
  ): { name: string; gateway: string; ipAddress: string } | null {
    // Find the default route (empty dst = 0.0.0.0/0)
    const defaultRoute = routes.find(
      (route) =>
        route.spec.family === 'inet4' &&
        route.spec.dst === '' &&
        route.spec.gateway &&
        route.spec.table === 'main'
    )

    if (!defaultRoute) {
      logger.warn(`No default route found on node ${nodeIp}`)
      return null
    }

    const publicInterfaceName = defaultRoute.spec.outLinkName
    const gateway = defaultRoute.spec.gateway

    // Find the IP address on this interface that matches the node IP
    const publicAddress = addresses.find(
      (addr) =>
        addr.spec.linkName === publicInterfaceName &&
        addr.spec.family === 'inet4' &&
        addr.spec.address.startsWith(nodeIp)
    )

    if (!publicAddress) {
      logger.warn(
        `Could not find public IP address on interface ${publicInterfaceName} for node ${nodeIp}`
      )
      return null
    }

    logger.info(
      `Detected public interface for ${nodeIp}: ${publicInterfaceName}, gateway: ${gateway}`
    )

    return {
      name: publicInterfaceName,
      gateway,
      ipAddress: publicAddress.spec.address,
    }
  }
}
