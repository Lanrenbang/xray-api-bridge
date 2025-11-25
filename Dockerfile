# syntax=docker/dockerfile:latest

# General base layer
FROM --platform=$BUILDPLATFORM golang:alpine AS base
ARG TARGETOS TARGETARCH
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0
## Shared Go cache
ENV GOMODCACHE=/go/pkg/mod GOCACHE=/root/.cache/go-build
RUN apk add --no-cache ca-certificates git tzdata jq && \
  update-ca-certificates && \
  adduser -D -u 65532 -h /home/nonroot -s /sbin/nologin nonroot

# envwarp - Get
FROM base AS envwarp-src
ARG SRC_REPO=Lanrenbang/envwarp
ARG SRC_RELEASE=https://api.github.com/repos/${SRC_REPO}/releases/latest \
    SRC_GIT=https://github.com/${SRC_REPO}.git
WORKDIR /src
ADD ${SRC_RELEASE} /tmp/latest-release.json
RUN --mount=type=cache,id=gitcache,target=/root/.cache/git \
    set -eux; \
    SRC_TAG=$(jq -r '.tag_name' /tmp/latest-release.json); \
    if [ -z "$SRC_TAG" ] || [ "$SRC_TAG" = "null" ]; then \
      echo "Error: Failed to get tag_name from GitHub API." >&2; \
      exit 1; \
    fi; \
    echo "Fetching tag: $SRC_TAG"; \
    git init .; \
    git remote add origin "$SRC_GIT"; \
    git fetch --depth=1 origin "$SRC_TAG"; \
    git checkout --detach FETCH_HEAD; \
    if git describe --tags --always 2>/dev/null | grep -qv '^[0-9a-f]\{7\}$'; then \
      echo "Tags found, skipping fetch"; \
    else \
      echo "Fetching full history for tags..."; \
      git fetch --unshallow || true; \
      git fetch --tags --force; \
    fi

# envwarp - Build
FROM base AS envwarp-build
WORKDIR /src
COPY --from=envwarp-src /src/ .
RUN --mount=type=cache,id=gomodcache,target=/go/pkg/mod \
    --mount=type=cache,id=gobuildcache,target=/root/.cache/go-build \
    go build -o /out/envwarp -trimpath -tags=osusergo,netgo -buildvcs=false \
      -ldflags "-X main.version=$(git describe --tags --always --dirty | cut -c2-) -s -w -buildid=" .


# xray-api-bridge - Get
FROM base AS bridge-src
ARG SRC_REPO=Lanrenbang/xray-api-bridge
ARG SRC_RELEASE=https://api.github.com/repos/${SRC_REPO}/releases/latest \
    SRC_GIT=https://github.com/${SRC_REPO}.git
WORKDIR /src
ADD ${SRC_RELEASE} /tmp/latest-release.json
RUN --mount=type=cache,id=gitcache,target=/root/.cache/git \
    set -eux; \
    SRC_TAG=$(jq -r '.tag_name' /tmp/latest-release.json); \
    if [ -z "$SRC_TAG" ] || [ "$SRC_TAG" = "null" ]; then \
      echo "Error: Failed to get tag_name from GitHub API." >&2; \
      exit 1; \
    fi; \
    echo "Fetching tag: $SRC_TAG"; \
    git init .; \
    git remote add origin "$SRC_GIT"; \
    git fetch --depth=1 origin "$SRC_TAG"; \
    git checkout --detach FETCH_HEAD; \
    if git describe --tags --always 2>/dev/null | grep -qv '^[0-9a-f]\{7\}$'; then \
      echo "Tags found, skipping fetch"; \
    else \
      echo "Fetching full history for tags..."; \
      git fetch --unshallow || true; \
      git fetch --tags --force; \
    fi

# xray-api-bridge - Build
FROM base AS bridge-build
WORKDIR /src
COPY --from=bridge-src /src/ .
RUN --mount=type=cache,id=gomodcache,target=/go/pkg/mod \
    --mount=type=cache,id=gobuildcache,target=/root/.cache/go-build \
    go build -o /out/xray-api-bridge -trimpath -tags=osusergo,netgo -buildvcs=false \
      -ldflags "-X xray-api-bridge/bridge.build=$(git describe --tags --always --dirty | cut -c2-) -s -w -buildid=" .

RUN mkdir -p /tmp/etc/templates /tmp/etc/bridge


# Build finally image
FROM scratch

LABEL org.opencontainers.image.title="xray-api-bridge" \
      org.opencontainers.image.authors="bobbynona" \
      org.opencontainers.image.vendor="L.R.B" \
      org.opencontainers.image.source="https://github.com/Lanrenbang/xray-api-bridge" \
      org.opencontainers.image.url="https://github.com/Lanrenbang/xray-api-bridge"

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=envwarp-build --chown=0:0 --chmod=755 /out/envwarp /usr/local/bin/envwarp
COPY --from=bridge-build --chown=0:0 --chmod=755 /out/xray-api-bridge /usr/local/bin/xray-api-bridge

COPY --from=bridge-build --chown=65532:65532 --chmod=0775 /tmp/etc /usr/local/etc/

VOLUME /usr/local/etc/templates

ARG TZ=Etc/UTC
ENV TZ=$TZ

ENTRYPOINT [ "/usr/local/bin/envwarp" ]

