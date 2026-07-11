---
schema_version: 1
id: api-contracts-and-gateway
title: API contracts and gateway architecture
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - gateway
  - api
  - grpc
  - contract
applies_to:
  - web-gin-service/config/**
  - web-gin-service/router/**
  - web-gin-service/handler/**
  - web-gin-service/middleware/**
  - web-gin-service/rpc/**
  - logic-grpc-service/proto/**
source_refs:
  - web-gin-service/config/config.go
  - web-gin-service/router/router.go
  - web-gin-service/rpc/client.go
  - web-gin-service/middleware/body_limit.go
  - web-gin-service/middleware/ratelimit.go
  - web-gin-service/middleware/observability.go
  - logic-grpc-service/proto/recruitment.proto
  - logic-grpc-service/main.go
last_verified: 2026-07-12
review_after: 2026-10-08
---

# API Contracts and Gateway Architecture

The Gin gateway is the HTTP policy and transport boundary. It exposes `/api/v1` routes, applies timeout/body/rate/quota/risk middleware, performs auth and RBAC checks, and calls generated gRPC clients. It should not own core recruitment state transitions, persistence rules, or AI runtime selection.

## Contract Layers

- `web-gin-service/router/router.go` wires public, candidate, staff, admin, AI, notification, analytics, collaboration, and configuration routes.
- `web-gin-service/handler/**` translates HTTP payloads, path/query parameters, and streaming responses to protobuf-backed gRPC calls.
- `web-gin-service/middleware/**` owns HTTP concerns such as JWT, role/permission checks, body limits, rate limits, quotas, risk blocking, CSP, request IDs, and timeouts.
- `web-gin-service/config/config.go` owns gateway runtime configuration, including Notification, AI Agent, and Identity route mode rollback switches.
- `web-gin-service/rpc/client.go` creates generated gRPC clients, forwards internal auth/request metadata, restricts retry policy to known read-only methods, and can route only the Notification generated client to `NOTIFICATION_GRPC_ADDR`, AI Agent-owned generated clients to `AI_AGENT_GRPC_ADDR`, or Identity AuthService plus Identity-owned AdminService methods to `IDENTITY_GRPC_ADDR`.
- `logic-grpc-service/proto/recruitment.proto` is the source contract for gateway and logic generated clients/servers.
- `logic-grpc-service/main.go` registers gRPC service implementations and health checks.

## Impact Guidance

- Adding an HTTP endpoint usually requires handler, route, permission, frontend API/types, and a matching gRPC or existing handler contract.
- Changing a protobuf message or service method is a public-contract change and must update generated Go code in both service trees.
- Adding request bodies should check `MaxBodyBytes` limits and timeout category.
- Adding streaming endpoints should check gateway response flushing and frontend event parsing.
- Retrying write RPCs is unsafe unless idempotency is explicitly designed.
- Notification gateway cutover must keep `NOTIFICATION_ROUTE_MODE=logic` as the default rollback path and require `NOTIFICATION_GRPC_ADDR` before routing the generated Notification client to an extracted service.
- AI Agent gateway cutover must keep `AI_AGENT_ROUTE_MODE=logic` as the default rollback path and require `AI_AGENT_GRPC_ADDR` before routing AI Agent-owned generated clients to an extracted service.
- Identity gateway cutover must keep `IDENTITY_ROUTE_MODE=logic` as the default rollback path, require `IDENTITY_GRPC_ADDR` before routing to an extracted service, and preserve non-Identity AdminService methods on the logic target until separately scoped.

## Verification

Verified against gateway configuration loading, route setup, Notification, AI Agent, and Identity gRPC cutover controls, middleware categories, protobuf definitions, and logic service registration on 2026-07-12.
