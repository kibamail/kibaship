# kind Cluster Setup for Kibaship E2E Testing

This directory contains scripts, Dockerfile, and manifests for quickly setting up a kind cluster with all required infrastructure pre-staged.

## Quick Start

**To build and use the custom kind image (recommended):**

```bash
# 1. Generate manifests
./kind/prepare.sh

# 2. Build custom kind image (10-15 minutes, one-time)
./kind/build.sh

# 3. Create cluster with custom image (~30 seconds)
kind create cluster --name kibaship --image kibaship/kind-node:v1.31.0-kibaship

# 4. Apply manifests (~2 minutes)
kubectl apply -f kind/manifests/gateway-crds-v-1.2.0.yaml
kubectl apply -f kind/manifests/cilium-v-1.18.2.yaml
kubectl apply -f kind/manifests/cert-manager-v-1.18.2.yaml
kubectl apply -f kind/manifests/prometheus-operator-v-0.77.1.yaml
kubectl apply -f kind/manifests/tekton-pipelines-v-1.4.0.yaml
kubectl apply -f kind/manifests/valkey-operator-v-0.0.59.yaml
kubectl apply -f kind/manifests/mysql-operator-v-9.4.0-2.2.5.yaml
kubectl apply -f kind/manifests/storage-classes.yaml
```

**Result:** Full e2e testing environment ready in ~2-3 minutes (after initial build)!

---

## Two Approaches

### Approach 1: Custom kind Image (Recommended for CI)
Build a custom kind node image with all container images pre-loaded. **Fastest startup!**

### Approach 2: Apply Manifests (Quick for local dev)
Use standard kind image and apply pre-generated manifests.

## Comparison

| Feature | Approach 1: Custom Image | Approach 2: Standard Image |
|---------|-------------------------|---------------------------|
| **Setup Time** | 2-3 minutes | 8-12 minutes |
| **Image Pull Time** | 0 (pre-loaded) | 5-10 minutes |
| **Initial Build** | 10-20 minutes (one-time) | N/A |
| **Disk Space** | ~3-4GB (custom image) | ~1GB (standard) |
| **CI Friendly** | ⭐⭐⭐⭐⭐ Excellent | ⭐⭐⭐ Good |
| **Offline Capable** | ✅ Yes (after build) | ⚠️ Partial (manifests only) |
| **Best For** | CI/CD, repeated testing | Quick local development |

---

## Approach 1: Custom kind Image with Pre-loaded Images

This approach builds a custom kind node image based on `kindest/node:v1.34.0` with all infrastructure container images pre-loaded. This eliminates image pulling during cluster setup.

### Benefits
- ⚡ **Fastest** - No image pulling during cluster creation
- 🔒 **Deterministic** - All images embedded in the custom image
- 📦 **Portable** - Push to registry and use anywhere
- 🚀 **CI-ready** - Perfect for CI/CD pipelines

### Prerequisites
- Docker installed
- 10-20 minutes for initial build
- ~15-20GB disk space (temporary)

### Step 1: Prepare Manifests

Generate all Kubernetes manifests:

```bash
./kind/prepare.sh
```

This creates `kind/manifests/` with:
- Cilium CNI (v1.18.2)
- Gateway API CRDs (v1.2.0)
- cert-manager (v1.18.2)
- Prometheus Operator (v0.77.1)
- Tekton Pipelines (v1.4.0)
- Valkey Operator (v0.0.59)
- MySQL Operator (v9.4.0-2.2.5)
- Storage Classes

### Step 2: Build Custom Image

Build the custom kind node image with pre-loaded infrastructure images:

```bash
./kind/build.sh
```

This will:
1. Extract all image references from manifests (~50-60 images)
2. Pull each image with Docker
3. Save images as tar files
4. Build custom kind node image: `kibaship/kind-node:v1.31.0-kibaship`
5. Clean up tar files

**Time:** 10-20 minutes (first time)
**Size:** ~3-4GB final image

### Step 3: Create Cluster with Custom Image

```bash
# Using command line
kind create cluster --name kibaship --image kibaship/kind-node:v1.31.0-kibaship

# Or using config file (recommended)
cat > kind-cluster-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kibaship/kind-node:v1.31.0-kibaship
EOF

kind create cluster --name kibaship --config kind-cluster-config.yaml
```

### Step 4: Apply Manifests

All images are already loaded, so this is **fast**:

```bash
kubectl apply -f kind/manifests/gateway-crds-v-1.2.0.yaml
kubectl apply -f kind/manifests/cilium-v-1.18.2.yaml
kubectl apply -f kind/manifests/cert-manager-v-1.18.2.yaml
kubectl apply -f kind/manifests/prometheus-operator-v-0.77.1.yaml
kubectl apply -f kind/manifests/tekton-pipelines-v-1.4.0.yaml
kubectl apply -f kind/manifests/valkey-operator-v-0.0.59.yaml
kubectl apply -f kind/manifests/mysql-operator-v-9.4.0-2.2.5.yaml
kubectl apply -f kind/manifests/storage-classes.yaml
```

**Time:** ~2-3 minutes (vs 10-15 without pre-loaded images!)

### Optional: Push to Registry

To share the custom image with your team or CI:

```bash
# Tag for your registry
docker tag kibaship/kind-node:v1.31.0-kibaship myregistry.com/kibaship/kind-node:v1.31.0-kibaship

# Push
docker push myregistry.com/kibaship/kind-node:v1.31.0-kibaship

# Or push directly during build
PUSH_IMAGE=true IMAGE_NAME=myregistry.com/kibaship/kind-node ./kind/build.sh
```

---

## Approach 2: Standard kind Image with Manifests (Quick Local Setup)

If you don't want to build a custom image, use the standard kind image and apply manifests:

### Step 1: Prepare Manifests

```bash
./kind/prepare.sh
```

### Step 2: Create Standard kind Cluster

```bash
kind create cluster --name kibaship
```

### Step 3: Apply Manifests

Apply the pre-generated manifests to quickly set up infrastructure:

```bash
# Apply in order
kubectl apply -f kind/manifests/gateway-crds-v-1.2.0.yaml
kubectl apply -f kind/manifests/cilium-v-1.18.2.yaml
kubectl apply -f kind/manifests/cert-manager-v-1.18.2.yaml
kubectl apply -f kind/manifests/prometheus-operator-v-0.77.1.yaml
kubectl apply -f kind/manifests/tekton-pipelines-v-1.4.0.yaml
kubectl apply -f kind/manifests/valkey-operator-v-0.0.59.yaml
kubectl apply -f kind/manifests/mysql-operator-v-9.4.0-2.2.5.yaml
kubectl apply -f kind/manifests/storage-classes.yaml
```

## Benefits of Pre-staged Manifests

### Without Pre-staging
- Download manifests during setup: ~2-3 minutes
- Network-dependent (fails if offline)
- Non-deterministic (remote files can change)

### With Pre-staging
- Apply from local files: ~30-60 seconds
- Works offline
- Deterministic (manifests are versioned in repo)

## Directory Structure

```
kind/
├── README.md           # This file
├── prepare.sh          # Script to download/generate manifests
└── manifests/          # Generated manifests (created by prepare.sh)
    ├── cilium-v-1.18.2.yaml
    ├── gateway-crds-v-1.2.0.yaml
    ├── cert-manager-v-1.18.2.yaml
    ├── prometheus-operator-v-0.77.1.yaml
    ├── tekton-pipelines-v-1.4.0.yaml
    ├── valkey-operator-v-0.0.59.yaml
    ├── mysql-operator-v-9.4.0-2.2.5.yaml
    └── storage-classes.yaml
```

## Updating Manifests

When component versions change:

1. Edit `prepare.sh` to update versions
2. Run `./kind/prepare.sh` to regenerate manifests
3. Commit the updated manifests to git

## Component Versions

| Component | Version | Source |
|-----------|---------|--------|
| Cilium | 1.18.2 | Helm chart (cilium.io) |
| Gateway API | 1.2.0 | kubernetes-sigs/gateway-api |
| cert-manager | 1.18.2 | cert-manager/cert-manager |
| Prometheus Operator | 0.77.1 | prometheus-operator/prometheus-operator |
| Tekton Pipelines | 1.4.0 | Google Cloud Storage |
| Valkey Operator | 0.0.59 | hyperspike/valkey-operator |
| MySQL Operator | 9.4.0-2.2.5 | mysql/mysql-operator |

## Development Workflow

### Initial Setup
```bash
# 1. Prepare manifests (one-time or when versions change)
./kind/prepare.sh

# 2. Create and configure cluster (TBD - will be automated)
kind create cluster --name kibaship
kubectl apply -f kind/manifests/

# 3. Run tests
make test-e2e
```

### Iterative Testing
```bash
# 1. Make code changes
# 2. Build images
# 3. Load into kind
# 4. Run tests
# 5. Repeat
```

### Reset Cluster
```bash
# Quick reset - delete and recreate
kind delete cluster --name kibaship
kind create cluster --name kibaship
kubectl apply -f kind/manifests/
```

## CI Integration

In CI, you can:
1. Cache `kind/manifests/` directory
2. Apply manifests directly without re-downloading
3. Save 2-3 minutes per CI run

Example GitHub Actions:
```yaml
- name: Cache kind manifests
  uses: actions/cache@v3
  with:
    path: kind/manifests
    key: manifests-${{ hashFiles('kind/prepare.sh') }}

- name: Prepare manifests if not cached
  run: |
    if [ ! -d "kind/manifests" ]; then
      ./kind/prepare.sh
    fi
```

## Troubleshooting

### Helm not found
```bash
brew install helm  # macOS
```

### Manifests out of date
```bash
rm -rf kind/manifests
./kind/prepare.sh
```

### Network issues during prepare
The script requires internet access to download manifests. Run it once, commit the manifests, and they'll work offline afterward.
