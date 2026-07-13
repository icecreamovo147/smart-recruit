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
  - smart-recruit-worker-service/cmd/worker-service/main.go
  - smart-recruit-deploy/docker/go-service.Dockerfile
  - smart-recruit-deploy/docker-compose.microservices.yml
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Service Binary Convention Runbook

Service unit conventions live in `smart-recruit-platform-go/servicebinary/`. Each independent service has a `cmd/<service>/main.go` entrypoint and service-owned runtime package. Docker build and local microservice composition live under `smart-recruit-deploy/`. Review command name, image name, config prefix, health/readiness, internal gRPC token/TLS, metrics, and route-mode defaults before changing a service binary.

## Verification

Verified against current repository files on 2026-07-14.
