# Minimal Railpack CLI image for prepare step (v0.7.2)
# Multi-arch capable via buildx; uses curl installer from official docs

FROM alpine:3.20 AS downloader
ENV RAILPACK_VERSION=0.7.2
RUN apk add --no-cache curl ca-certificates && update-ca-certificates
# Install railpack into /usr/local/bin using official installer
RUN curl -sSL https://railpack.com/install.sh | RAILPACK_VERSION=${RAILPACK_VERSION} sh -s -- --bin-dir /usr/local/bin

FROM alpine:3.20
# Copy only the railpack binary and ca-certs for TLS
COPY --from=downloader /usr/local/bin/railpack /usr/local/bin/railpack
RUN apk add --no-cache ca-certificates && update-ca-certificates \
    && adduser -D -u 10001 app && chown 10001:0 /usr/local/bin/railpack \
    && chmod 0755 /usr/local/bin/railpack
USER 10001
ENTRYPOINT ["/usr/local/bin/railpack"]
CMD ["--help"]

