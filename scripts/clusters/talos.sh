#!/usr/bin/env bash
set -euo pipefail

# Generate an 8-node Talos cluster (3 control planes, 5 workers)
# - Picks a free CIDR from a predefined list of 10
# - Persists used CIDRs in scripts/clusters/clusters.json
# - Writes kubeconfig.yaml and talosconfig.yaml into a per-cluster folder (by UUID)
# - Skips k8s node readiness checks since CNI will be installed later
#
# Usage:
#   scripts/clusters/talos.sh <cluster-name>
#
# Optional env vars:
#   TALOS_VERSION=<image tag> (optional)
#   KUBERNETES_VERSION=<semver> (optional)

command -v talosctl >/dev/null 2>&1 || {
  echo "talosctl is required but not found in PATH" >&2
  exit 1
}
command -v python3 >/dev/null 2>&1 || {
  echo "python3 is required for CIDR/IP calculations" >&2
  exit 1
}

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <cluster-name>" >&2
  exit 1
fi
CLUSTER_NAME="$1"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INDEX_JSON="${SCRIPT_DIR}/clusters.json"

# 10 candidate CIDRs for local networks
POSSIBLE_CIDRS=(
  "10.6.1.0/24" "10.5.2.0/24" "10.5.2.0/24" "10.5.3.0/24" "10.5.4.0/24"
  "10.6.0.0/24" "10.5.1.0/24" "10.6.2.0/24" "10.6.3.0/24" "10.6.4.0/24"
)

# Create JSON index if missing
if [[ ! -f "${INDEX_JSON}" ]]; then
  echo "[]" > "${INDEX_JSON}"
fi

# Pick a free CIDR not recorded in clusters.json
SELECTED_CIDR="$(python3 - "$INDEX_JSON" "${POSSIBLE_CIDRS[@]}" <<'PY'
import json, sys
path = sys.argv[1]
candidates = sys.argv[2:]
try:
    with open(path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = []
used = {item.get('cidr') for item in data if isinstance(item, dict)}
for c in candidates:
    if c not in used:
        print(c)
        break
PY
)"
if [[ -z "${SELECTED_CIDR}" ]]; then
  echo "No available CIDRs left in the candidate list." >&2
  exit 1
fi

# Create a RFC4122 cluster UUID
make_uuid() {
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]'
  else
    python3 - <<'PY'
import uuid
print(uuid.uuid4())
PY
  fi
}

CLUSTER_ID="$(make_uuid)"
OUT_DIR="${SCRIPT_DIR}/${CLUSTER_ID}"
mkdir -p "${OUT_DIR}"

KUBECONFIG_PATH="${OUT_DIR}/kubeconfig.yaml"
TALOSCONFIG_PATH="${OUT_DIR}/talosconfig.yaml"

# Compute gateway (first host) and first control-plane IP (second host) from CIDR
read -r GATEWAY_IP CP1_IP < <(python3 - <<PY
import ipaddress
net = ipaddress.ip_network("${SELECTED_CIDR}", strict=False)
hosts = list(net.hosts())
# First host is gateway, second host for first control plane
print(str(hosts[0]), str(hosts[1]))
PY
)

# Build cluster create arguments
ARGS=(
  cluster create
  --name "${CLUSTER_NAME}"
  --controlplanes 3
  --workers 3
  --cidr "${SELECTED_CIDR}"
  --config-patch @"${SCRIPT_DIR}/cluster.yaml"
  --skip-k8s-node-readiness-check
)

# Optional versions
if [[ -n "${TALOS_VERSION:-}" ]]; then
  ARGS+=(--talos-version "${TALOS_VERSION}")
fi
if [[ -n "${KUBERNETES_VERSION:-}" ]]; then
  ARGS+=(--kubernetes-version "${KUBERNETES_VERSION}")
fi

# Create the cluster
echo "Creating Talos cluster '${CLUSTER_NAME}' (id=${CLUSTER_ID}) using CIDR ${SELECTED_CIDR}..."
if ! talosctl "${ARGS[@]}"; then
  echo "Cluster creation failed." >&2
  exit 1
fi

echo "Generating kubeconfig from control plane node ${CP1_IP} at ${KUBECONFIG_PATH}..."
# Download admin kubeconfig from the node into our path
if ! talosctl --nodes "${CP1_IP}" --endpoints "${CP1_IP}" kubeconfig --force "${KUBECONFIG_PATH}"; then
  echo "Failed to generate kubeconfig." >&2
  exit 1
fi

# Persist talosconfig: copy current talos client config as created by cluster create
if [[ -f "${HOME}/.talos/config" ]]; then
  cp "${HOME}/.talos/config" "${TALOSCONFIG_PATH}"
  echo "Saved talosconfig to ${TALOSCONFIG_PATH}"
else
  echo "Warning: ${HOME}/.talos/config not found; talosconfig not saved" >&2
fi

# Update clusters.json index
python3 - <<PY
import json, os, sys, datetime
idx_path = "${INDEX_JSON}"
rec = {
  "id": "${CLUSTER_ID}",
  "name": "${CLUSTER_NAME}",
  "cidr": "${SELECTED_CIDR}",
  "folder": "${OUT_DIR}",
  "gateway": "${GATEWAY_IP}",
  "cp1": "${CP1_IP}",
  "created_at": datetime.datetime.utcnow().isoformat() + 'Z',
}
try:
    with open(idx_path) as f:
        data = json.load(f)
except FileNotFoundError:
    data = []
# Avoid duplicates by id
if not any(isinstance(x, dict) and x.get('id') == rec['id'] for x in data):
    data.append(rec)
with open(idx_path, 'w') as f:
    json.dump(data, f, indent=2)
PY

# Summary
cat <<EOF

Cluster created successfully.

  Cluster ID:       ${CLUSTER_ID}
  Cluster Name:     ${CLUSTER_NAME}
  CIDR:             ${SELECTED_CIDR}
  Gateway IP:       ${GATEWAY_IP}
  CP1 IP:           ${CP1_IP}
  Output Directory: ${OUT_DIR}
  Kubeconfig:       ${KUBECONFIG_PATH}
  Talosconfig:      ${TALOSCONFIG_PATH}
  Index JSON:       ${INDEX_JSON}

Next steps:
  - export KUBECONFIG=${KUBECONFIG_PATH}
  - talosctl --talosconfig ${TALOSCONFIG_PATH} --nodes ${CP1_IP} get machines
  - Install CNI manually (as planned), then verify Kubernetes node readiness.
EOF
