---
schema_version: 1
id: resume-intelligence
title: Resume intelligence and candidate matching
kind: domain
status: active
owners:
  - agent-platform
tags:
  - resume
  - matching
  - ai
  - candidate
applies_to:
  - smart-recruit-commons/oss/**
  - smart-recruit-commons/resumeparser/**
  - smart-recruit-recruitment-service/**
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_aggregation.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
source_refs:
  - smart-recruit-commons/oss/storage.go
  - smart-recruit-commons/resumeparser/parser.go
  - smart-recruit-commons/resumeparser/registry.go
  - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
  - smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/capability_context.go
  - smart-recruit-commons/migrations/000071_add_structured_ai_release_trace.sql
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_aggregation.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_resume_profile_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_candidate_match_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_persistence_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-gateway/middleware/resume_quota.go
last_verified: 2026-07-20
review_after: 2026-10-14
---

# Resume Intelligence and Candidate Matching

Resume upload and ownership validation are Recruitment/Gateway concerns, shared OSS and parser support lives in Commons, and AI-driven profile extraction plus matching are exposed through AI Agent native gRPC adapters and capability services. Protect candidate-sensitive data and avoid raw resume text in logs, metrics, reports, or prompts beyond scoped processing.

The native RecruitingIntelligence adapter supports synchronous resume reparse and candidate-match evaluation when its authorized store and runtime dependencies are bound. Resume reparse strictly validates the `resume_profile_extractor` structured result and persists a new version transactionally; eligible failures use the deterministic heuristic only when the existing fallback policy enables it. Candidate matching derives an internal validated job-requirement profile, builds bounded/redacted evidence, evaluates deterministic matches first, uses `candidate_match_evaluator` only for unresolved requirements, and computes the final score and recommendation deterministically.

For paid gateway requests, `resume_profile_extractor`, `job_requirement_extractor`, and `candidate_match_evaluator` Prompts are loaded as true System messages only from the entitlement-fixed capability release, and the effective model is resolved from the same release. Job requirement profiles remain transient. Existing profile/evaluation versions, current/latest demotion, evidence rows, and `agent_run_id` association are preserved; parse runs and match evaluations also persist requested/effective model, fallback reason, release ID, and snapshot hash. Missing dependencies and fallback-disabled primary failures remain explicit non-success and do not create misleading snapshots.

Recruiting observations normalize every field at the application/native boundary and defensively at the log adapter. Every non-empty external request ID becomes an idempotent, process-keyed HMAC correlation token regardless of alphabet. Fixed classifications reject unknown input as `unknown`; valid UTF-8 Prompt/model values within the persisted 256/128-byte contracts retain stable bounded correlations, overlong values use a separate non-reversible correlation domain, invalid/control text fails closed, and exact internal parser/scorer versions remain readable. This preserves correlation without allowing request headers, configured secret-shaped text, candidate data, provider bodies, control characters, or oversized values to become reversible log fields.

Both native write-or-read methods finalize through one deferred terminal state installed before validation. Nil requests, invalid IDs, missing stores/runtime dependencies, application authorization/not-found/errors, capability and structured-path decisions, compatibility reads, generation/fallback, timeouts, persistence failures, and success all produce exactly one terminal event. Generation or aggregation success remains non-terminal until the save succeeds.

## Verification

Verified against the structured runtime/extractor/evaluator/aggregator implementation, capability-release propagation, native orchestration, real persistence and terminal-observation tests, and public gateway/Proto contracts on 2026-07-20.
