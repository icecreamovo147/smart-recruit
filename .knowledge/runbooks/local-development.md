---
schema_version: 1
id: local-development
title: Local development runbook
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - development
  - docker
  - go
  - frontend
applies_to:
  - README.md
  - start-dev.sh
  - stop-dev.sh
  - docker/**
  - logic-grpc-service/**
  - web-gin-service/**
  - hr-frontend/**
  - user-frontend/**
  - interviewer-frontend/**
source_refs:
  - README.md
  - start-dev.sh
  - stop-dev.sh
  - docker/docker-compose.yml
  - docker/.env.example
  - docs/backend-ddd-microservices-evolution-internal-service-security.md
  - docs/backend-ddd-microservices-evolution-observability-baseline.md
  - docs/backend-ddd-microservices-evolution-deployment-readiness-baseline.md
  - docs/backend-ddd-microservices-evolution-load-test-harness.md
  - docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json
  - scripts/backend-load-test.mjs
last_verified: 2026-07-12
review_after: 2026-10-08
---

# Local Development Runbook

Use this runbook to orient local startup and validation. Always prefer checked-in scripts and the current `README.md` over copied shell fragments in older notes.

## Prerequisites

- Go, Node.js, pnpm, Docker, and Docker Compose compatible with the versions documented in `README.md`.
- Local service configuration copied from example files and filled with placeholders or local-only credentials.
- Local Docker Compose requires `GRPC_INTERNAL_TOKEN`; internal gRPC TLS remains `GRPC_INTERNAL_TLS=optional` unless local certificates are mounted.
- Gateway metrics are available at `http://localhost:<HTTP_PORT>/metrics`; logic gRPC metrics require setting `METRICS_ADDR` before starting `logic-grpc-service`.
- Worker-only health requires setting `WORKER_HEALTH_ADDR` before starting `logic-grpc-service --worker-only`; `/readyz` fails when MySQL, configured Redis, or RabbitMQ is unavailable.
- Backend load-test dry-run evidence is generated with `node scripts/backend-load-test.mjs --dry-run --output docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json`; live 200 QPS/50 QPS/AI concurrency runs require a running isolated stack and authenticated test fixtures.
- MySQL, Redis, and RabbitMQ available through Docker Compose or an equivalent local stack.

## Standard Flow

1. Start infrastructure from `docker/` or use `./start-dev.sh` when the script matches the task.
2. Start `logic-grpc-service` before `web-gin-service` because the gateway depends on gRPC clients.
3. Start only the frontend package needed for the task:
   - HR app: `pnpm --filter hr-frontend dev`
   - candidate app: `pnpm --filter user-frontend dev`
   - interviewer app: `pnpm --filter interviewer-frontend dev`
4. Run targeted checks for touched services or apps before broad checks.

## Validation Commands

- Logic service tests: run `go test ./...` from `logic-grpc-service/`.
- Service binary convention tests: run `go test ./internal/platform/servicebinary` from `logic-grpc-service/`.
- Gateway tests: run `go test ./...` from `web-gin-service/`.
- Observability smoke check: after starting the gateway, request `/metrics` and verify Prometheus text output contains `smart_recruit_http_requests_total`.
- Worker health smoke check: start `logic-grpc-service --worker-only` with `WORKER_HEALTH_ADDR=:9092`, then request `http://localhost:9092/readyz`.
- Load-test harness dry run: `node scripts/backend-load-test.mjs --dry-run --output docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json`.
- HR frontend typecheck: `pnpm --filter hr-frontend typecheck`.
- Candidate frontend typecheck: `pnpm --filter user-frontend typecheck`.
- Interviewer frontend typecheck: `pnpm --filter interviewer-frontend typecheck`.

## Cross-Platform Notes

- Use repository-relative paths in docs and scripts.
- Do not store local credentials, device paths, or generated catalogs in Git.
- Do not copy production gRPC TLS private keys into local examples. Keep local cert paths empty unless testing TLS explicitly.
- On Windows, prefer WSL2 for shell-heavy workflows; core knowledge tooling remains Node and Git based.

## When This Runbook Is Stale

Mark this document stale if startup scripts, frontend package commands, Docker service names, service binary conventions, internal gRPC security defaults, metrics or worker health endpoint conventions, or required service order changes.
