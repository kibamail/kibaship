import env from '#start/env'

const vaultConfig = {
  /*
  |--------------------------------------------------------------------------
  | Vault connection settings
  |--------------------------------------------------------------------------
  */
  connection: {
    address: env.get('VAULT_ADDR'),
  },

  /*
  |--------------------------------------------------------------------------
  | AppRole authentication for read operations
  |--------------------------------------------------------------------------
  */
  readRole: {
    roleId: env.get('VAULT_READ_ROLE_ID'),
    secretId: env.get('VAULT_READ_SECRET_ID'),
  },

  /*
  |--------------------------------------------------------------------------
  | AppRole authentication for write operations
  |--------------------------------------------------------------------------
  */
  writeRole: {
    roleId: env.get('VAULT_WRITE_ROLE_ID'),
    secretId: env.get('VAULT_WRITE_SECRET_ID'),
  },
}

export default vaultConfig

export interface VaultConfig {
  connection: {
    address: string
  }
  readRole: {
    roleId: string
    secretId: string
  }
  writeRole: {
    roleId: string
    secretId: string
  }
}
