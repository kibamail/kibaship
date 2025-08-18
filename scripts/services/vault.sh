#!/bin/bash

set -e

if [ -z "$VAULT_READ_SECRET_ID" ]; then
    echo "Error: VAULT_READ_SECRET_ID environment variable is required"
    exit 1
fi

if [ -z "$VAULT_WRITE_SECRET_ID" ]; then
    echo "Error: VAULT_WRITE_SECRET_ID environment variable is required"
    exit 1
fi

echo "🚀 Starting Vault configuration for workspace-scoped secrets..."

# --- 1. Enable Necessary Secrets & Auth Engines ---
echo "Enabling KVv2 secrets engine at path 'secrets'..."
vault secrets enable -path=secrets kv-v2 || echo "KV 'secrets' engine already enabled."

echo "Enabling AppRole auth method..."
vault auth enable approle || echo "AppRole auth already enabled."


# --- 2. Create Granular Access Policies ---
READ_POLICY_NAME="kibaship-reads-policy"
WRITE_POLICY_NAME="kibaship-writes-policy"

echo "Creating read-only policy: $READ_POLICY_NAME"
vault policy write $READ_POLICY_NAME - <<EOF
# Grant read access to secrets within any workspace.
path "secrets/data/workspaces/*" {
  capabilities = ["read"]
}
EOF

echo "Creating write-only policy: $WRITE_POLICY_NAME"
vault policy write $WRITE_POLICY_NAME - <<EOF
# Grant write access to secrets within any workspace.
path "secrets/data/workspaces/*" {
  capabilities = ["create", "update"]
}
EOF


# --- 3. Configure AppRoles for the Application ---
READ_APPROLE_NAME="kibaship-reads"
WRITE_APPROLE_NAME="kibaship-writes"

echo "Creating AppRole: $READ_APPROLE_NAME"
vault write auth/approle/role/$READ_APPROLE_NAME \
    role_id="$READ_APPROLE_NAME" \
    token_policies="$READ_POLICY_NAME" \
    token_ttl=1h \
    token_max_ttl=4h

echo "Creating AppRole: $WRITE_APPROLE_NAME"
vault write auth/approle/role/$WRITE_APPROLE_NAME \
    role_id="$WRITE_APPROLE_NAME" \
    token_policies="$WRITE_POLICY_NAME" \
    token_ttl=10m \
    token_max_ttl=30m


vault write auth/approle/role/$READ_APPROLE_NAME/custom-secret-id secret_id="$VAULT_READ_SECRET_ID" > /dev/null
vault write auth/approle/role/$WRITE_APPROLE_NAME/custom-secret-id secret_id="$VAULT_WRITE_SECRET_ID" > /dev/null

echo "✅ Vault configuration complete!"
