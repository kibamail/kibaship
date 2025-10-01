# Docker Registry v3.0.0 - Complete Documentation

This document provides comprehensive documentation for the Docker Registry v3.0.0 implementation in the Kibaship Operator, including JWT token authentication, namespace-based multi-tenancy, and step-by-step setup instructions.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [How the Docker Registry Works](#how-the-docker-registry-works)
3. [JWT Token Authentication Flow](#jwt-token-authentication-flow)
4. [Critical Docker Registry v3.0.0 Changes](#critical-docker-registry-v30-changes)
5. [Components](#components)
6. [Setup Instructions](#setup-instructions)
7. [Troubleshooting](#troubleshooting)
8. [Security Considerations](#security-considerations)

---

## Architecture Overview

The Docker Registry implementation consists of three main components:

```
┌─────────────────────────────────────────────────────────────┐
│                    Namespace: registry                       │
│                                                              │
│  ┌──────────────┐         ┌─────────────┐                  │
│  │   registry   │◄────────│ registry-   │                  │
│  │   (v3.0.0)   │  JWT    │   auth      │                  │
│  │              │  tokens │  service    │                  │
│  └──────┬───────┘         └─────────────┘                  │
│         │                                                    │
│         │ Validates JWT using:                             │
│         │ 1. /etc/registry-auth-keys/tls.crt              │
│         │ 2. /etc/registry-auth-keys-jwks/jwks.json       │
│         │                                                    │
└─────────┼──────────────────────────────────────────────────┘
          │
          │ HTTPS (TLS)
          │
┌─────────┼──────────────────────────────────────────────────┐
│         │        Namespace: test-build                      │
│         │                                                    │
│  ┌──────▼───────┐                                          │
│  │   BuildKit   │  Credentials from:                       │
│  │     Pod      │  - test-build-registry-credentials       │
│  │              │  - registry-docker-config                │
│  └──────────────┘  - registry-ca-cert                      │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Key Design Principles

1. **Namespace-based Multi-tenancy**: Repository paths must match the namespace name (e.g., `test-build/myapp` for namespace `test-build`)
2. **Per-namespace Credentials**: Each namespace has its own `<namespace>-registry-credentials` secret
3. **JWT Token Authentication**: Uses RS256 algorithm with JWKS (JSON Web Key Set) validation
4. **TLS Everywhere**: All communication is encrypted using self-signed certificates managed by cert-manager

---

## How the Docker Registry Works

### Registry Configuration

The Docker Registry v3.0.0 is configured with the following key settings:

**Location**: `config/registry/base/configmap.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: registry-config
  namespace: registry
data:
  config.yml: |
    version: 0.1
    log:
      level: debug
    storage:
      filesystem:
        rootdirectory: /var/lib/registry
      delete:
        enabled: true
    http:
      addr: :5000
      tls:
        certificate: /etc/registry-tls/tls.crt
        key: /etc/registry-tls/tls.key
    auth:
      token:
        realm: http://registry-auth.registry.svc.cluster.local/auth
        service: docker-registry
        issuer: registry-token-issuer
        rootcertbundle: /etc/registry-auth-keys/tls.crt
        jwks: /etc/registry-auth-keys-jwks/jwks.json  # CRITICAL for v3.0.0
```

### Critical Configuration Details

1. **Config Path**: Docker Registry v3.0.0 expects the configuration file at `/etc/distribution/config.yml` (NOT `/etc/docker/registry/config.yml` like v2.x)

2. **TLS Configuration**: The registry serves HTTPS on port 5000 internally, exposed as port 443 via Kubernetes Service

3. **Auth Token Configuration**:
   - `realm`: The URL where Docker clients will be redirected for authentication
   - `service`: The service name that must match in JWT tokens (audience claim)
   - `issuer`: The issuer name that must match in JWT tokens (issuer claim)
   - `rootcertbundle`: X.509 certificate file containing the public key for JWT signature verification
   - `jwks`: **NEW in v3.0.0** - JSON Web Key Set file for JWT validation

### Registry Deployment

**Location**: `config/registry/base/deployment.yaml`

Key aspects of the deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: registry
spec:
  replicas: 2
  template:
    spec:
      enableServiceLinks: false  # CRITICAL: Prevents env var pollution
      containers:
        - name: registry
          image: registry:3.0.0
          ports:
            - containerPort: 5000
              name: https
          volumeMounts:
            - name: registry-config
              mountPath: /etc/distribution  # v3.0.0 path
              readOnly: true
            - name: registry-storage
              mountPath: /var/lib/registry
            - name: registry-tls
              mountPath: /etc/registry-tls
              readOnly: true
            - name: registry-auth-keys
              mountPath: /etc/registry-auth-keys
              readOnly: true
            - name: registry-auth-keys-jwks  # NEW for v3.0.0
              mountPath: /etc/registry-auth-keys-jwks
              readOnly: true
          env:
            - name: REGISTRY_HTTP_SECRET
              valueFrom:
                secretKeyRef:
                  name: registry-registry-auth
                  key: http-secret
      volumes:
        - name: registry-config
          configMap:
            name: registry-config
        - name: registry-storage
          persistentVolumeClaim:
            claimName: registry-storage
        - name: registry-tls
          secret:
            secretName: registry-tls
        - name: registry-auth-keys
          secret:
            secretName: registry-auth-keys
            items:
              - key: tls.crt
                path: tls.crt
        - name: registry-auth-keys-jwks  # NEW for v3.0.0
          secret:
            secretName: registry-auth-keys-jwks
```

**Important Notes**:

1. `enableServiceLinks: false` - **CRITICAL**: Without this, Kubernetes injects environment variables like `REGISTRY_AUTH_PORT` which conflict with the registry's configuration parsing, causing startup failures.

2. Config mount path is `/etc/distribution` for v3.0.0 (changed from `/etc/docker/registry` in v2.x)

3. Two separate secrets for auth keys:
   - `registry-auth-keys`: Contains the X.509 certificate (tls.crt)
   - `registry-auth-keys-jwks`: Contains the JWKS JSON file (jwks.json)

### Registry Service

**Location**: `config/registry/base/service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: registry
spec:
  type: ClusterIP
  ports:
    - name: https
      port: 443
      targetPort: 5000
      protocol: TCP
  selector:
    app: registry
```

The service exposes the registry on standard HTTPS port 443, forwarding to container port 5000.

---

## JWT Token Authentication Flow

The complete authentication flow involves multiple steps:

### 1. Initial Push Attempt (No Auth)

```
BuildKit Pod → Registry
  POST /v2/test-build/myapp/manifests/latest
```

**Registry Response**:
```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="http://registry-auth.registry.svc.cluster.local/auth",service="docker-registry",scope="repository:test-build/myapp:push,pull"
```

### 2. Token Request to Registry-Auth Service

```
BuildKit Pod → Registry-Auth Service
  GET /auth?service=docker-registry&scope=repository:test-build/myapp:push,pull
  Authorization: Basic dGVzdC1idWlsZDp0ZXN0LXBhc3N3b3JkLTEyMw==
```

**Registry-Auth Processing**:

1. **Extract credentials** from Basic Auth header
   - Username: `test-build`
   - Password: `test-password-123`

2. **Parse scope** parameter to extract repository and actions
   - Scope: `repository:test-build/myapp:push,pull`
   - Repository: `test-build/myapp`
   - Actions: `["push", "pull"]`

3. **Extract namespace** from repository path
   - Repository: `test-build/myapp`
   - Namespace: `test-build` (first path component)

4. **Validate credentials** against Kubernetes secret
   - Secret name: `test-build-registry-credentials` (pattern: `<namespace>-registry-credentials`)
   - Namespace: `test-build`
   - Compare username and password using constant-time comparison

5. **Generate JWT token** if credentials are valid
   - Algorithm: RS256 (RSA signature with SHA-256)
   - Private key: `/etc/registry-auth-keys/tls.key`
   - **Key ID (kid)**: `registry-auth-jwt-signer` (CRITICAL for v3.0.0)

**JWT Token Structure**:

Header:
```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "registry-auth-jwt-signer"
}
```

Payload:
```json
{
  "iss": "registry-token-issuer",
  "sub": "test-build",
  "aud": ["docker-registry"],
  "exp": 1759252997,
  "iat": 1759252697,
  "jti": "34e4b99f-7d48-436d-943b-198f1565d458",
  "access": [
    {
      "type": "repository",
      "name": "test-build/myapp",
      "actions": ["push", "pull"]
    }
  ]
}
```

**Registry-Auth Response**:
```json
{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJlZ2lzdHJ5LWF1dGgtand0LXNpZ25lciJ9...",
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJlZ2lzdHJ5LWF1dGgtand0LXNpZ25lciJ9...",
  "expires_in": 300,
  "issued_at": "2025-09-30T17:23:17Z"
}
```

### 3. Retry with JWT Token

```
BuildKit Pod → Registry
  POST /v2/test-build/myapp/manifests/latest
  Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InJlZ2lzdHJ5LWF1dGgtand0LXNpZ25lciJ9...
```

**Registry JWT Validation Process**:

1. **Extract JWT token** from Authorization header
2. **Parse JWT header** to get Key ID (kid)
3. **Lookup public key** in JWKS file using kid
4. **Verify JWT signature** using public key
5. **Validate claims**:
   - Issuer (iss) matches `registry-token-issuer`
   - Audience (aud) contains `docker-registry`
   - Token not expired (exp)
   - Token issued time valid (iat)
6. **Check access permissions**:
   - Repository name matches request
   - Requested action (push/pull) is in access list

**Registry Response** (on success):
```
HTTP/1.1 201 Created
Location: /v2/test-build/myapp/manifests/sha256:2dfa91aad1877a0b0bf2a2cf703bbedc06a50eef772b9dc6cda8e152b03fe677
```

### Registry-Auth Service Implementation

**Location**: `internal/registryauth/handler.go`

Key implementation details:

```go
func (h *Handler) ServeAuth(w http.ResponseWriter, r *http.Request) {
    // Extract query parameters
    service := r.URL.Query().Get("service")
    scope := r.URL.Query().Get("scope")

    // Extract Basic Auth credentials
    username, password, ok := r.BasicAuth()

    // Parse scope: "repository:test-build/myapp:push,pull"
    repo, actions, err := parseScope(scope)

    // Extract namespace from repository path
    // "test-build/myapp" → "test-build"
    namespace, err := extractNamespaceFromRepo(repo)

    // Build access grants
    accessGrants := []AccessEntry{
        {
            Type:    "repository",
            Name:    repo,
            Actions: actions,
        },
    }

    // Validate credentials against <namespace>-registry-credentials secret
    if !h.validator.ValidateCredentials(r.Context(), namespace, username, password) {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Generate JWT token with kid header
    token, expiresAt, err := h.tokenGenerator.GenerateToken(username, h.serviceName, accessGrants)

    // Return token response
    response := TokenResponse{
        Token:       token,
        AccessToken: token,
        ExpiresIn:   int(time.Until(expiresAt).Seconds()),
        IssuedAt:    time.Now().UTC().Format(time.RFC3339),
    }

    json.NewEncoder(w).Encode(response)
}
```

**Location**: `internal/registryauth/token.go`

JWT token generation with kid header:

```go
func (tg *TokenGenerator) GenerateToken(subject, audience string, access []AccessEntry) (string, time.Time, error) {
    now := time.Now()
    expiresAt := now.Add(tg.expiration)

    claims := TokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    tg.issuer,
            Subject:   subject,
            Audience:  jwt.ClaimStrings{audience},
            ExpiresAt: jwt.NewNumericDate(expiresAt),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        uuid.New().String(),
        },
        Access: access,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

    // CRITICAL: Set the Key ID (kid) header to match the JWKS key ID
    token.Header["kid"] = "registry-auth-jwt-signer"

    tokenString, err := token.SignedString(tg.privateKey)

    return tokenString, expiresAt, nil
}
```

**Location**: `internal/registryauth/validator.go`

Credential validation against Kubernetes secrets:

```go
func (v *Validator) ValidateCredentials(ctx context.Context, namespace, username, password string) bool {
    // Check cache first
    if cached, ok := v.cache.Get(namespace); ok {
        if cached.Username == username &&
           subtle.ConstantTimeCompare([]byte(cached.Password), []byte(password)) == 1 {
            return true
        }
    }

    // Fetch from Kubernetes
    // Secret must be named <namespace>-registry-credentials
    secretName := fmt.Sprintf("%s-registry-credentials", namespace)
    creds, err := v.k8sClient.GetCredentials(ctx, namespace, secretName)
    if err != nil {
        return false
    }

    // Validate credentials using constant-time comparison
    if creds.Username != username {
        return false
    }

    if subtle.ConstantTimeCompare([]byte(creds.Password), []byte(password)) != 1 {
        return false
    }

    // Cache valid credentials
    v.cache.Set(namespace, creds)
    return true
}
```

---

## Critical Docker Registry v3.0.0 Changes

### Breaking Changes from v2.x

Docker Distribution v3.0.0 introduced **breaking changes** in JWT token validation that caused our initial implementation to fail with "invalid token" errors.

#### The Problem

In **v2.8.3 and earlier**:
- JWT tokens could be validated using **only** the `rootcertbundle` parameter
- The registry would extract the public key from the X.509 certificate and use it to verify JWT signatures
- No Key ID (kid) was required in JWT headers

In **v3.0.0**:
- JWT tokens must **EITHER**:
  1. Include an `x5c` (X.509 Certificate Chain) header in the JWT, OR
  2. The registry must be configured with a **JWKS (JSON Web Key Set)** file
- The `rootcertbundle` alone is **NO LONGER SUFFICIENT**
- JWT tokens **MUST** include a `kid` (Key ID) header that matches a key in the JWKS file

#### The Solution

We implemented the **JWKS approach** with the following steps:

### 1. Generate JWKS JSON File

The JWKS file contains the public key in JSON Web Key format, derived from the X.509 certificate:

**Script to generate JWKS**:

```bash
#!/bin/bash
# Extract public key from certificate
kubectl get secret registry-auth-keys -n registry -o jsonpath='{.data.tls\.crt}' | base64 -d > /tmp/cert.pem

# Extract public key in PEM format
openssl x509 -in /tmp/cert.pem -pubkey -noout > /tmp/pubkey.pem

# Extract modulus and exponent from public key
MODULUS=$(openssl rsa -pubin -in /tmp/pubkey.pem -noout -modulus 2>/dev/null | sed 's/Modulus=//')
EXPONENT=$(openssl rsa -pubin -in /tmp/pubkey.pem -text -noout 2>/dev/null | grep Exponent | awk '{print $2}')

# Convert modulus from hex to base64url
MODULUS_BASE64=$(echo -n "$MODULUS" | xxd -r -p | base64 | tr '+/' '-_' | tr -d '=')

# Convert exponent to base64url (exponent is typically 65537 = 0x010001)
EXPONENT_HEX=$(printf '%x' "$EXPONENT")
if [ ${#EXPONENT_HEX} -eq 5 ]; then
    EXPONENT_HEX="0$EXPONENT_HEX"
fi
EXPONENT_BASE64=$(echo -n "$EXPONENT_HEX" | xxd -r -p | base64 | tr '+/' '-_' | tr -d '=')

# Generate JWKS JSON
cat > /tmp/jwks.json <<EOF
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "registry-auth-jwt-signer",
      "use": "sig",
      "alg": "RS256",
      "n": "$MODULUS_BASE64",
      "e": "$EXPONENT_BASE64"
    }
  ]
}
EOF
```

**Example JWKS file**:

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "registry-auth-jwt-signer",
      "use": "sig",
      "alg": "RS256",
      "n": "tEsKG4N8f8DaRvOT3Y68bcylDvshm_ODLXvFf07YdWCUPLW73oh3H8oUaRf7e4o_SbMVtoh-1N-V0bloa4kctdNOSDTLCcHa1T5uq65lO1BWn6lrsHXWN2AqRmm-1QBulcQtwkXRxKnw6m7fZQAgNn03ubDknzDl8okZ4eW3i1nh2bi2aaarfPCH5wJNYzJu9LpAIjWhu6FnzG6ItJCQyNo0IW_lifskgvBUYCL-CLhuNUx3vz4bZ-gyDpwe3TqpGgks8tWicL87HCaOTlFHMOcDq8zKnvfW3FrT73ezGT-qOPDOMsj5IswFZ2kZK2kJsoCOcmNafSKCfp3YA0SiR3YOlpTkz7-gKnqRrR3pf70pfpDSayXmyHtn0lJSBHAjyqqNlbUfHjJtzDjrNvXCU_Z7GOjT3H5_zE52J1Qm-XkTHu15TnWcYKLfRuvmNJ64NZMSDD4-LWFyR0dFgcW8dFy3b_7hC-oNH5tAzH0kat3uGCQs-x8uA_f8R8RMQ7-yHbmI-YrrMNrxLv1sKBM5lfVF7odRUy94JgTOlPMYsYgX9Oe1UtqRqlUNwgaztTB7UCUJmtmJEqkSx5IU2EeMWGHnhhRugdrI-Ku4B-P3SCqXZSbOfJUD1GZaksma5oe8MPIzx6teP9pmhv5ALDmvqzbS8aZkzeYpMZb3zFhUPa8",
      "e": "AQAB"
    }
  ]
}
```

**JWKS Field Descriptions**:
- `kty`: Key type (RSA for our use case)
- `kid`: Key ID - must match the `kid` in JWT token headers
- `use`: Public key use (sig = signature)
- `alg`: Algorithm (RS256 = RSA signature with SHA-256)
- `n`: RSA modulus (base64url-encoded)
- `e`: RSA public exponent (base64url-encoded, typically "AQAB" which is 65537)

### 2. Create JWKS Secret

```bash
kubectl create secret generic registry-auth-keys-jwks \
  -n registry \
  --from-file=jwks.json=/tmp/jwks.json
```

### 3. Update Registry Configuration

Add the `jwks` parameter to the registry configuration:

```yaml
auth:
  token:
    realm: http://registry-auth.registry.svc.cluster.local/auth
    service: docker-registry
    issuer: registry-token-issuer
    rootcertbundle: /etc/registry-auth-keys/tls.crt
    jwks: /etc/registry-auth-keys-jwks/jwks.json  # ADD THIS
```

### 4. Mount JWKS Secret in Registry Deployment

```yaml
volumeMounts:
  - name: registry-auth-keys-jwks
    mountPath: /etc/registry-auth-keys-jwks
    readOnly: true

volumes:
  - name: registry-auth-keys-jwks
    secret:
      secretName: registry-auth-keys-jwks
```

### 5. Include kid Header in JWT Tokens

Modify the token generator to include the `kid` header:

```go
token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
// Set the Key ID (kid) header to match the JWKS key ID
token.Header["kid"] = "registry-auth-jwt-signer"
```

### Verification

After implementing these changes, JWT tokens are properly validated:

**Before (v3.0.0 without JWKS)**:
```
time="2025-09-30T17:04:41.900246095Z" level=info msg="failed to verify token: invalid token"
```

**After (v3.0.0 with JWKS)**:
```
# No errors - tokens validated successfully
# Image push succeeds
pushing manifest for registry.registry.svc.cluster.local/test-build/myapp:latest@sha256:... done
```

---

## Components

### 1. Registry TLS Certificate

**Certificate**: `registry-tls`
**Purpose**: Provides TLS encryption for the registry HTTPS endpoint

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: registry-tls
  namespace: registry
spec:
  secretName: registry-tls
  issuerRef:
    name: registry-selfsigned-issuer
    kind: Issuer
  commonName: registry.registry.svc.cluster.local
  dnsNames:
    - registry.registry.svc.cluster.local
    - registry.registry.svc
    - registry.registry
    - registry
  duration: 87600h  # 10 years
  renewBefore: 720h  # 30 days
  privateKey:
    algorithm: RSA
    size: 4096
  usages:
    - digital signature
    - key encipherment
    - server auth
```

### 2. Registry Auth Keys Certificate

**Certificate**: `registry-auth-keys`
**Purpose**: Provides RSA key pair for JWT token signing and verification

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: registry-auth-keys
  namespace: registry
spec:
  secretName: registry-auth-keys
  issuerRef:
    name: registry-selfsigned-issuer
    kind: Issuer
  commonName: registry-auth-jwt-signer
  subject:
    organizations:
      - kibaship
  duration: 87600h  # 10 years
  renewBefore: 720h  # 30 days
  privateKey:
    algorithm: RSA
    size: 4096
  usages:
    - digital signature      # Required for JWT signing
    - key encipherment
```

**Contents**:
- `tls.crt`: X.509 certificate containing the public key
- `tls.key`: RSA private key (PKCS#1 or PKCS#8 format)

### 3. Registry Auth Keys JWKS Secret

**Secret**: `registry-auth-keys-jwks`
**Purpose**: Provides JWKS file for Docker Registry v3.0.0 JWT validation

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-auth-keys-jwks
  namespace: registry
type: Opaque
data:
  jwks.json: <base64-encoded-jwks-json>
```

### 4. Registry HTTP Secret

**Secret**: `registry-registry-auth`
**Purpose**: Provides HTTP session secret for the registry

Generated by the operator's bootstrap process:

```go
func generateHTTPSecret() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}
```

### 5. Self-Signed Issuer

**Issuer**: `registry-selfsigned-issuer`
**Purpose**: Issues self-signed certificates for internal registry components

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: registry-selfsigned-issuer
  namespace: registry
spec:
  selfSigned: {}
```

---

## Setup Instructions

### Prerequisites

1. Kubernetes cluster with cert-manager installed
2. kubectl configured to access the cluster
3. Operator deployed with registry components

### Step 1: Create the Target Namespace

Create a namespace where your build pods will run and push images:

```bash
kubectl create namespace test-build
```

### Step 2: Create Registry Credentials Secret

Each namespace must have a secret named `<namespace>-registry-credentials` containing the username and password for registry authentication.

**IMPORTANT**: The username **MUST** match the namespace name, and the secret name **MUST** follow the pattern `<namespace>-registry-credentials`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-build-registry-credentials
  namespace: test-build
type: Opaque
stringData:
  username: test-build
  password: test-password-123  # Use a strong password in production
```

Apply the secret:

```bash
kubectl apply -f test-build-registry-credentials.yaml
```

### Step 3: Create Docker Config Secret

BuildKit and other Docker clients need a `.docker/config.json` file with authentication credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-docker-config
  namespace: test-build
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "registry.registry.svc.cluster.local": {
          "username": "test-build",
          "password": "test-password-123",
          "auth": "dGVzdC1idWlsZDp0ZXN0LXBhc3N3b3JkLTEyMw=="
        }
      }
    }
```

**Note**: The `auth` field is the base64-encoded string of `username:password`. Generate it with:

```bash
echo -n "test-build:test-password-123" | base64
```

Apply the secret:

```bash
kubectl apply -f registry-docker-config.yaml
```

### Step 4: Copy Registry CA Certificate

Since the registry uses a self-signed certificate, build pods need to trust the CA certificate:

```bash
# Extract the CA certificate from the registry namespace
kubectl get secret registry-tls -n registry -o jsonpath='{.data.ca\.crt}' | base64 -d > /tmp/registry-ca.crt

# Create a secret in the build namespace
kubectl create secret generic registry-ca-cert \
  -n test-build \
  --from-file=ca.crt=/tmp/registry-ca.crt
```

Or using YAML:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-ca-cert
  namespace: test-build
type: Opaque
data:
  ca.crt: <base64-encoded-ca-certificate>
```

### Step 5: Create RBAC Resources (Optional)

If your build pods need to read from the registry, create appropriate RBAC resources:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: buildkit
  namespace: test-build
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: buildkit-registry-reader
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
    resourceNames: ["registry-docker-config", "registry-ca-cert"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: buildkit-registry-reader
subjects:
  - kind: ServiceAccount
    name: buildkit
    namespace: test-build
roleRef:
  kind: ClusterRole
  name: buildkit-registry-reader
  apiGroup: rbac.authorization.k8s.io
```

### Step 6: Deploy BuildKit Pod

Create a BuildKit pod that can push images to the registry:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: buildkit-test
  namespace: test-build
spec:
  serviceAccountName: buildkit
  containers:
  - name: buildkitd
    image: moby/buildkit:v0.12.5
    args:
      - --addr
      - unix:///run/buildkit/buildkitd.sock
    securityContext:
      privileged: true  # Required for BuildKit
    volumeMounts:
      # Mount Docker config for authentication
      - name: docker-config
        mountPath: /root/.docker
        readOnly: true
      # Mount registry CA certificate for TLS trust
      - name: registry-ca
        mountPath: /etc/ssl/certs/registry-ca.crt
        subPath: ca.crt
        readOnly: true
  volumes:
    - name: docker-config
      secret:
        secretName: registry-docker-config
        items:
          - key: .dockerconfigjson
            path: config.json  # Must be named config.json
    - name: registry-ca
      secret:
        secretName: registry-ca-cert
```

Apply the pod:

```bash
kubectl apply -f buildkit-pod.yaml
```

Wait for the pod to be ready:

```bash
kubectl wait --for=condition=ready pod/buildkit-test -n test-build --timeout=60s
```

### Step 7: Test Image Build and Push

Create a test Dockerfile inside the BuildKit pod:

```bash
kubectl exec -n test-build buildkit-test -- sh -c 'mkdir -p /tmp/test-build && cat > /tmp/test-build/Dockerfile <<EOF
FROM alpine:latest
RUN echo "Hello from BuildKit test"
CMD ["/bin/sh"]
EOF'
```

Build and push the image:

```bash
kubectl exec -n test-build buildkit-test -- buildctl \
  --addr unix:///run/buildkit/buildkitd.sock \
  build \
  --frontend dockerfile.v0 \
  --local context=/tmp/test-build \
  --local dockerfile=/tmp/test-build \
  --output type=image,name=registry.registry.svc.cluster.local/test-build/myapp:latest,push=true
```

**Expected Output**:

```
#1 [internal] load build definition from Dockerfile
#1 transferring dockerfile: 108B done
#1 DONE 0.0s

#2 [internal] load metadata for docker.io/library/alpine:latest
#2 DONE 1.5s

#3 [internal] load .dockerignore
#3 transferring context: 2B done
#3 DONE 0.0s

#4 [1/2] FROM docker.io/library/alpine:latest
#4 DONE 1.7s

#5 [2/2] RUN echo "Hello from BuildKit test"
#5 0.031 Hello from BuildKit test
#5 DONE 0.1s

#6 exporting to image
#6 exporting layers done
#6 exporting manifest sha256:2dfa91aad1877a0b0bf2a2cf703bbedc06a50eef772b9dc6cda8e152b03fe677 done
#6 exporting config sha256:b62755ffcdbad1024fbf244a0a7e21c0881047130c1b0fbdeeb4f05eb066c808 done
#6 pushing layers done
#6 pushing manifest for registry.registry.svc.cluster.local/test-build/myapp:latest@sha256:... done
#6 DONE 0.1s

#7 [auth] test-build/myapp:pull,push token for registry.registry.svc.cluster.local
#7 DONE 0.0s
```

### Step 8: Verify Image in Registry

Check the registry storage:

```bash
kubectl exec -n registry deployment/registry -- ls -la /var/lib/registry/docker/registry/v2/repositories/
```

You should see a directory named `test-build` containing your repository.

---

## Troubleshooting

### Common Issues and Solutions

#### 1. "failed to verify token: invalid token"

**Symptoms**:
- Registry logs show: `level=info msg="failed to verify token: invalid token"`
- Image push fails with authentication errors

**Root Causes**:
- Missing JWKS configuration (Docker Registry v3.0.0 requirement)
- JWT tokens missing `kid` header
- JWKS key ID doesn't match JWT token kid header
- JWKS file not mounted or incorrect path

**Solutions**:
1. Verify JWKS secret exists:
   ```bash
   kubectl get secret registry-auth-keys-jwks -n registry
   ```

2. Check registry config includes jwks path:
   ```bash
   kubectl get configmap registry-config -n registry -o yaml | grep jwks
   ```

3. Verify JWKS is mounted in registry pod:
   ```bash
   kubectl describe pod -n registry -l app=registry | grep jwks -A 5
   ```

4. Check JWT token includes kid header:
   ```bash
   # Get a token and decode the header
   kubectl run test-curl --image=curlimages/curl:latest --rm -i --restart=Never -- \
     curl -s -u test-build:test-password-123 \
     "http://registry-auth.registry.svc.cluster.local/auth?service=docker-registry&scope=repository:test-build/myapp:push,pull" \
     | jq -r '.token' | cut -d'.' -f1 | base64 -d 2>/dev/null | jq .

   # Should show: {"alg":"RS256","kid":"registry-auth-jwt-signer","typ":"JWT"}
   ```

#### 2. "authorization token required"

**Symptoms**:
- Registry logs show: `level=warning msg="error authorizing context: authorization token required"`
- Image push fails before authentication

**Root Causes**:
- Docker config not mounted correctly in build pod
- Wrong registry hostname in Docker config
- Credentials not in correct format

**Solutions**:
1. Verify Docker config secret exists and has correct format:
   ```bash
   kubectl get secret registry-docker-config -n test-build -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d | jq .
   ```

2. Check Docker config is mounted at correct path:
   ```bash
   kubectl exec -n test-build buildkit-test -- cat /root/.docker/config.json
   ```

3. Verify hostname matches registry service:
   - Use: `registry.registry.svc.cluster.local` (no port)
   - NOT: `registry.registry.svc.cluster.local:443`

#### 3. "x509: certificate signed by unknown authority"

**Symptoms**:
- TLS verification errors when connecting to registry
- Build fails with certificate errors

**Root Causes**:
- Registry CA certificate not trusted by build pod
- CA certificate not mounted correctly
- Wrong CA certificate copied

**Solutions**:
1. Verify CA certificate secret exists:
   ```bash
   kubectl get secret registry-ca-cert -n test-build
   ```

2. Check CA certificate is mounted:
   ```bash
   kubectl exec -n test-build buildkit-test -- cat /etc/ssl/certs/registry-ca.crt
   ```

3. Verify CA certificate matches registry certificate:
   ```bash
   # Get registry CA
   kubectl get secret registry-tls -n registry -o jsonpath='{.data.ca\.crt}' | base64 -d > /tmp/registry-ca.crt

   # Get CA from build namespace
   kubectl get secret registry-ca-cert -n test-build -o jsonpath='{.data.ca\.crt}' | base64 -d > /tmp/build-ca.crt

   # Compare
   diff /tmp/registry-ca.crt /tmp/build-ca.crt
   ```

#### 4. "invalid credentials"

**Symptoms**:
- Registry-auth logs show: `level=warning msg="auth: invalid credentials for namespace=test-build"`
- Token request returns 401 Unauthorized

**Root Causes**:
- Credentials in Docker config don't match `<namespace>-registry-credentials` secret
- Wrong secret name pattern
- Username doesn't match namespace name

**Solutions**:
1. Verify credentials secret exists with correct name:
   ```bash
   kubectl get secret test-build-registry-credentials -n test-build
   ```

2. Check username matches namespace:
   ```bash
   kubectl get secret test-build-registry-credentials -n test-build -o jsonpath='{.data.username}' | base64 -d
   # Should output: test-build
   ```

3. Verify credentials match between secrets:
   ```bash
   # Get credentials from registry secret
   echo "Username: $(kubectl get secret test-build-registry-credentials -n test-build -o jsonpath='{.data.username}' | base64 -d)"
   echo "Password: $(kubectl get secret test-build-registry-credentials -n test-build -o jsonpath='{.data.password}' | base64 -d)"

   # Get credentials from Docker config
   kubectl get secret registry-docker-config -n test-build -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d | jq -r '.auths["registry.registry.svc.cluster.local"] | .username, .password'
   ```

#### 5. "failed to get credentials secret"

**Symptoms**:
- Registry-auth logs show: `auth: failed to get credentials secret test-build/test-build-registry-credentials: not found`

**Root Causes**:
- Credentials secret doesn't exist
- Secret not in correct namespace
- Wrong secret name

**Solutions**:
1. Create credentials secret with correct name pattern:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: test-build-registry-credentials  # Pattern: <namespace>-registry-credentials
     namespace: test-build
   type: Opaque
   stringData:
     username: test-build  # Must match namespace name
     password: your-secure-password
   ```

2. Verify registry-auth has RBAC permissions to read secrets:
   ```bash
   kubectl describe clusterrole registry-auth-secret-reader
   ```

#### 6. "could not find /tmp/test-build: no such file or directory"

**Symptoms**:
- BuildKit build fails with file not found errors

**Root Causes**:
- Dockerfile not created in build pod
- Wrong path specified

**Solutions**:
1. Create Dockerfile in build pod:
   ```bash
   kubectl exec -n test-build buildkit-test -- sh -c 'mkdir -p /tmp/test-build && cat > /tmp/test-build/Dockerfile <<EOF
   FROM alpine:latest
   RUN echo "Hello from test"
   EOF'
   ```

2. Verify Dockerfile exists:
   ```bash
   kubectl exec -n test-build buildkit-test -- cat /tmp/test-build/Dockerfile
   ```

#### 7. Registry Pod CrashLoopBackOff

**Symptoms**:
- Registry pod keeps restarting
- Error: `parsing environment variable REGISTRY_AUTH_PORT: yaml: unmarshal errors`

**Root Causes**:
- Kubernetes auto-injecting service environment variables that conflict with registry config

**Solution**:
Add `enableServiceLinks: false` to pod spec:

```yaml
spec:
  template:
    spec:
      enableServiceLinks: false  # Prevents env var pollution
      containers:
        - name: registry
          # ...
```

#### 8. "config file not found" or Registry Using HTTP Instead of HTTPS

**Symptoms**:
- Registry logs don't show TLS configuration
- Registry serving plain HTTP on port 5000
- Config file not loaded

**Root Causes**:
- Config mounted at wrong path
- Docker Registry v3.0.0 expects config at `/etc/distribution/config.yml`

**Solution**:
Ensure config is mounted at correct path:

```yaml
volumeMounts:
  - name: registry-config
    mountPath: /etc/distribution  # v3.0.0 path, NOT /etc/docker/registry
    readOnly: true
```

### Diagnostic Commands

#### Check Registry Pod Status

```bash
kubectl get pods -n registry
kubectl describe pod -n registry -l app=registry
kubectl logs -n registry deployment/registry --tail=100
```

#### Check Registry-Auth Pod Status

```bash
kubectl get pods -n registry -l app=registry-auth
kubectl logs -n registry deployment/registry-auth --tail=100
```

#### Test JWT Token Generation

```bash
kubectl run test-curl --image=curlimages/curl:latest --rm -i --restart=Never -- \
  curl -v -u test-build:test-password-123 \
  "http://registry-auth.registry.svc.cluster.local/auth?service=docker-registry&scope=repository:test-build/myapp:push,pull"
```

#### Verify Registry Configuration

```bash
kubectl get configmap registry-config -n registry -o yaml
```

#### Check Certificate Status

```bash
kubectl get certificate -n registry
kubectl describe certificate registry-auth-keys -n registry
kubectl describe certificate registry-tls -n registry
```

#### Verify JWKS Secret

```bash
kubectl get secret registry-auth-keys-jwks -n registry -o jsonpath='{.data.jwks\.json}' | base64 -d | jq .
```

#### Test Registry Connectivity from Build Pod

```bash
kubectl exec -n test-build buildkit-test -- wget --spider --no-check-certificate https://registry.registry.svc.cluster.local/v2/
```

---

## Security Considerations

### Credential Management

1. **Strong Passwords**: Always use strong, randomly generated passwords for registry credentials
   ```bash
   openssl rand -base64 32
   ```

2. **Secret Rotation**: Regularly rotate registry credentials and JWT signing keys

3. **Namespace Isolation**: Each namespace has its own credentials, preventing cross-namespace access

4. **Constant-Time Comparison**: Password validation uses `subtle.ConstantTimeCompare` to prevent timing attacks

### TLS/Certificate Management

1. **Self-Signed Certificates**: The registry uses self-signed certificates managed by cert-manager
   - Suitable for internal/development use
   - For production, consider using a trusted CA

2. **Certificate Rotation**: Certificates are automatically renewed by cert-manager 30 days before expiration

3. **Private Key Protection**: Private keys are stored in Kubernetes secrets with restricted access

### JWT Token Security

1. **RS256 Algorithm**: Uses RSA signature with SHA-256, providing strong cryptographic security

2. **Token Expiration**: Tokens expire after 5 minutes (300 seconds) by default
   - Configurable via `TOKEN_EXPIRATION_SEC` environment variable

3. **Unique Token IDs**: Each token has a unique `jti` (JWT ID) claim to prevent replay attacks

4. **Audience Validation**: Registry validates the `aud` claim matches "docker-registry"

5. **Issuer Validation**: Registry validates the `iss` claim matches "registry-token-issuer"

### RBAC Permissions

The registry-auth service requires cluster-wide secret read permissions to validate credentials across all namespaces:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: registry-auth-secret-reader
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
```

**Security Note**: This is a privileged permission. The registry-auth service should be carefully audited and monitored.

### Network Security

1. **TLS Encryption**: All registry communication is encrypted using TLS

2. **ClusterIP Services**: Registry services are only accessible within the cluster

3. **No External Exposure**: The registry is not exposed outside the cluster by default

### Best Practices

1. **Use Strong Credentials**: Generate credentials with sufficient entropy
2. **Rotate Secrets Regularly**: Implement a secret rotation policy
3. **Monitor Access**: Enable audit logging for registry access
4. **Limit RBAC**: Grant minimal necessary permissions to service accounts
5. **Use NetworkPolicies**: Restrict network access between namespaces
6. **Enable Image Scanning**: Scan pushed images for vulnerabilities
7. **Implement Resource Quotas**: Prevent registry storage exhaustion

---

## Additional Notes

### Namespace Naming Requirements

- Repository paths **MUST** start with the namespace name
- Example: namespace `test-build` can only push to `test-build/*` repositories
- Attempting to push to `other-namespace/image` will fail with unauthorized error

### Storage Considerations

- Registry storage is backed by a PersistentVolume
- Default size: 100GB (production), 10Gi (e2e)
- Storage class should support ReadWriteMany if running multiple registry replicas

### Performance Tuning

Registry configuration options for performance:

```yaml
storage:
  cache:
    blobdescriptor: inmemory  # Cache blob descriptors in memory
  delete:
    enabled: true  # Allow image deletion
```

### Backup and Recovery

To backup registry images:

1. **PVC Backup**: Backup the `registry-storage` PersistentVolumeClaim
2. **Image Export**: Export images using `docker save` or `skopeo copy`

### Migration from v2.x to v3.0.0

If migrating from Docker Registry v2.x:

1. ✅ Generate JWKS file from existing certificates
2. ✅ Create `registry-auth-keys-jwks` secret
3. ✅ Update registry config to add `jwks` parameter
4. ✅ Update registry deployment to mount JWKS secret
5. ✅ Update registry-auth to include `kid` header in JWT tokens
6. ✅ Rebuild and redeploy registry-auth service
7. ✅ Test token validation before rolling out

### Monitoring and Observability

Registry provides debug-level logging. To view authentication events:

```bash
kubectl logs -n registry deployment/registry --tail=500 | grep -i "auth\|token"
```

Registry-auth service logs:

```bash
kubectl logs -n registry deployment/registry-auth --tail=500
```

### References

- [Docker Distribution Documentation](https://distribution.github.io/distribution/)
- [JWT Token Authentication Spec](https://distribution.github.io/distribution/spec/auth/jwt/)
- [Docker Registry v3.0.0 Release Notes](https://github.com/distribution/distribution/releases/tag/v3.0.0)
- [GitHub Issue: Token Auth requires JWKS](https://github.com/distribution/distribution/issues/4470)
- [GitHub Issue: JWT validation problem on 3.0.0-rc.1](https://github.com/distribution/distribution/issues/4533)

---

**Document Version**: 1.0
**Last Updated**: 2025-09-30
**Docker Registry Version**: 3.0.0
**Operator Version**: v0.1.2
