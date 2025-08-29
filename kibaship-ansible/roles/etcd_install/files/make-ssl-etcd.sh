#!/bin/bash

# etcd Certificate Generation Script
# Based on Kubespray patterns for production etcd clusters

set -o errexit
set -o nounset
set -o pipefail

ETCD_CONFIG_DIR="${1:-/etc/etcd}"
HOSTNAME="${2:-$(hostname)}"
IP_ADDRESS="${3:-127.0.0.1}"

# Ensure config directory exists
mkdir -p "${ETCD_CONFIG_DIR}"
cd "${ETCD_CONFIG_DIR}"

# Generate CA private key
if [ ! -f ca-key.pem ]; then
    openssl genrsa -out ca-key.pem 2048
fi

# Generate CA certificate
if [ ! -f ca.pem ]; then
    openssl req -new -x509 -key ca-key.pem -out ca.pem -days 3650 -subj "/CN=etcd-ca"
fi

# Generate server private key
if [ ! -f server-key.pem ]; then
    openssl genrsa -out server-key.pem 2048
fi

# Generate server certificate signing request
openssl req -new -key server-key.pem -out server.csr -subj "/CN=${HOSTNAME}" \
    -config <(cat << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = ${HOSTNAME}

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${HOSTNAME}
DNS.2 = localhost
IP.1 = ${IP_ADDRESS}
IP.2 = 127.0.0.1
EOF
)

# Generate server certificate
if [ ! -f server.pem ]; then
    openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
        -out server.pem -days 3650 -extensions v3_req \
        -extfile <(cat << EOF
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${HOSTNAME}
DNS.2 = localhost
IP.1 = ${IP_ADDRESS}
IP.2 = 127.0.0.1
EOF
)
fi

# Generate client private key
if [ ! -f client-key.pem ]; then
    openssl genrsa -out client-key.pem 2048
fi

# Generate client certificate signing request
openssl req -new -key client-key.pem -out client.csr -subj "/CN=etcd-client"

# Generate client certificate
if [ ! -f client.pem ]; then
    openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
        -out client.pem -days 3650
fi

# Clean up CSR files
rm -f server.csr client.csr

# Set proper permissions
chmod 644 ca.pem server.pem client.pem
chmod 600 ca-key.pem server-key.pem client-key.pem
chown etcd:etcd server-key.pem || echo "etcd user not yet created, will set ownership later"
chown root:root client-key.pem ca-key.pem

echo "etcd TLS certificates generated successfully in ${ETCD_CONFIG_DIR}"