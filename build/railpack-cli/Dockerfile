# Minimal Railpack CLI image for prepare step (v0.7.2)
# Multi-arch capable via buildx; manual installation to avoid script bugs

FROM alpine:3.20 AS downloader
ENV RAILPACK_VERSION=0.7.2
RUN apk add --no-cache curl ca-certificates && update-ca-certificates

# Manual installation to avoid bugs in the official installer script
RUN ARCH=$(uname -m | tr '[:upper:]' '[:lower:]') && \
    case "${ARCH}" in \
        amd64|x86_64) ARCH="x86_64" ;; \
        arm64|aarch64) ARCH="arm64" ;; \
        *) ARCH="x86_64" ;; \
    esac && \
    curl -sSL "https://github.com/railwayapp/railpack/releases/download/v${RAILPACK_VERSION}/railpack-v${RAILPACK_VERSION}-${ARCH}-unknown-linux-musl.tar.gz" | \
    tar -xzf - -C /usr/local/bin

FROM alpine:3.20
# Copy only the railpack binary and ca-certs for TLS
COPY --from=downloader /usr/local/bin/railpack /usr/local/bin/railpack
RUN apk add --no-cache ca-certificates && update-ca-certificates \
    && adduser -D -u 10001 app && chown 10001:0 /usr/local/bin/railpack \
    && chmod 0755 /usr/local/bin/railpack
USER 10001
ENTRYPOINT ["/usr/local/bin/railpack"]
CMD ["--help"]

