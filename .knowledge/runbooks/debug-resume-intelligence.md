---
schema_version: 1
id: debug-resume-intelligence
title: Debug resume intelligence
kind: runbook
status: active
owners:
  - agent-platform
tags:
  - resume
  - matching
  - debug
applies_to:
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-recruitment-service/**
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
  - smart-recruit-commons/resumeparser/**
source_refs:
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-gateway/middleware/resume_quota.go
  - smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service.go
  - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
  - smart-recruit-commons/resumeparser/parser.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/llm_runtime.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_store.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/resume_profile.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/job_requirement.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/candidate_match.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/recruiting_observability_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_resume_profile_test.go
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/native_candidate_match_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/native_persistence_test.go
  - smart-recruit-ai-agent-service/internal/infrastructure/provider/doc.go
  - smart-recruit-ai-agent-service/internal/application/service/capability_service.go
last_verified: 2026-07-15
review_after: 2026-10-14
---

# Debug Resume Intelligence

Verify gateway quota/body limits, Recruitment resume ownership and upload-session validation, Commons OSS/parser behavior, AI Agent profile extraction and candidate matching, and privacy-safe logs/reports.

## Structured Runtime Checks

1. Confirm the AI Agent, Identity, Recruitment, and Gateway dependencies required by the chosen path are healthy. Do not point a local smoke command at production.
2. Query only Prompt metadata and confirm exactly one selected active row for each of `resume_profile_extractor`, `job_requirement_extractor`, and `candidate_match_evaluator`, with `prompt_role=system`. Do not select `content` or credentials.
3. Reparse a permitted local test resume. Confirm a new profile version/current snapshot only after success. If the provider is unavailable, record a provider-dependent skip and run the fake-provider native orchestration plus real persistence tests; do not label the live path PASS.
4. Evaluate a permitted local test application. Confirm job requirements are internal, deterministic evidence matches avoid evaluator calls, unresolved requirements use the active evaluator Prompt, and the saved total/recommendation come from deterministic aggregation.
5. For fallback-disabled invalid Prompt/provider/JSON/schema output, confirm the RPC is non-success and no profile/evaluation version is added. For fallback-enabled execution, confirm provenance identifies the fallback rather than primary LLM success.

Use this metadata-only MySQL query to reproduce the runtime's active/latest selection. It deliberately excludes `content`, provider configuration, credentials, and candidate data:

```sql
WITH ranked AS (
  SELECT id, name, agent_type, prompt_role, version, is_active, updated_at,
         ROW_NUMBER() OVER (
           PARTITION BY agent_type, prompt_role
           ORDER BY updated_at DESC, id DESC
         ) AS selection_rank
  FROM prompt_templates
  WHERE is_active = 1
    AND prompt_role = 'system'
    AND agent_type IN (
      'resume_profile_extractor',
      'job_requirement_extractor',
      'candidate_match_evaluator'
    )
)
SELECT id, name, agent_type, prompt_role, version, is_active, updated_at
FROM ranked
WHERE selection_rank = 1
ORDER BY agent_type;
```

Expected: exactly three rows, one per listed `agent_type`, all active and `prompt_role=system`. A missing row is a Prompt/configuration failure; do not inspect body content to compensate.

## Privacy-Safe Observability

Recruiting logs may contain only a process-keyed HMAC request correlation, non-negative internal numeric resource ID, fixed resource/operation/stage/category/outcome/agent/fallback classifications, bounded Prompt/model correlations, exact trusted internal parser/scorer versions, terminal marker, bounded counts, and bounded duration in addition to the standard fixed log envelope. Unknown classifications must appear only as fixed `unknown`; invalid UTF-8/control identity text must be empty, never truncated or copied. Safe-alphabet phones, body text, API keys, JWTs, UUIDs, ULIDs, and normal request IDs must never appear raw. Every method invocation has exactly one `terminal=true` event, including validation/dependency/authorization/disabled/compatibility early returns; generation or aggregation success remains `terminal=false` until persistence succeeds.

## Focused Local Validation

From the repository root, use these exact commands:

```bash
(cd smart-recruit-ai-agent-service && GOWORK=off go test ./internal/application/recruiting_intelligence -run 'TestRuntimeLoadsEveryRecruitingSystemPromptOnEveryRequest|TestResumeFallbackObservationContainsClassificationNotProviderBody|TestObservationBoundary|TestObservationRequestID|TestObservationIdentity' -count=10)
(cd smart-recruit-ai-agent-service && GOWORK=off go test ./internal/interfaces/grpc -run 'TestRecruitingRuntimeObserver|TestRecruitingNativeObservationRejectsPropagatedExternalRequestID|TestRecruitingNativeOperationsEmitExactlyOneTerminalOutcome|TestRecruitingNativeMethodBoundaryFinalizesEveryEarlyReturn|TestParseResumeProfileSaveFailureReturnsNonSuccessWithoutCommittedDraft|TestEvaluateCandidateMatchDeadlineCoversSourceReadAndParentCancellationPreventsSave' -count=10)
(cd smart-recruit-ai-agent-service && GOWORK=off go test ./internal/infrastructure/persistence -run 'TestNativeStoreSaveRecruitingResumeProfileDraft|TestNativeStoreSaveRecruitingCandidateMatchDraft' -count=10)
(cd smart-recruit-ai-agent-service && GOWORK=off go test ./...)
(cd smart-recruit-commons && GOWORK=off go test ./ai -run TestGenerateStructured -count=10)
```

Expected: every command exits `0`; the HMAC tests prove same-process stability, distinct-input separation, repeated-normalization idempotence, schema-length Unicode identity handling, and absence of raw secret-shaped values. The terminal tests cover all early returns plus primary, fallback, schema/provider/timeout/source/persistence outcomes without double or missing terminals; persistence failure never becomes overall success.

Before any live smoke, determine service availability without reading configuration secrets:

```bash
for port in 50061 50062 50066 8080; do lsof -nP -iTCP:${port} -sTCP:LISTEN; done
```

Identity `50061`, Recruitment `50062`, AI Agent `50066`, and Gateway `8080` must all have listeners. If any listener is absent, record a service-dependent `SKIP`. If services listen but the runtime returns the fixed “no enabled llm model/provider” classification, record a provider-dependent `SKIP`; do not print environment variables, API keys, endpoints, DSNs, or decrypted provider rows.

Run a live API smoke only with a pre-authorized, fully synthetic local staff account, resume, application, and job fixture created through normal local APIs. The synthetic resume must contain invented names/contact domains and no copied production text. Record only fixture numeric IDs, response codes, created version numbers, terminal categories, and bounded counts. Reparse that synthetic resume, evaluate its synthetic application, verify one new current/latest snapshot, then delete the fixture through its normal cleanup path. If no such authorized fixture exists, do not call the live API: record the explicit skip and rely on the fake-provider orchestration plus SQLite persistence commands above.

## Verification

Verified against current gateway/Recruitment boundaries, AI Agent runtime/persistence, and local fake-provider/privacy test procedures on 2026-07-15.
