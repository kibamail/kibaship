import { randomUUID } from 'node:crypto'
import { exec } from 'node:child_process'
import { readFile, unlink } from 'node:fs/promises'
import { promisify } from 'node:util'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const execAsync = promisify(exec)

export interface SshKeyPair {
  publicKey: string | null
  privateKey: string | null
  id: string
}

export class SshKeyService {
  static async generateEd25519KeyPair(): Promise<SshKeyPair> {
    const id = randomUUID()
    const tempDir = tmpdir()
    const privateKeyPath = join(tempDir, `ssh_key_${id}`)
    const publicKeyPath = `${privateKeyPath}.pub`

    try {
      await execAsync(`ssh-keygen -t ed25519 -f "${privateKeyPath}" -N "" -C "${id}@kibaship.com"`)

      const [privateKey, publicKey] = await Promise.all([
        readFile(privateKeyPath, 'utf8'),
        readFile(publicKeyPath, 'utf8')
      ])

      await Promise.all([
        unlink(privateKeyPath).catch(() => { }),
        unlink(publicKeyPath).catch(() => { })
      ])

      return {
        id,
        publicKey: publicKey.trim(),
        privateKey: privateKey.trim()
      }
    } catch (error) {
      try {
        await Promise.all([
          unlink(privateKeyPath).catch(() => { }),
          unlink(publicKeyPath).catch(() => { })
        ])
      } catch {
      }

      return {
        id,
        publicKey: null,
        privateKey: null
      }
    }
  }
}
