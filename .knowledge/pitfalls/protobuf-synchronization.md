---
schema_version: 1
id: protobuf-synchronization
title: Protobuf synchronization pitfall
kind: pitfall
status: active
owners:
  - engineering-platform
tags:
  - protobuf
  - grpc
  - contract
applies_to:
  - smart-recruit-proto/proto/**
  - smart-recruit-proto/recruitment/pb/**
  - smart-recruit-gateway/rpc/**
  - smart-recruit-*-service/internal/interfaces/grpc/**
source_refs:
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-proto/recruitment/pb/recruitment_grpc.pb.go
  - smart-recruit-proto/scripts/generate-go.sh
  - smart-recruit-proto/proto_contract_test.go
  - smart-recruit-gateway/rpc/client.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Protobuf Synchronization Pitfall

The canonical protobuf source is `smart-recruit-proto/proto/recruitment.proto`. Generated Go contracts under `smart-recruit-proto/recruitment/pb/` are shared by the gateway and services. Wire-shape changes can widen compile scope across all generated clients and test fakes.

For internal owner contracts that are not part of frontend/gateway behavior, prefer a separate internal gRPC service over appending methods to a public-facing service interface. This avoids forcing unrelated `pb.<Service>Client` fakes to implement internal-only methods while keeping protobuf changes additive.

## Verification

Verified against current repository files on 2026-07-14.
