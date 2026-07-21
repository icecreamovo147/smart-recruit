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
  - dev-log-viewer/**
  - docker/**
  - deploy/**
  - smart-recruit-gateway/**
  - smart-recruit-*-service/**
  - smart-recruit-commons/**
  - smart-recruit-platform-go/**
  - smart-recruit-deploy/**
  - hr-frontend/**
  - user-frontend/**
  - interviewer-frontend/**
source_refs:
  - README.md
  - start-dev.sh
  - stop-dev.sh
  - smart-recruit-commons/cmd/migrate/main.go
  - smart-recruit-commons/cmd/time-preflight/main.go
  - smart-recruit-platform-go/businessclock/clock.go
  - smart-recruit-platform-go/mysqltime/mysql.go
  - smart-recruit-commons/migration/runner.go
  - dev-log-viewer/README.md
  - dev-log-viewer/package.json
  - dev-log-viewer/scripts/build-production.sh
  - dev-log-viewer/scripts/smoke-test.sh
  - dev-log-viewer/cmd/dev-log-viewer/main.go
  - docker/docker-compose.yml
  - docker/.env.example
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-internal-service-security.md
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-observability-baseline.md
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-deployment-readiness-baseline.md
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-load-test-harness.md
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-final-readiness-review.md
  - .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-final-readiness-audit.json
  - scripts/backend-load-test.mjs
  - scripts/backend-final-readiness-audit.mjs
last_verified: 2026-07-21
review_after: 2026-10-08
---

# Local Development Runbook

Use this runbook to orient local startup and validation. Always prefer checked-in scripts and the current `README.md` over copied shell fragments in older notes.

## Prerequisites

- Go, Node.js, pnpm, Docker, and Docker Compose compatible with the versions documented in `README.md`.
- Local service configuration copied from example files and filled with placeholders or local-only credentials.
- Local Docker Compose requires `GRPC_INTERNAL_TOKEN`; internal gRPC TLS remains `GRPC_INTERNAL_TLS=optional` unless local certificates are mounted.
- Gateway metrics are available at `http://localhost:<HTTP_PORT>/metrics`; service metrics use each service's configured observability address.
- Worker health is owned by `smart-recruit-worker-service/`; readiness fails when required MySQL, configured Redis, or RabbitMQ dependencies are unavailable.
- Backend load-test dry-run evidence is generated with `node scripts/backend-load-test.mjs --dry-run --output .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json`; live 200 QPS/50 QPS/AI concurrency runs require a running isolated stack and authenticated test fixtures.
- Final readiness audit evidence is generated with `node scripts/backend-final-readiness-audit.mjs --feature-dir .spec/backend-ddd-microservices-evolution --allow-current-task TASK-BDME-052 --output .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-final-readiness-audit.json` during the closing TASK.
- MySQL, Redis, and RabbitMQ available through Docker Compose or an equivalent local stack.
- Set `TZ=Asia/Shanghai`. MySQL DSNs must use `parseTime=true`, `loc=Asia%2FShanghai`, and `time_zone=%27%2B08%3A00%27`; service startup validates the resulting `+08:00` session rather than trusting the host timezone.

## Standard Flow

1. Start infrastructure from `docker/` or use `./start-dev.sh` when the script matches the task.
2. When any backend target is selected, `start-dev.sh` builds `smart-recruit-commons/cmd/migrate`, verifies MySQL/Redis/RabbitMQ availability, and applies `smart-recruit-commons/migrations/` before starting services. Migration failure stops startup.
3. Start the independent backend services before `smart-recruit-gateway` because the gateway depends on generated gRPC clients. The full `./start-dev.sh` target performs this ordering automatically.
4. Start only the frontend package needed for the task:
   - HR app: `pnpm --filter hr-frontend dev`
   - candidate app: `pnpm --filter user-frontend dev`
   - interviewer app: `pnpm --filter interviewer-frontend dev`
5. Start the local log viewer only when explicitly needed with `./start-dev.sh logs` or `./start-dev.sh log-viewer`; it uses `127.0.0.1:8090`, `.dev/pids/dev-log-viewer.pid`, and `.dev/logs/dev-log-viewer.log`.
6. Run targeted checks for touched services or apps before broad checks.

## Validation Commands

- Service tests: run `go test ./...` from the touched `smart-recruit-*-service/` module.
- Service binary convention tests: run `go test ./servicebinary` from `smart-recruit-platform-go/`.
- Gateway tests: run `go test ./...` from `smart-recruit-gateway/`.
- Migration runner checks: run `go test ./migration` from `smart-recruit-commons/`; MySQL consistency checks additionally require the repository's configured MySQL test environment.
- UTC+8 data gate: before migration in a maintenance window, run `go run ./cmd/time-preflight --dsn "$MYSQL_DSN"` from `smart-recruit-commons/`; any ambiguous or invalid row returns nonzero and must be resolved before applying migration 000082.
- Static timezone gate: run `node scripts/check-timezone-contract.mjs` from the repository root.
- Observability smoke check: after starting the gateway, request `/metrics` and verify Prometheus text output contains `smart_recruit_http_requests_total`.
- Worker health smoke check: start `smart-recruit-worker-service` with its worker health address configured, then request the configured `/readyz` endpoint.
- Load-test harness dry run: `node scripts/backend-load-test.mjs --dry-run --output .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json`.
- Final readiness audit: `node scripts/backend-final-readiness-audit.mjs --feature-dir .spec/backend-ddd-microservices-evolution --allow-current-task TASK-BDME-052 --output .spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-final-readiness-audit.json`.
- HR frontend typecheck: `pnpm --filter hr-frontend typecheck`.
- Candidate frontend typecheck: `pnpm --filter user-frontend typecheck`.
- Interviewer frontend typecheck: `pnpm --filter interviewer-frontend typecheck`.
- Dev log viewer frontend build: `pnpm --filter dev-log-viewer build`.
- Dev log viewer production binary build: `dev-log-viewer/scripts/build-production.sh`.
- Dev log viewer loopback smoke test: `dev-log-viewer/scripts/smoke-test.sh`.
- Dev log viewer Go checks: run `go test ./...` and `go vet ./...` from `dev-log-viewer/`.

## Cross-Platform Notes

- Use repository-relative paths in docs and scripts.
- Do not store local credentials, device paths, or generated catalogs in Git.
- Do not copy production gRPC TLS private keys into local examples. Keep local cert paths empty unless testing TLS explicitly.
- On Windows, prefer WSL2 for shell-heavy workflows; core knowledge tooling remains Node and Git based.

## When This Runbook Is Stale

Mark this document stale if startup scripts, frontend package commands, Docker service names, service binary conventions, internal gRPC security defaults, metrics or worker health endpoint conventions, or required service order changes.

## Verification

Verified against `start-dev.sh`, the Commons migration command and runner, current workspace commands, Docker Compose assets, and dev-log-viewer scripts on 2026-07-19.
