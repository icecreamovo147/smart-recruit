---
schema_version: 1
id: system-overview
title: Smart Recruit system overview
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - architecture
  - services
  - frontend
  - backend
applies_to:
  - README.md
  - hr-frontend/**
  - user-frontend/**
  - interviewer-frontend/**
  - web-gin-service/**
  - logic-grpc-service/**
source_refs:
  - README.md
  - web-gin-service/router/router.go
  - logic-grpc-service/main.go
  - logic-grpc-service/internal/platform/events/envelope.go
  - pnpm-workspace.yaml
last_verified: 2026-07-11
review_after: 2026-10-08
---

# Smart Recruit System Overview

Smart Recruit is split into three Vue frontends, a Gin HTTP gateway, and a Go gRPC logic service. The frontends are workspace packages under `hr-frontend/`, `user-frontend/`, and `interviewer-frontend/`. The gateway owns HTTP routing, request middleware, RBAC checks, limits, body-size controls, SSE endpoints, and calls into the logic service through generated gRPC clients.

The logic service owns core recruitment behavior, persistence orchestration, AI agent runtime, embedding services, message publishing, object storage integration, and protobuf service implementations. Persistent data flows through `repository/` and `model/`, while business workflows live mostly under `logic-grpc-service/service/`. Shared internal platform contracts, such as the domain-event envelope in `logic-grpc-service/internal/platform/events/`, sit under the logic service and are intended for Outbox, Inbox, consumer, and projection code.

Use this document for orientation only. For concrete behavior, prefer the active `.spec` contract, source code, generated protobufs, migrations, and tests.

## Main Boundaries

- `hr-frontend/`: HR and admin workflows, including AI chat, model configuration, Agent Skill management, embedding configuration, recruiting intelligence, security audit, and operational pages.
- `user-frontend/`: candidate job browsing, application tracking, resume, interview, offer, notification, and candidate AI flows.
- `interviewer-frontend/`: interviewer-facing task and feedback flows.
- `web-gin-service/`: HTTP API surface, auth middleware, staff/candidate route groups, rate limits, quotas, security headers, request body limits, Swagger, and gateway-to-gRPC calls.
- `logic-grpc-service/`: domain services, AI orchestration, repositories, models, embedding and memory behavior, domain-event contracts, message queue integration, storage integration, and protobuf service servers.

## Change Impact Hints

- HTTP route, middleware, permission, or request limit changes usually affect `web-gin-service/router/router.go`, handlers, frontend API clients, and RBAC tests.
- Business workflow changes usually affect a service, repository, model, protobuf contract, and at least one frontend role.
- Agent runtime, memory, embedding, or Skill changes usually affect logic service code first; HR admin debug pages are common verification surfaces.
- Frontend workspace changes should be checked per package because there is no root `package.json`.

## Verification

The structure was verified from `README.md`, `web-gin-service/router/router.go`, `logic-grpc-service/main.go`, `logic-grpc-service/internal/platform/events/envelope.go`, and current repository paths on 2026-07-11.
