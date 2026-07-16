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
  - smart-recruit-recruitment-service/cmd/recruitment-service/main.go
  - smart-recruit-recruitment-service/internal/infrastructure/persistence/native_adapters.go
  - smart-recruit-interview-service/internal/runtime/runtime.go
  - smart-recruit-offer-service/internal/runtime/runtime.go
  - smart-recruit-notification-service/internal/runtime/runtime.go
  - smart-recruit-ai-agent-service/internal/runtime/runtime.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-analytics-service/internal/runtime/runtime.go
  - smart-recruit-worker-service/internal/runtime/runtime.go
  - smart-recruit-interview-service/internal/infrastructure/client/application_adapter.go
  - smart-recruit-interview-service/internal/infrastructure/client/authorizer.go
  - smart-recruit-interview-service/internal/infrastructure/persistence/interview_repository.go
  - smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher.go
  - smart-recruit-offer-service/internal/infrastructure/client/application_adapter.go
  - smart-recruit-offer-service/internal/infrastructure/client/authorizer.go
  - smart-recruit-offer-service/internal/infrastructure/persistence/offer_repository.go
  - smart-recruit-offer-service/internal/infrastructure/mq/outbox_publisher.go
  - smart-recruit-commons/internal/platform/events/envelope.go
last_verified: 2026-07-15
review_after: 2026-10-14
---

# Service Boundaries and Ownership

The gateway is a transport and policy boundary; service packages own business behavior. Each service runtime registers its bounded-context gRPC servers and uses internal domain/application/infrastructure/interfaces layering where extraction is complete. 

Cross-context business reads/writes should use owner contracts instead of copied repositories. Identity exposes `AuthService.GetPrincipal` and internal `AuthService.AuthorizeInternal` for principal, permission, data-scope, and audit checks. Recruitment exposes internal `ApplicationOwnerService` for application snapshot and lifecycle transitions while preserving the existing public-facing `ApplicationService` shape.

Recruitment has retired its service-local `internal/legacydomain/` copy. Its active runtime constructs the gRPC surface through explicit `runtime.Deps` backed by focused local persistence adapters in `internal/infrastructure/persistence/native_adapters.go` for job, taxonomy, admin/invite, usage, candidate/resume, application lifecycle, application-owner contract, and collaboration responsibilities.

AI Agent has retired and deleted its service-local `internal/legacydomain/` copy; `cmd/ai-agent-service` wires `internal/interfaces/grpc/native_servers.go` with `internal/infrastructure/persistence/native_store.go` for chat, sessions, tool traces, and durable agent runs. Interview and Offer services have retired their service-local `internal/legacydomain/` copies. Their active runtimes now use local persistence and outbox adapters under `internal/infrastructure/**`, while application snapshots, lifecycle transitions, and authorization are reached through explicit Recruitment and Identity owner adapters. Interviewer assignment checks are Interview-owned local reads over `interview_schedules`. Backend boundary and MySQL table-ownership checks now fail if any targeted service reintroduces `internal/legacydomain`.

Recruitment continues to own applications, resumes, candidate profiles, and resume-profile source rows. AI Agent reads those authorized sources for scoped recruiting intelligence and owns structured generation, deterministic matching/aggregation, `candidate_match_evaluations`, `candidate_match_evidence`, Prompt/model runtime access, and durable Agent-run association. This flow does not introduce a cross-service write, new table, or public contract.

For HR-wide application/candidate reads, AI Agent composes the existing HR job inventory and per-job application RPCs rather than reading Recruitment tables or adding a new public aggregate RPC. The composition is bounded to 100 jobs, ten pages per job, four concurrent fetches, and 5,000 returned rows; partial failures are explicit and all-job failure remains non-success.

`smart-recruit-proto/proto/recruitment.proto` is the canonical wire contract. `smart-recruit-deploy/mysql-table-ownership.json` is the table ownership source for the current single-MySQL deployment.

## Verification

Verified against current service adapters, bounded HR aggregation, persistence mappings, table ownership, and generated contracts on 2026-07-16.
