# End-to-End Tests

This directory contains end-to-end tests for the Kibaship operator that run against a real Kubernetes cluster.

## Directory Structure

```
end-to-end-tests/
├── manifests/          # Kustomize project with test resources
│   ├── kustomization.yaml
│   ├── project.yaml
│   ├── environments.yaml
│   ├── applications.yaml
│   └── deployments.yaml
├── run.sh             # Main test execution script
├── kubeconfig         # Place your cluster kubeconfig here (not committed)
└── README.md          # This file
```

## Prerequisites

1. **Docker Hub Account**
   - You need a Docker Hub account with push permissions
   - Create a personal access token at https://hub.docker.com/settings/security

2. **Remote Kubernetes Cluster**
   - A running Kubernetes cluster (not Kind)
   - Operator and dependencies already installed
   - `kubeconfig` file placed in this directory

3. **Required Tools**
   - `docker` - For building and pushing images
   - `kubectl` - For cluster operations
   - `kustomize` - For manifest generation
   - `jq` - For JSON processing

## Usage

### 1. Set up credentials

```bash
export DOCKERHUB_USERNAME="your-username"
export DOCKERHUB_TOKEN="your-access-token"
```

### 2. Place kubeconfig

Copy your cluster's kubeconfig to this directory:

```bash
cp ~/.kube/config ./kubeconfig
# OR if using a specific cluster config
cp /path/to/your/cluster-config.yaml ./kubeconfig
```

### 3. Run the tests

```bash
./run.sh
```

## What the Script Does

1. **Validation**
   - Checks for Docker Hub credentials
   - Verifies kubeconfig exists and cluster is accessible
   - Validates required tools are installed

2. **Image Building** (Parallel)
   - Operator image
   - API server image
   - Cert-manager webhook image
   - Registry auth service image
   - Railpack CLI image
   - Railpack build image

3. **Image Tagging & Pushing**
   - Tags all images with unique run ID (e.g., `e2e-20250115-143022-a3b4c5d6`)
   - Pushes all images to Docker Hub in parallel
   - Tracks pushed images for cleanup

4. **Cluster Deployment**
   - Applies Kustomize manifests to the cluster
   - Creates:
     - 1 Project
     - 3 Environments (dev, staging, production)
     - 7 Applications (MySQL, MySQLCluster, Valkey, ValkeyCluster, DockerImage, GitRepository, ImageFromRegistry)
     - 7 Deployments (one for each application)

5. **Automatic Cleanup**
   - On failure/abort: Deletes all pushed Docker Hub tags
   - On success: Leaves images for inspection (you can manually clean up)

## Cleanup

### Delete test resources from cluster
```bash
kubectl delete namespace e2e-test-project
```

### Delete Docker Hub tags manually
The script provides automatic cleanup on failure. To manually clean up after success:

```bash
# List images with your run ID
docker images | grep e2e-20250115-143022-a3b4c5d6

# Or use Docker Hub web UI to delete tags
```

## Customization

### Modify test resources

Edit the manifests in `manifests/` to add or modify test cases:

- `project.yaml` - Project configuration
- `environments.yaml` - Environment definitions
- `applications.yaml` - Application types to test
- `deployments.yaml` - Deployment configurations

### Build logs

Build logs are saved in this directory:
- `build-operator.log`
- `build-apiserver.log`
- `build-webhook.log`
- `build-registry-auth.log`
- `build-railpack-cli.log`
- `build-railpack-build.log`
- `push-*.log`

## Troubleshooting

### Build failures
Check the build log files in this directory for details.

### Push failures
- Verify Docker Hub credentials
- Check network connectivity
- Ensure repository exists and you have push permissions

### Cluster connectivity
```bash
export KUBECONFIG=./kubeconfig
kubectl cluster-info
kubectl get nodes
```

### Image cleanup not working
The cleanup uses Docker Hub API. If it fails:
1. Check your credentials
2. Manually delete tags via Docker Hub web UI
3. Or use the Docker Hub CLI

## Security Notes

- **DO NOT commit** the `kubeconfig` file to git (it's in .gitignore)
- **DO NOT commit** Docker Hub credentials
- Use environment variables or a secure secrets manager for credentials
- Consider using a dedicated Docker Hub account for CI/CD
