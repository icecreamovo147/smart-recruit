# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.5

FROM golang:${GO_VERSION}-alpine AS build

ARG SERVICE_DIR
ARG CMD_PATH
ARG BINARY_NAME

WORKDIR /src

RUN apk add --no-cache build-base ca-certificates git tzdata

COPY go.work ./
COPY smart-recruit-platform-go ./smart-recruit-platform-go
COPY smart-recruit-proto ./smart-recruit-proto
COPY web-gin-service ./web-gin-service
COPY logic-grpc-service ./logic-grpc-service
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

FROM alpine:3.22

ARG BINARY_NAME

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S smartrecruit && \
    adduser -S -G smartrecruit -h /app smartrecruit

WORKDIR /app
COPY --from=build "/out/${BINARY_NAME}" /app/service

USER smartrecruit:smartrecruit
ENTRYPOINT ["/app/service"]
