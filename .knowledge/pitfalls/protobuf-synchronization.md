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
  - generated-code
applies_to:
  - logic-grpc-service/proto/**
  - web-gin-service/proto/**
  - logic-grpc-service/recruitment/pb/**
  - web-gin-service/recruitment/pb/**
source_refs:
  - logic-grpc-service/proto/recruitment.proto
  - logic-grpc-service/recruitment/pb/recruitment.pb.go
  - web-gin-service/recruitment/pb/recruitment.pb.go
  - .spec/agent-skill-selection-confirmation/reports/TASK-ASC-002-report.md
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Protobuf Synchronization Pitfall

Proto changes are easy to under-scope because the source `.proto` file and generated Go code exist in more than one service tree. A change that updates only one side can compile in one package while leaving gateway and logic contracts inconsistent.

## Trigger Conditions

- Adding, renaming, or deleting fields in `recruitment.proto`.
- Adding service methods, request messages, response messages, enum values, or stream event fields.
- Changing generated Go files without the matching source proto.
- Updating frontend payloads for a protobuf-backed API without checking generated gateway and logic structs.

## Risk

The HTTP gateway may marshal, forward, or stream a different contract than the logic service expects. Missing generated updates can also hide until a package-specific build or test runs.

## Prevention

- Treat proto source and generated files as one public-contract change.
- Check both `logic-grpc-service/recruitment/pb/` and `web-gin-service/recruitment/pb/`.
- Run targeted Go tests in both services after protobuf-related changes.
- Update frontend API/types only after the server contract is known.

## Verification

This pitfall was verified from current proto locations and historical Agent Skill selection contract work on 2026-07-10.
