# Minimal Railpack CLI image for prepare step (v0.7.2)
# Switch to glibc-based base image to ensure runtime tools (e.g., mise) execute correctly

FROM debian:bookworm-slim AS downloader
ENV RAILPACK_VERSION=0.7.2
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Manual installation of Railpack CLI (musl binary still works on Debian)
RUN set -eux; \
    ARCH="$(uname -m | tr '[:upper:]' '[:lower:]')"; \
    case "$ARCH" in \
        amd64|x86_64) ARCH="x86_64" ;; \
        arm64|aarch64) ARCH="arm64" ;; \
        *) ARCH="x86_64" ;; \
    esac; \
    curl -sSL "https://github.com/railwayapp/railpack/releases/download/v${RAILPACK_VERSION}/railpack-v${RAILPACK_VERSION}-${ARCH}-unknown-linux-musl.tar.gz" \
      | tar -xzf - -C /usr/local/bin

FROM debian:bookworm-slim
# Copy only the railpack binary and ca-certs for TLS
COPY --from=downloader /usr/local/bin/railpack /usr/local/bin/railpack
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && update-ca-certificates \
    && useradd -u 10001 -m -s /usr/sbin/nologin app \
    && chown 10001:0 /usr/local/bin/railpack \
    && chmod 0755 /usr/local/bin/railpack \
    && rm -rf /var/lib/apt/lists/*
USER 10001
ENTRYPOINT ["/usr/local/bin/railpack"]
CMD ["--help"]
