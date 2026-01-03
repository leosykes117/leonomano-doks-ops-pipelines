# syntax=docker/dockerfile:1.6
FROM alpine:3.23.2

# Tools needed by your uploader script
RUN apk add --no-cache \
  bash \
  git \
  tar \
  ca-certificates \
  curl

# Install MinIO Client (mc) - binary download (official)
# Use TARGETARCH so it works on arm64 (M1) and amd64 if you ever build multi-arch
ARG TARGETARCH
RUN set -eux; \
  case "${TARGETARCH}" in \
  "arm64") MC_ARCH="linux-arm64" ;; \
  "amd64") MC_ARCH="linux-amd64" ;; \
  *) echo "Unsupported TARGETARCH=${TARGETARCH}"; exit 1 ;; \
  esac; \
  curl -fsSL -o /usr/local/bin/mc "https://dl.min.io/client/mc/release/${MC_ARCH}/mc"; \
  chmod +x /usr/local/bin/mc; \
  mc --version

# Your uploader script
COPY ./repo-uploader.sh /usr/local/bin/repo-uploader
RUN chmod +x /usr/local/bin/repo-uploader

WORKDIR /repo
ENTRYPOINT ["repo-uploader"]
