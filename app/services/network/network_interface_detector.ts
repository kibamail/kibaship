import { TalosAddress } from '#services/talos/talos_ctl'

export interface NetworkInterface {
  linkName: string
  hasPrivateIP: boolean
  hasPublicIP: boolean
  privateIPs: string[]
  publicIPs: string[]
}

export interface NetworkInterfaceDetectionResult {
  privateInterface: string | null
  publicInterface: string | null
  allInterfaces: Record<string, NetworkInterface>
}

export class NetworkInterfaceDetector {
  /**
   * Detects network interfaces from Talos address statuses
   */
  static detectNetworkInterfaces(addresses: TalosAddress[]): NetworkInterfaceDetectionResult {
    const interfaces = new Map<string, NetworkInterface>()

    // Parse all addresses and group by interface
    addresses.forEach((addr) => {
      const { linkName, address } = addr.spec
      
      // Skip loopback and external virtual interfaces
      if (linkName === 'lo' || linkName === 'external') return
      
      // Skip IPv6 addresses for simplicity
      if (addr.spec.family === 'inet6') return

      const ip = address.split('/')[0]
      const isPrivate = this.isPrivateIP(ip)
      
      if (!interfaces.has(linkName)) {
        interfaces.set(linkName, {
          linkName,
          hasPrivateIP: false,
          hasPublicIP: false,
          privateIPs: [],
          publicIPs: []
        })
      }
      
      const iface = interfaces.get(linkName)!
      
      if (isPrivate) {
        iface.hasPrivateIP = true
        iface.privateIPs.push(ip)
      } else {
        iface.hasPublicIP = true  
        iface.publicIPs.push(ip)
      }
    })

    // Priority logic for private interface selection:
    // 1. Interface with private IP and scope 'global' (dedicated private network)
    // 2. Interface with only private IPs
    // 3. Fallback to first interface with private IP
    
    const privateInterface = Array.from(interfaces.values()).find(iface => 
      iface.hasPrivateIP && !iface.hasPublicIP
    ) || Array.from(interfaces.values()).find(iface => iface.hasPrivateIP)
    
    const publicInterface = Array.from(interfaces.values()).find(iface => 
      iface.hasPublicIP
    )

    return {
      privateInterface: privateInterface?.linkName || null,
      publicInterface: publicInterface?.linkName || null,
      allInterfaces: Object.fromEntries(interfaces)
    }
  }

  /**
   * Checks if an IP address is in private range (RFC 1918)
   */
  private static isPrivateIP(ip: string): boolean {
    const parts = ip.split('.').map(Number)
    
    // Validate IP format
    if (parts.length !== 4 || parts.some(part => isNaN(part) || part < 0 || part > 255)) {
      return false
    }
    
    // 10.0.0.0/8 (10.0.0.0 to 10.255.255.255)
    if (parts[0] === 10) return true
    
    // 172.16.0.0/12 (172.16.0.0 to 172.31.255.255)
    if (parts[0] === 172 && parts[1] >= 16 && parts[1] <= 31) return true
    
    // 192.168.0.0/16 (192.168.0.0 to 192.168.255.255)
    if (parts[0] === 192 && parts[1] === 168) return true
    
    return false
  }

  /**
   * Gets the primary private IP address for a given interface from addresses
   */
  static getPrimaryPrivateIP(addresses: TalosAddress[], interfaceName: string): string | null {
    const privateAddress = addresses.find(addr => 
      addr.spec.linkName === interfaceName && 
      addr.spec.family === 'inet4' &&
      this.isPrivateIP(addr.spec.address.split('/')[0])
    )
    
    return privateAddress ? privateAddress.spec.address.split('/')[0] : null
  }

  /**
   * Gets the primary public IP address for a given interface from addresses
   */
  static getPrimaryPublicIP(addresses: TalosAddress[], interfaceName: string): string | null {
    const publicAddress = addresses.find(addr => 
      addr.spec.linkName === interfaceName && 
      addr.spec.family === 'inet4' &&
      !this.isPrivateIP(addr.spec.address.split('/')[0])
    )
    
    return publicAddress ? publicAddress.spec.address.split('/')[0] : null
  }
}