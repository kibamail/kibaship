# Client-only image for BuildKit (buildctl) to be used in Tekton build step
# - Copies buildctl from moby/buildkit:rootless
# - Adds CA certs
# - Runs as non-root user

FROM moby/buildkit:rootless AS bk

FROM debian:bookworm-slim
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates; \
    update-ca-certificates; \
    useradd -u 10001 -m -s /usr/sbin/nologin app; \
    rm -rf /var/lib/apt/lists/*

COPY --from=bk /usr/bin/buildctl /usr/local/bin/buildctl

USER 10001
ENTRYPOINT ["/usr/local/bin/buildctl"]

