# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.5
ARG DEBIAN_VERSION=trixie
ARG DEBIAN_MIRROR=http://mirrors.aliyun.com/debian
ARG DEBIAN_SECURITY_MIRROR=http://mirrors.aliyun.com/debian-security
ARG GOPROXY=https://goproxy.cn,direct

FROM golang:${GO_VERSION}-${DEBIAN_VERSION} AS build

ARG SERVICE_DIR
ARG CMD_PATH
ARG BINARY_NAME
ARG DEBIAN_MIRROR
ARG DEBIAN_SECURITY_MIRROR
ARG GOPROXY

ENV GOPROXY=${GOPROXY}

WORKDIR /src

RUN set -eux; \
    if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
        sed -i "s#http://deb.debian.org/debian-security#${DEBIAN_SECURITY_MIRROR}#g" /etc/apt/sources.list.d/debian.sources; \
        sed -i "s#http://deb.debian.org/debian#${DEBIAN_MIRROR}#g" /etc/apt/sources.list.d/debian.sources; \
    fi && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        git \
        tzdata && \
    rm -rf /var/lib/apt/lists/*

COPY go.work go.work.sum ./
COPY smart-recruit-platform-go ./smart-recruit-platform-go
COPY smart-recruit-proto ./smart-recruit-proto
COPY smart-recruit-commons ./smart-recruit-commons
COPY smart-recruit-gateway ./smart-recruit-gateway
COPY smart-recruit-identity-service ./smart-recruit-identity-service
COPY smart-recruit-recruitment-service ./smart-recruit-recruitment-service
COPY smart-recruit-interview-service ./smart-recruit-interview-service
COPY smart-recruit-offer-service ./smart-recruit-offer-service
COPY smart-recruit-notification-service ./smart-recruit-notification-service
COPY smart-recruit-ai-agent-service ./smart-recruit-ai-agent-service
COPY smart-recruit-analytics-service ./smart-recruit-analytics-service
COPY smart-recruit-worker-service ./smart-recruit-worker-service

RUN test -n "${SERVICE_DIR}" && test -n "${CMD_PATH}" && test -n "${BINARY_NAME}" && \
    cd "${SERVICE_DIR}" && \
    go build -trimpath -ldflags="-s -w" -o "/out/${BINARY_NAME}" "${CMD_PATH}"

FROM debian:${DEBIAN_VERSION}-slim

ARG BINARY_NAME
ARG DEBIAN_MIRROR
ARG DEBIAN_SECURITY_MIRROR

RUN set -eux; \
    if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
        sed -i "s#http://deb.debian.org/debian-security#${DEBIAN_SECURITY_MIRROR}#g" /etc/apt/sources.list.d/debian.sources; \
        sed -i "s#http://deb.debian.org/debian#${DEBIAN_MIRROR}#g" /etc/apt/sources.list.d/debian.sources; \
    fi && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        libstdc++6 \
        tzdata && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system smartrecruit && \
    useradd --system --gid smartrecruit --home-dir /app --create-home smartrecruit

WORKDIR /app
COPY --from=build "/out/${BINARY_NAME}" /app/service
COPY --from=build /src/smart-recruit-commons/config /app/config

USER smartrecruit:smartrecruit
ENTRYPOINT ["/app/service"]
