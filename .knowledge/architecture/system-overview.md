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
  - hr-frontend/**
  - user-frontend/**
  - interviewer-frontend/**
  - smart-recruit-gateway/**
  - smart-recruit-*-service/**
  - smart-recruit-commons/**
  - smart-recruit-platform-go/**
  - smart-recruit-proto/**
  - smart-recruit-deploy/**
source_refs:
  - README.md
  - pnpm-workspace.yaml
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-gateway/config/config.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-platform-go/servicebinary/convention.go
  - smart-recruit-commons/internal/platform/events/envelope.go
  - smart-recruit-deploy/docker-compose.microservices.yml
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Smart Recruit System Overview

Smart Recruit has three Vue frontends, a Gin gateway, independent Go services, shared protobuf contracts, shared platform utilities, and shared commons packages. The gateway owns HTTP routing, middleware, auth/RBAC enforcement, quotas, request limits, SSE endpoints, and generated gRPC clients.

Backend responsibilities are split across Identity, Recruitment, Interview, Offer, Notification, AI Agent, Analytics, and Worker services. Shared protocol definitions live in `smart-recruit-proto/`; shared config, Nacos, gRPC, health, trace, metrics, metadata, and service binary conventions live in `smart-recruit-platform-go/`; shared migrations, MQ, OSS, email, authz/JWT helpers, resume parser, event envelope, and AI support live in `smart-recruit-commons/`.

## Verification

Verified against current repository files on 2026-07-14.
