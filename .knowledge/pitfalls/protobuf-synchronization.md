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
  - smart-recruit-proto/scripts/**
  - scripts/check-proto-sync.mjs
  - .github/workflows/ci.yml
  - smart-recruit-gateway/rpc/**
  - smart-recruit-*-service/internal/interfaces/grpc/**
source_refs:
  - smart-recruit-proto/proto/recruitment.proto
  - smart-recruit-proto/recruitment/pb/recruitment.pb.go
  - smart-recruit-proto/recruitment/pb/recruitment_grpc.pb.go
  - smart-recruit-proto/scripts/tool-versions.env
  - smart-recruit-proto/scripts/bootstrap-tools.sh
  - smart-recruit-proto/scripts/generate-go.sh
  - scripts/check-proto-sync.mjs
  - .github/workflows/ci.yml
  - smart-recruit-proto/proto_contract_test.go
  - smart-recruit-gateway/rpc/client.go
  - smart-recruit-gateway/router/contract_baseline_test.go
  - scripts/check-agent-skill-v2-cutover.mjs
last_verified: 2026-07-28
review_after: 2026-10-14
---

# Protobuf Synchronization Pitfall

The canonical protobuf source is `smart-recruit-proto/proto/recruitment.proto`. Generated Go contracts under `smart-recruit-proto/recruitment/pb/` are shared by the gateway and services. Wire-shape changes can widen compile scope across all generated clients and test fakes.

Protobuf generation is reproducible only with the repository-pinned toolchain in `smart-recruit-proto/scripts/tool-versions.env`. Run `scripts/bootstrap-tools.sh` before generation; `generate-go.sh` prefers that cache and fails before writing when `protoc`, `protoc-gen-go`, or `protoc-gen-go-grpc` differs from the pins. The sync checker validates canonical file presence and generated tool-version headers, while CI regenerates and requires an empty Git diff. A `proto_sync_result: PASS` therefore confirms the expected files and pinned generator identity; the subsequent empty-diff gate proves source-to-output reproducibility.

For internal owner contracts that are not part of frontend/gateway behavior, prefer a separate internal gRPC service over appending methods to a public-facing service interface. This avoids forcing unrelated `pb.<Service>Client` fakes to implement internal-only methods while keeping protobuf changes additive.

Agent Skill Package v2 intentionally is not backward compatible. Retired Skill ID/boolean confirmation, `skill_capability_keys`, and `skill_md`/`flow_json` package fields keep their Proto tags/names reserved so they cannot be reused accidentally. Active requests use exact `agent_skill_version_ids` plus data/Tool `capability_keys`, and Agent Skill confirmation is separate from opaque MCP confirmation. Pinned generation is necessary but not sufficient: run the gateway removed-route/legacy-JSON tests and `scripts/check-agent-skill-v2-cutover.mjs` to prove no application alias reintroduces the retired contract.

## Verification

Verified against the pinned tool version source, Package v2 active/reserved fields, cross-platform bootstrap, fail-fast generator, generated headers, sync checker, Proto contract tests, gateway removed-route tests, cutover scanner, and Proto Lint workflow on 2026-07-28.
