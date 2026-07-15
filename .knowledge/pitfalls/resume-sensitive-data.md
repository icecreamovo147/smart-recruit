---
schema_version: 1
id: resume-sensitive-data
title: Resume sensitive data pitfall
kind: pitfall
status: active
owners:
  - agent-platform
tags:
  - resume
  - privacy
  - security
applies_to:
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-commons/oss/**
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
source_refs:
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-gateway/middleware/resume_quota.go
  - smart-recruit-commons/oss/storage.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_persistence_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
last_verified: 2026-07-15
review_after: 2026-10-14
---

# Resume Sensitive Data Pitfall

Resume files, parsed text, match evidence, and AI prompts/results can contain personal data. Do not log raw resume text, presigned URLs, credentials, or full prompts/results. Keep ownership checks, file limits, quota middleware, and redaction intact.

Structured recruiting diagnostics must be allow-list based, not produced by logging request structs, arbitrary context values, provider errors, or decoded model objects. Never infer trust from an identifier's alphabet: phones, body fragments, API keys, JWTs, UUIDs, ULIDs, and ordinary-looking IDs can all be externally controlled. The recruiting boundary replaces every non-empty request ID with a process-random-key, domain-separated HMAC token and carries an internal normalization marker so repeated application/native/Zap boundaries are idempotent without trusting a caller-supplied token shape. It maps every resource/operation/stage/category/outcome/agent/fallback value through a fixed whitelist; represents valid Prompt/model identity with bounded non-reversible correlations; preserves only exact internal parser/scorer versions; and bounds non-negative numeric resource/count/duration fields. Evidence sent to the per-requirement evaluator must remain allow-listed, contact-redacted, deterministically bounded, and tied to a persisted source ID; complete legacy scoring text is request-local and must never be logged or copied into reports/evidence.

When a live local provider or service path is unavailable, record an explicit skip reason and use synthetic fake-provider fixtures plus real persistence integration tests. Never copy a real resume, Prompt body, raw model response, database snapshot, credential, or candidate evidence into smoke artifacts to manufacture proof.

## Verification

Verified against current recruiting runtime, evidence builder, privacy observer, and focused regression tests on 2026-07-15.
