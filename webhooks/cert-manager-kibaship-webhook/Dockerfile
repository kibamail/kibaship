# Multi-stage build for Kibaship cert-manager DNS01 webhook

# --- Builder stage ---
FROM golang:1.23 AS builder
WORKDIR /src

# Enable reproducible builds
ENV CGO_ENABLED=0

# Pre-cache dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

# Build for linux; allow overriding arch via buildx
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH}

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/cert-manager-kibaship-webhook .

# --- Runtime stage ---
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/cert-manager-kibaship-webhook /cert-manager-kibaship-webhook
USER 65532:65532
ENTRYPOINT ["/cert-manager-kibaship-webhook"]

