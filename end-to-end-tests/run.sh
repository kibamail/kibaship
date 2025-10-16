#!/usr/bin/env bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Generate unique run ID
RUN_ID="e2e-$(date +%Y%m%d-%H%M%S)-$(echo $RANDOM | md5sum | head -c 8)"
echo -e "${BLUE}==> Run ID: ${RUN_ID}${NC}"

# Global variables
DOCKER_HUB_USERNAME=""
DOCKER_HUB_TOKEN=""
KUBECONFIG_FILE="${SCRIPT_DIR}/kubeconfig"
IMAGES_PUSHED=()
CLEANUP_REQUIRED=false

# Image names and tags
declare -A IMAGES=(
    ["operator"]="kibamail/kibaship"
    ["apiserver"]="kibamail/kibaship-apiserver"
    ["registry-auth"]="kibamail/kibaship-registry-auth"
    ["railpack-cli"]="kibamail/kibaship-railpack-cli"
    ["railpack-build"]="kibamail/kibaship-railpack-build"
)

# Cleanup function
cleanup() {
    local exit_code=$?

    if [ "$CLEANUP_REQUIRED" = true ]; then
        echo -e "\n${YELLOW}==> Cleaning up Docker Hub tags for run ${RUN_ID}...${NC}"

        for image_tag in "${IMAGES_PUSHED[@]}"; do
            echo -e "${YELLOW}  → Deleting ${image_tag}...${NC}"
            delete_dockerhub_tag "$image_tag" || echo -e "${RED}    Failed to delete ${image_tag}${NC}"
        done

        echo -e "${GREEN}==> Cleanup completed${NC}"
    fi

    if [ $exit_code -ne 0 ]; then
        echo -e "\n${RED}==> Script failed with exit code ${exit_code}${NC}"
    fi

    exit $exit_code
}

# Set up trap for cleanup
trap cleanup EXIT INT TERM

# Function to delete a tag from DockerHub
delete_dockerhub_tag() {
    local full_tag=$1
    local repo=$(echo "$full_tag" | cut -d: -f1)
    local tag=$(echo "$full_tag" | cut -d: -f2)
    local repo_name=$(echo "$repo" | cut -d/ -f2)

    # Get auth token
    local token=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"username\": \"${DOCKER_HUB_USERNAME}\", \"password\": \"${DOCKER_HUB_TOKEN}\"}" \
        https://hub.docker.com/v2/users/login/ | jq -r .token)

    if [ -z "$token" ] || [ "$token" = "null" ]; then
        echo -e "${RED}Failed to get Docker Hub auth token${NC}"
        return 1
    fi

    # Delete the tag
    local response=$(curl -s -w "%{http_code}" -o /dev/null -X DELETE \
        -H "Authorization: Bearer ${token}" \
        "https://hub.docker.com/v2/repositories/${DOCKER_HUB_USERNAME}/${repo_name}/tags/${tag}/")

    if [ "$response" -eq 204 ] || [ "$response" -eq 404 ]; then
        echo -e "${GREEN}    ✓ Deleted ${full_tag}${NC}"
        return 0
    else
        echo -e "${RED}    ✗ Failed to delete ${full_tag} (HTTP ${response})${NC}"
        return 1
    fi
}

# Validate prerequisites
echo -e "${BLUE}==> Validating prerequisites...${NC}"

# Check for Docker Hub credentials
if [ -z "${DOCKERHUB_USERNAME:-}" ] || [ -z "${DOCKERHUB_TOKEN:-}" ]; then
    echo -e "${RED}Error: DOCKERHUB_USERNAME and DOCKERHUB_TOKEN environment variables must be set${NC}"
    echo "Example: export DOCKERHUB_USERNAME=myuser DOCKERHUB_TOKEN=mytoken"
    exit 1
fi

DOCKER_HUB_USERNAME="$DOCKERHUB_USERNAME"
DOCKER_HUB_TOKEN="$DOCKERHUB_TOKEN"

echo -e "${GREEN}  ✓ Docker Hub credentials found${NC}"

# Check for kubeconfig
if [ ! -f "$KUBECONFIG_FILE" ]; then
    echo -e "${RED}Error: kubeconfig file not found at ${KUBECONFIG_FILE}${NC}"
    echo "Please place your kubeconfig file in the end-to-end-tests directory"
    exit 1
fi

export KUBECONFIG="$KUBECONFIG_FILE"
echo -e "${GREEN}  ✓ kubeconfig found and set${NC}"

# Verify kubectl connectivity
if ! kubectl cluster-info &>/dev/null; then
    echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
    kubectl cluster-info
    exit 1
fi

echo -e "${GREEN}  ✓ Kubernetes cluster connectivity verified${NC}"

# Check required tools
for tool in docker kubectl kustomize jq; do
    if ! command -v $tool &> /dev/null; then
        echo -e "${RED}Error: ${tool} is not installed${NC}"
        exit 1
    fi
done

echo -e "${GREEN}  ✓ All required tools available${NC}"

# Docker login
echo -e "\n${BLUE}==> Logging into Docker Hub...${NC}"
echo "$DOCKER_HUB_TOKEN" | docker login -u "$DOCKER_HUB_USERNAME" --password-stdin
echo -e "${GREEN}  ✓ Docker Hub login successful${NC}"

# Enable cleanup from this point
CLEANUP_REQUIRED=true

# Build all images in parallel
echo -e "\n${BLUE}==> Building Docker images in parallel...${NC}"

cd "$PROJECT_ROOT"

# Build functions
build_operator() {
    echo -e "${BLUE}  → Building operator image...${NC}"
    docker build -t "${IMAGES[operator]}:${RUN_ID}" -f Dockerfile . &> "${SCRIPT_DIR}/build-operator.log"
    echo -e "${GREEN}  ✓ Operator image built${NC}"
}

build_apiserver() {
    echo -e "${BLUE}  → Building API server image...${NC}"
    docker build -t "${IMAGES[apiserver]}:${RUN_ID}" -f Dockerfile.apiserver . &> "${SCRIPT_DIR}/build-apiserver.log"
    echo -e "${GREEN}  ✓ API server image built${NC}"
}


build_registry_auth() {
    echo -e "${BLUE}  → Building registry auth image...${NC}"
    docker build -t "${IMAGES[registry-auth]}:${RUN_ID}" -f Dockerfile.registry-auth . &> "${SCRIPT_DIR}/build-registry-auth.log"
    echo -e "${GREEN}  ✓ Registry auth image built${NC}"
}

build_railpack_cli() {
    echo -e "${BLUE}  → Building railpack CLI image...${NC}"
    docker build -t "${IMAGES[railpack-cli]}:${RUN_ID}" \
        -f build/railpack-cli/Dockerfile build/railpack-cli &> "${SCRIPT_DIR}/build-railpack-cli.log"
    echo -e "${GREEN}  ✓ Railpack CLI image built${NC}"
}

build_railpack_build() {
    echo -e "${BLUE}  → Building railpack build image...${NC}"
    docker build -t "${IMAGES[railpack-build]}:${RUN_ID}" \
        -f build/railpack-build/Dockerfile build/railpack-build &> "${SCRIPT_DIR}/build-railpack-build.log"
    echo -e "${GREEN}  ✓ Railpack build image built${NC}"
}

# Run builds in parallel
build_operator &
PID_OPERATOR=$!

build_apiserver &
PID_APISERVER=$!

build_registry_auth &
PID_REGISTRY_AUTH=$!

build_railpack_cli &
PID_RAILPACK_CLI=$!

build_railpack_build &
PID_RAILPACK_BUILD=$!

# Wait for all builds to complete
FAILED=0
wait $PID_OPERATOR || { echo -e "${RED}Operator build failed${NC}"; FAILED=1; }
wait $PID_APISERVER || { echo -e "${RED}API server build failed${NC}"; FAILED=1; }
wait $PID_REGISTRY_AUTH || { echo -e "${RED}Registry auth build failed${NC}"; FAILED=1; }
wait $PID_RAILPACK_CLI || { echo -e "${RED}Railpack CLI build failed${NC}"; FAILED=1; }
wait $PID_RAILPACK_BUILD || { echo -e "${RED}Railpack build image failed${NC}"; FAILED=1; }

if [ $FAILED -eq 1 ]; then
    echo -e "\n${RED}==> One or more builds failed. Check log files in ${SCRIPT_DIR}${NC}"
    exit 1
fi

echo -e "\n${GREEN}==> All images built successfully${NC}"

# Push all images in parallel
echo -e "\n${BLUE}==> Pushing Docker images to Docker Hub...${NC}"

push_image() {
    local image_name=$1
    local full_tag="${image_name}:${RUN_ID}"

    echo -e "${BLUE}  → Pushing ${full_tag}...${NC}"
    if docker push "$full_tag" &> "${SCRIPT_DIR}/push-$(basename $image_name).log"; then
        IMAGES_PUSHED+=("$full_tag")
        echo -e "${GREEN}  ✓ Pushed ${full_tag}${NC}"
    else
        echo -e "${RED}  ✗ Failed to push ${full_tag}${NC}"
        return 1
    fi
}

# Push all images in parallel
for image_name in "${IMAGES[@]}"; do
    push_image "$image_name" &
done

# Wait for all pushes
wait

if [ ${#IMAGES_PUSHED[@]} -ne ${#IMAGES[@]} ]; then
    echo -e "\n${RED}==> Some images failed to push${NC}"
    exit 1
fi

echo -e "\n${GREEN}==> All images pushed successfully${NC}"
echo -e "${BLUE}==> Images with tag ${RUN_ID}:${NC}"
for image_tag in "${IMAGES_PUSHED[@]}"; do
    echo -e "  - ${image_tag}"
done

# Apply manifests to cluster
echo -e "\n${BLUE}==> Applying test manifests to cluster...${NC}"
cd "${SCRIPT_DIR}/manifests"

if kustomize build . | kubectl apply -f -; then
    echo -e "${GREEN}  ✓ Manifests applied successfully${NC}"
else
    echo -e "${RED}  ✗ Failed to apply manifests${NC}"
    exit 1
fi

echo -e "\n${GREEN}==> End-to-end test setup completed successfully!${NC}"
echo -e "${BLUE}==> Run ID: ${RUN_ID}${NC}"
echo -e "${YELLOW}==> To clean up, run: kubectl delete namespace e2e-test-project${NC}"
echo -e "${YELLOW}==> Images will be automatically deleted from Docker Hub on script failure${NC}"

# Disable cleanup on success
CLEANUP_REQUIRED=false

echo -e "\n${GREEN}==> Success! Test environment is ready.${NC}"
