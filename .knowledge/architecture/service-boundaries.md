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
  - logic-grpc-service/internal/platform/events/envelope.go
  - logic-grpc-service/repository/user_repo.go
last_verified: 2026-07-11
review_after: 2026-10-08
---

# Service Boundaries and Ownership

The HTTP gateway is a transport and policy boundary. It should translate HTTP requests to gRPC calls, enforce request-level middleware, require roles and permissions, apply quotas and limits, and stream responses where needed. It should not own domain state machines or persistence rules.

The logic gRPC service is the business boundary. It owns domain services, repositories, model persistence, AI runtime composition, embedding and memory behavior, outbox and message queue behavior, generated protobuf service implementations, and internal platform contracts shared by backend contexts.

`logic-grpc-service/internal/platform/events/` defines shared domain-event envelope contracts for Outbox, Inbox, asynchronous consumers, and Analytics projections. It is an internal backend contract and does not change HTTP, gRPC, protobuf, or database schemas by itself.

Generated protobuf files are contract artifacts. When proto definitions change, generated code in both Go services must stay aligned with the source `.proto` files.

## Boundary Checklist

- Add or change HTTP endpoints in the gateway only when there is a matching gRPC or handler contract.
- Keep RBAC declarations close to gateway routes and verify equivalent permissions exist in authz packages.
- Keep domain invariants in logic services and repositories.
- Keep request deadlines and body limits in the gateway unless the logic service owns a deeper operation timeout.
- Treat protobuf and database schema changes as public-contract or persistence changes that require explicit scope.

## Common Review Questions

- Does this change alter HTTP behavior, gRPC behavior, or both?
- Are generated protobufs synchronized?
- Are gateway permissions and logic-side authorization assumptions still aligned?
- Did the change modify a transport concern when it should have modified a domain service, or the reverse?

## Verification

The boundary was verified from gateway route registration, protobuf service definitions, representative logic service/repository files, and `logic-grpc-service/internal/platform/events/envelope.go` on 2026-07-11.
