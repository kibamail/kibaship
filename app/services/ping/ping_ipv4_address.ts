import ping from 'ping'
import logger from '@adonisjs/core/services/logger'

export class PingIpv4Address {
  /**
   * Ping an IPv4 address to check if it's reachable
   * @param ipAddress - The IPv4 address to ping
   * @returns Promise<boolean> - True if the host is alive, false otherwise
   */
  async ping(ipAddress: string): Promise<boolean> {
    try {
      const result = await ping.promise.probe(ipAddress, {
        timeout: 10,
        min_reply: 1,
      })

      logger.info(`Ping result for ${ipAddress}: ${result.alive ? 'alive' : 'dead'}`)

      return result.alive
    } catch (error) {
      logger.error(`Error pinging ${ipAddress}:`, error)
      return false
    }
  }

  /**
   * Wait for a server to become reachable by pinging it repeatedly
   * @param ipAddress - The IPv4 address to ping
   * @param maxAttempts - Maximum number of attempts (default: 24 for 8 minutes at 20s intervals)
   * @param intervalSeconds - Interval between attempts in seconds (default: 20)
   * @returns Promise<boolean> - True if server became reachable, false if timeout
   */
  async waitUntilReachable(
    ipAddress: string,
    maxAttempts: number = 24,
    intervalSeconds: number = 20
  ): Promise<boolean> {
    logger.info(
      `Waiting for ${ipAddress} to become reachable (max ${maxAttempts} attempts, ${intervalSeconds}s interval)`
    )

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      logger.info(`Attempt ${attempt}/${maxAttempts} to ping ${ipAddress}`)

      const isReachable = await this.ping(ipAddress)

      if (isReachable) {
        logger.info(`Server ${ipAddress} is now reachable!`)
        return true
      }

      if (attempt < maxAttempts) {
        logger.info(`Server ${ipAddress} not reachable yet, waiting ${intervalSeconds}s...`)
        await this.sleep(intervalSeconds * 1000)
      }
    }

    logger.warn(`Server ${ipAddress} did not become reachable after ${maxAttempts} attempts`)
    return false
  }

  /**
   * Sleep for a specified number of milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms))
  }
}
