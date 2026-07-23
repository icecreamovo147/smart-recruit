---
schema_version: 1
id: service-binary-convention
title: Service binary convention runbook
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - services
  - deployment
  - binaries
  - cutover
applies_to:
  - smart-recruit-platform-go/servicebinary/**
  - smart-recruit-*-service/cmd/**
  - smart-recruit-worker-service/**
  - smart-recruit-deploy/**
  - deploy/**
  - docker/**
source_refs:
  - smart-recruit-platform-go/servicebinary/convention.go
  - smart-recruit-platform-go/servicebinary/convention_test.go
  - smart-recruit-gateway/cmd/gateway/main.go
  - smart-recruit-identity-service/cmd/identity-service/main.go
  - smart-recruit-recruitment-service/cmd/recruitment-service/main.go
  - smart-recruit-interview-service/cmd/interview-service/main.go
  - smart-recruit-offer-service/cmd/offer-service/main.go
  - smart-recruit-notification-service/cmd/notification-service/main.go
  - smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go
  - smart-recruit-analytics-service/cmd/analytics-service/main.go
  - smart-recruit-billing-service/cmd/billing-service/main.go
  - smart-recruit-worker-service/cmd/worker-service/main.go
  - start-dev.sh
  - smart-recruit-deploy/docker/go-service.Dockerfile
  - smart-recruit-deploy/docker-compose.microservices.yml
  - deploy/k8s/README-service-binaries.md
  - docker/docker-compose.yml
last_verified: 2026-07-23
review_after: 2026-10-21
---

# Service Binary Convention Runbook

Service unit conventions live in `smart-recruit-platform-go/servicebinary/`. Each independent service has a `cmd/<service>/main.go` entrypoint and service-owned runtime package. Microservice image and composition assets live under `smart-recruit-deploy/`; the full local/container stack lives under `docker/`; Kubernetes manifests and service-binary deployment notes live under `deploy/k8s/`. Review command name, image name, config prefix, health/readiness, internal gRPC token/TLS, metrics, and route-mode defaults before changing a service binary or its deployment surface.

## Independent service entrypoints

| Service root | Main | Default local listen | Notes |
|---|---|---|---|
| `smart-recruit-gateway` | `cmd/gateway/main.go` | HTTP gateway | Dials every domain service, including Billing on `BILLING_GRPC_ADDR` |
| `smart-recruit-identity-service` | `cmd/identity-service/main.go` | `:50061` | Auth, tenants, RBAC |
| `smart-recruit-recruitment-service` | `cmd/recruitment-service/main.go` | `:50062` | Jobs, applications, resumes, candidate profiles |
| `smart-recruit-interview-service` | `cmd/interview-service/main.go` | `:50063` | Interviews and feedback |
| `smart-recruit-offer-service` | `cmd/offer-service/main.go` | `:50064` | Offers |
| `smart-recruit-notification-service` | `cmd/notification-service/main.go` | `:50065` | Notification outbox and delivery |
| `smart-recruit-ai-agent-service` | `cmd/ai-agent-service/main.go` | `:50066` | Agent runtime; settles usage to Billing |
| `smart-recruit-analytics-service` | `cmd/analytics-service/main.go` | `:50067` | Analytics projections |
| `smart-recruit-worker-service` | `cmd/worker-service/main.go` | health `:50068` | Async workers |
| `smart-recruit-billing-service` | `cmd/billing-service/main.go` | `:50069` | Entitlements, ledger, Alipay; requires `--serve` and local billing config |

Local startup builds Billing with `./cmd/billing-service` into `.dev/bin/billing-service` and starts it with `--serve --addr :50069 --config <billing-config>`. Microservice Compose uses image build args `SERVICE_DIR=smart-recruit-billing-service`, `CMD_PATH=./cmd/billing-service`, and `BINARY_NAME=billing-service`.

## Focused verification

```sh
GOWORK=off go test ./servicebinary
# from smart-recruit-platform-go/

GOWORK=off go test ./...
# from smart-recruit-billing-service/ when Billing binaries or contracts change
```

## Verification

Verified against every `smart-recruit-*-service/cmd/*/main.go` entrypoint, Gateway defaults, `start-dev.sh` service ports, and `smart-recruit-deploy/docker-compose.microservices.yml` Billing service definition on 2026-07-23.
