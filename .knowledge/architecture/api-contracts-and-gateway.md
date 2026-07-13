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
  - smart-recruit-gateway/config/**
  - smart-recruit-gateway/router/**
  - smart-recruit-gateway/handler/**
  - smart-recruit-gateway/middleware/**
  - smart-recruit-gateway/rpc/**
  - smart-recruit-proto/**
source_refs:
  - smart-recruit-gateway/router/router.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-gateway/config/config.go
  - smart-recruit-gateway/middleware/body_limit.go
  - smart-recruit-gateway/middleware/ratelimit.go
  - smart-recruit-gateway/middleware/observability.go
  - smart-recruit-gateway/cmd/gateway/main.go
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# API Contracts and Gateway Architecture

The gateway exposes `/api/v1`, applies timeout/body/rate/quota/risk/auth middleware, and calls generated gRPC clients. `smart-recruit-gateway/rpc/client.go` defaults route modes to independent service targets and forwards internal auth, request, and trace metadata.

Changing HTTP routes normally requires route registration, handler mapping, permission metadata, frontend API/types, and a matching protobuf or service contract. Changing protobuf wire shape is a public-contract change rooted in `smart-recruit-proto/proto/recruitment.proto`.

## Verification

Verified against current repository files on 2026-07-14.
