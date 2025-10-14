#!/bin/bash
set -e

# Build Custom kind Node Image with Pre-loaded Images
# This script pulls all infrastructure images and builds a custom kind node image

echo "=========================================="
echo "Building Custom kind Node Image"
echo "=========================================="

# Color output helpers
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"
IMAGES_DIR="${SCRIPT_DIR}/images"

# Configuration
IMAGE_NAME="${IMAGE_NAME:-kibaship/kind-node}"
IMAGE_TAG="${IMAGE_TAG:-v1.31.0-kibaship}"
PUSH_IMAGE="${PUSH_IMAGE:-false}"

# Check prerequisites
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if [ ! -d "${MANIFESTS_DIR}" ]; then
    log_error "Manifests directory not found: ${MANIFESTS_DIR}"
    log_info "Run ./prepare.sh first to generate manifests"
    exit 1
fi

################################################################################
# EXTRACT IMAGE REFERENCES
################################################################################

log_info "Extracting image references from manifests..."

# Create images directory
mkdir -p "${IMAGES_DIR}"

# Extract unique image references (with and without quotes)
IMAGES=$(grep -h "image:" "${MANIFESTS_DIR}"/*.yaml | \
    grep -v "^#" | \
    sed 's/.*image: *//g' | \
    sed 's/^"//g' | \
    sed 's/"$//g' | \
    sort -u)

# Convert to array
IFS=$'\n' read -rd '' -a IMAGE_ARRAY <<<"$IMAGES" || true

log_info "Found ${#IMAGE_ARRAY[@]} unique images to pre-load"

################################################################################
# PULL AND SAVE IMAGES
################################################################################

log_info "Pulling and saving container images..."
echo "This may take 10-20 minutes depending on your internet connection..."

PULLED_COUNT=0
FAILED_COUNT=0
FAILED_IMAGES=()

for image in "${IMAGE_ARRAY[@]}"; do
    # Skip empty lines
    if [ -z "$image" ]; then
        continue
    fi

    # Create safe filename from image reference
    # Replace special chars with underscores
    safe_name=$(echo "$image" | tr '/:@' '_' | sed 's/^_//')
    tar_file="${IMAGES_DIR}/${safe_name}.tar"

    # Skip if already exists
    if [ -f "$tar_file" ]; then
        log_info "Skipping (already exists): $image"
        ((PULLED_COUNT++))
        continue
    fi

    log_info "Pulling: $image"

    # Pull image
    if docker pull "$image" 2>&1 | grep -q "Error"; then
        log_warning "Failed to pull: $image"
        ((FAILED_COUNT++))
        FAILED_IMAGES+=("$image")
        continue
    fi

    # Save image to tar
    log_info "Saving: $safe_name.tar"
    if ! docker save "$image" -o "$tar_file"; then
        log_warning "Failed to save: $image"
        rm -f "$tar_file"
        ((FAILED_COUNT++))
        FAILED_IMAGES+=("$image")
        continue
    fi

    ((PULLED_COUNT++))
    log_success "Saved: $safe_name.tar ($(du -h "$tar_file" | cut -f1))"
done

echo ""
log_info "Pull summary: ${PULLED_COUNT} succeeded, ${FAILED_COUNT} failed"

if [ ${FAILED_COUNT} -gt 0 ]; then
    log_warning "Failed images:"
    for img in "${FAILED_IMAGES[@]}"; do
        echo "  - $img"
    done
fi

################################################################################
# BUILD CUSTOM IMAGE
################################################################################

log_info "Building custom kind node image..."

# Build the image
docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" "${SCRIPT_DIR}"

if [ $? -ne 0 ]; then
    log_error "Failed to build image"
    exit 1
fi

log_success "Custom kind node image built: ${IMAGE_NAME}:${IMAGE_TAG}"

# Show image size
IMAGE_SIZE=$(docker images "${IMAGE_NAME}:${IMAGE_TAG}" --format "{{.Size}}")
log_info "Image size: ${IMAGE_SIZE}"

################################################################################
# TAG AS LATEST
################################################################################

log_info "Tagging as latest..."
docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:latest"

################################################################################
# OPTIONAL: PUSH TO REGISTRY
################################################################################

if [ "$PUSH_IMAGE" = "true" ]; then
    log_info "Pushing image to registry..."

    log_info "Pushing: ${IMAGE_NAME}:${IMAGE_TAG}"
    docker push "${IMAGE_NAME}:${IMAGE_TAG}"

    log_info "Pushing: ${IMAGE_NAME}:latest"
    docker push "${IMAGE_NAME}:latest"

    log_success "Image pushed to registry"
fi

################################################################################
# CLEANUP
################################################################################

log_info "Cleaning up image tars..."
rm -rf "${IMAGES_DIR}"
log_success "Image tars removed (saved space)"

################################################################################
# SUMMARY
################################################################################

echo ""
log_success "=========================================="
log_success "Build Complete!"
log_success "=========================================="
echo ""
log_info "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
log_info "Size: ${IMAGE_SIZE}"
log_info "Pre-loaded images: ${PULLED_COUNT}"
echo ""
log_info "Usage:"
echo "  kind create cluster --image ${IMAGE_NAME}:${IMAGE_TAG}"
echo ""
log_info "Or specify in kind-config.yaml:"
echo "  nodes:"
echo "  - role: control-plane"
echo "    image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo ""

if [ "$PUSH_IMAGE" != "true" ]; then
    log_warning "Image not pushed to registry (use PUSH_IMAGE=true to push)"
    log_info "To push later, run:"
    echo "  docker push ${IMAGE_NAME}:${IMAGE_TAG}"
fi

echo ""
log_success "You can now create a kind cluster with pre-loaded images!"
