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
  - web-gin-service/**
  - logic-grpc-service/**
  - logic-grpc-service/proto/**
source_refs:
  - web-gin-service/router/router.go
  - logic-grpc-service/proto/recruitment.proto
  - logic-grpc-service/service/auth_service.go
  - logic-grpc-service/service/recruitment_lifecycle_process_manager.go
  - logic-grpc-service/internal/analytics/application/event_ingestor.go
  - logic-grpc-service/internal/analytics/infrastructure/projection_repository.go
  - logic-grpc-service/internal/analytics/runtime/skeleton.go
  - logic-grpc-service/internal/analytics/runtime/runtime.go
  - logic-grpc-service/internal/platform/servicebinary/convention.go
  - logic-grpc-service/internal/platform/workers/runtime/skeleton.go
  - logic-grpc-service/internal/identity/runtime/skeleton.go
  - logic-grpc-service/internal/identity/runtime/runtime.go
  - logic-grpc-service/internal/aiagent/runtime/skeleton.go
  - logic-grpc-service/internal/recruitment/runtime/skeleton.go
  - logic-grpc-service/internal/recruitment/runtime/runtime.go
  - logic-grpc-service/internal/interview/runtime/skeleton.go
  - logic-grpc-service/internal/interview/runtime/runtime.go
  - logic-grpc-service/internal/offer/runtime/skeleton.go
  - logic-grpc-service/internal/offer/runtime/runtime.go
  - logic-grpc-service/service/ai_agent_runtime.go
  - logic-grpc-service/internal/platform/events/envelope.go
  - logic-grpc-service/repository/user_repo.go
last_verified: 2026-07-12
review_after: 2026-10-08
---

# Service Boundaries and Ownership

The HTTP gateway is a transport and policy boundary. It should translate HTTP requests to gRPC calls, enforce request-level middleware, require roles and permissions, apply quotas and limits, and stream responses where needed. It should not own domain state machines or persistence rules.

The logic gRPC service is the business boundary. It owns domain services, repositories, model persistence, AI runtime composition, embedding and memory behavior, outbox and message queue behavior, generated protobuf service implementations, and internal platform contracts shared by backend contexts.

`logic-grpc-service/internal/platform/events/` defines shared domain-event envelope contracts for Outbox, Inbox, asynchronous consumers, and Analytics projections. It is an internal backend contract and does not change HTTP, gRPC, protobuf, or database schemas by itself.

`logic-grpc-service/service/recruitment_lifecycle_process_manager.go` is the transitional process-manager boundary for cross-context Interview and Offer workflows that still need synchronous application lifecycle updates during the modular-monolith phase. Interview and Offer services should not directly write application status or application transition audit rows; they should call this boundary until the workflow can move fully to asynchronous domain events or extracted service APIs.

Analytics event projection ingestion lives under `logic-grpc-service/internal/analytics/`. The application ingestor consumes standard domain-event envelopes and writes Analytics-owned projection events/checkpoints through infrastructure adapters; it must not import source-domain services or introduce transitional service-read dependencies.

`logic-grpc-service/internal/platform/servicebinary/` records the target backend service unit registry and deployment convention for extracted binaries. The registry is compile-checked but is not wired into startup or gateway routing by TASK-BDME-026.

`logic-grpc-service/cmd/worker-services` is a service skeleton for decomposed worker workloads. It names outbox, notification, email, resume parsing, embedding, agent-run, and analytics-projection workloads for future independent scaling, but default execution does not start consumers and the active worker deployment remains `logic-grpc-service --worker-only`.

`logic-grpc-service/cmd/identity-service` is an unrouted service runtime for the Identity boundary. It can explicitly register AuthService plus the Identity-owned AdminService subset for validation, but it must not receive gateway traffic until a scoped cutover TASK records compatibility and rollback evidence.

`logic-grpc-service/cmd/ai-agent-service` is an unrouted service skeleton for the AI Agent boundary. It is compile-safe only and must not receive gateway traffic or start runtime workers until a scoped extraction/cutover TASK records rollback evidence.

`logic-grpc-service/cmd/recruitment-service` is a service skeleton for the Recruitment boundary. `logic-grpc-service/internal/recruitment/runtime` can explicitly register Recruitment-owned JobService, CandidateService, and ApplicationService adapters. Gateway traffic remains on the monolith by default and can route to `RECRUITMENT_GRPC_ADDR` only when `RECRUITMENT_ROUTE_MODE=recruitment` is explicitly configured with rollback evidence.

`logic-grpc-service/cmd/interview-service` is a service skeleton for the Interview boundary. `logic-grpc-service/internal/interview/runtime` can explicitly register the InterviewService adapter for controlled validation. Gateway traffic remains on the monolith by default and can route to `INTERVIEW_GRPC_ADDR` only when `INTERVIEW_ROUTE_MODE=interview` is explicitly configured with rollback evidence.

`logic-grpc-service/cmd/offer-service` is a service skeleton for the Offer boundary. `logic-grpc-service/internal/offer/runtime` can explicitly register the OfferService adapter for controlled validation. Gateway traffic remains on the monolith by default and can route to `OFFER_GRPC_ADDR` only when `OFFER_ROUTE_MODE=offer` is explicitly configured with rollback evidence.

`logic-grpc-service/cmd/analytics-service` is a service skeleton for the Analytics boundary. `logic-grpc-service/internal/analytics/runtime` can explicitly register the Analytics-owned AdminService reporting subset for controlled validation. The descriptor requires domain-event projection/read-model mode and rejects transitional service-read adapters; gateway traffic remains on the monolith until a scoped cutover TASK records compatibility and rollback evidence.

`logic-grpc-service/service/ai_agent_runtime.go` is the transitional AI Agent runtime boundary inside the monolith. It groups AI chat, candidate AI, provider fallback/config surface, embedding runtime, embedding workload execution, and durable agent-run execution without changing public API routing.

Generated protobuf files are contract artifacts. When proto definitions change, generated code in both Go services must stay aligned with the source `.proto` files.

## Boundary Checklist

- Add or change HTTP endpoints in the gateway only when there is a matching gRPC or handler contract.
- Keep RBAC declarations close to gateway routes and verify equivalent permissions exist in authz packages.
- Keep domain invariants in logic services and repositories.
- Keep cross-context application lifecycle writes behind explicit events, adapters, or process-manager boundaries instead of scattering direct repository writes through Interview or Offer services.
- Keep Analytics projections sourced from domain-event envelopes and Analytics-owned read-model repositories; avoid service-read adapters as a transition strategy.
- Keep new service binaries aligned with `docs/backend-ddd-microservices-evolution-service-binary-convention.md`; do not route traffic to them outside scoped cutover TASKs.
- Keep request deadlines and body limits in the gateway unless the logic service owns a deeper operation timeout.
- Treat protobuf and database schema changes as public-contract or persistence changes that require explicit scope.

## Common Review Questions

- Does this change alter HTTP behavior, gRPC behavior, or both?
- Are generated protobufs synchronized?
- Are gateway permissions and logic-side authorization assumptions still aligned?
- Did the change modify a transport concern when it should have modified a domain service, or the reverse?

## Verification

The boundary was verified from gateway route registration and route-mode cutover controls, protobuf service definitions, representative logic service/repository files, `logic-grpc-service/service/recruitment_lifecycle_process_manager.go`, Analytics projection ingestion/runtime files, Worker Services descriptor, Identity, Recruitment, Interview, Offer, and AI Agent skeleton/runtime files, `logic-grpc-service/internal/platform/servicebinary/convention.go`, and `logic-grpc-service/internal/platform/events/envelope.go` on 2026-07-12.
