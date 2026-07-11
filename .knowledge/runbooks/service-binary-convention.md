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
  - docs/backend-ddd-microservices-evolution-service-binary-convention.md
  - deploy/k8s/README-service-binaries.md
  - logic-grpc-service/internal/platform/servicebinary/**
  - logic-grpc-service/cmd/**
source_refs:
  - docs/backend-ddd-microservices-evolution-service-binary-convention.md
  - docs/backend-ddd-microservices-evolution-notification-service-skeleton.md
  - deploy/k8s/README-service-binaries.md
  - logic-grpc-service/internal/platform/servicebinary/convention.go
  - logic-grpc-service/internal/platform/servicebinary/convention_test.go
  - logic-grpc-service/cmd/notification-service/main.go
  - logic-grpc-service/internal/notification/runtime/skeleton.go
  - deploy/k8s/logic-deployment.yaml
  - deploy/k8s/worker-deployment.yaml
  - docker/logic-grpc-service.Dockerfile
last_verified: 2026-07-11
review_after: 2026-10-09
---

# Service Binary Convention Runbook

Use this runbook when adding or reviewing backend service binaries, worker binaries, or deployment skeletons during the backend DDD microservices evolution.

## Current Contract

- `logic-grpc-service/internal/platform/servicebinary/` contains the compile-checked service unit registry.
- `docs/backend-ddd-microservices-evolution-service-binary-convention.md` is the operator and implementation convention.
- `deploy/k8s/README-service-binaries.md` records deployment label conventions without adding routed manifests.
- Current active deployments remain `logic-grpc-service` and `logic-worker`; extracted service entries are future command/image conventions until scoped TASKs create and cut them over.
- `cmd/notification-service` is the first compile-safe extracted service skeleton. It supports `--describe` and `--check`, exits non-zero without flags, and keeps `TrafficEnabled=false` until a scoped cutover TASK changes that behavior.

## Review Checklist

1. Confirm the unit name exists in the registry or is added with command, image, config prefix, health, and cutover fields.
2. Confirm the binary is not wired to production traffic unless the current TASK is a cutover TASK.
3. Confirm the service does not import another bounded context's repository package without a documented exception.
4. Confirm docs, deployment labels, and knowledge routes remain aligned.
5. For service skeletons, confirm default execution does not bind a listener or start consumers unless explicitly scoped.
6. Run `cd logic-grpc-service && go test ./internal/platform/servicebinary ./...`.

## Staleness Signals

Mark this document stale if the monorepo service root changes, Docker image naming changes, active deployment manifests change service roles, or extracted services receive traffic through a new routing mechanism.
