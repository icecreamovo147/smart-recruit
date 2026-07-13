---
schema_version: 1
id: service-boundaries
title: Service boundaries and ownership
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - architecture
  - grpc
  - gateway
  - rbac
applies_to:
  - smart-recruit-gateway/**
  - smart-recruit-*-service/**
  - smart-recruit-proto/**
  - smart-recruit-platform-go/servicebinary/**
  - smart-recruit-commons/**
  - smart-recruit-deploy/**
source_refs:
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-gateway/config/config.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-platform-go/servicebinary/convention.go
  - smart-recruit-deploy/mysql-table-ownership.json
  - smart-recruit-deploy/docker-compose.microservices.yml
  - smart-recruit-identity-service/internal/runtime/runtime.go
  - smart-recruit-recruitment-service/internal/runtime/runtime.go
  - smart-recruit-interview-service/internal/runtime/runtime.go
  - smart-recruit-offer-service/internal/runtime/runtime.go
  - smart-recruit-notification-service/internal/runtime/runtime.go
  - smart-recruit-ai-agent-service/internal/runtime/runtime.go
  - smart-recruit-analytics-service/internal/runtime/runtime.go
  - smart-recruit-worker-service/internal/runtime/runtime.go
  - smart-recruit-commons/internal/platform/events/envelope.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Service Boundaries and Ownership

The gateway is a transport and policy boundary; service packages own business behavior. Each service runtime registers its bounded-context gRPC servers and uses internal domain/application/infrastructure/interfaces layering where extraction is complete. Current `internal/legacydomain/` packages are service-local carried-over implementation debt and should be treated as current source, not as deleted service roots.

`smart-recruit-proto/proto/recruitment.proto` is the canonical wire contract. `smart-recruit-deploy/mysql-table-ownership.json` is the table ownership source for the current single-MySQL deployment.

## Verification

Verified against current repository files on 2026-07-14.
