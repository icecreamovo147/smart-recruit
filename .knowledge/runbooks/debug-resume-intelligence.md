---
schema_version: 1
id: debug-resume-intelligence
title: Debug resume intelligence
kind: runbook
status: active
owners:
  - recruitment-domain
  - agent-platform
tags:
  - resume
  - matching
  - ai
  - debug
applies_to:
  - logic-grpc-service/service/resume_parse_consumer.go
  - logic-grpc-service/service/resume_profile_service.go
  - logic-grpc-service/service/recruiting_intelligence_service.go
  - logic-grpc-service/service/candidate_match_service.go
  - web-gin-service/handler/candidate/resume.go
  - web-gin-service/handler/hr/recruiting_intelligence.go
  - user-frontend/src/views/candidate/ResumeUploadView.vue
  - hr-frontend/src/views/hr/ApplicationIntelligenceView.vue
source_refs:
  - logic-grpc-service/service/resume_parse_consumer.go
  - logic-grpc-service/service/resume_profile_service.go
  - logic-grpc-service/service/resume_profile_extractor_llm.go
  - logic-grpc-service/service/recruiting_intelligence_service.go
  - logic-grpc-service/service/candidate_match_service.go
  - logic-grpc-service/service/candidate_match_llm_matcher.go
  - web-gin-service/handler/candidate/resume.go
  - web-gin-service/handler/hr/recruiting_intelligence.go
  - web-gin-service/middleware/resume_quota.go
  - hr-frontend/src/api/recruitingIntelligence.ts
  - user-frontend/src/api/resume.ts
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Debug Resume Intelligence

Use this runbook when resume upload, parsed text, structured profile, AI parsing, match evaluation, evidence display, or candidate comparison is wrong or missing.

## 1. Locate the Failing Stage

- Upload/presign: candidate cannot get an upload URL, upload to OSS fails, or confirm fails.
- Text extraction: resume row exists but `parsed_text` is empty or invalid.
- Structured profile: HR parse returns missing text, extractor error, invalid output, or failed parse run.
- Matching: evaluation returns missing application, job, resume, current profile, incomplete profile, or scoring error.
- Display: backend has profile/evaluation data but frontend intelligence page shows missing, stale, or malformed evidence.

## 2. Upload and Quota Checks

Check candidate route permission, risk-block middleware, presign quota, confirm quota, file extension, frontend size limit, content type, presign expiry, upload ID, OSS object existence, and object size verification. For 429 responses, record quota type and reset time instead of retrying in a tight loop.

## 3. Parser Checks

Confirm the file type is supported by the parser registry and magic bytes match the claimed type. Inspect extraction logs for resume ID, file type, object key, text length, cleaning statistics if available, and parser error. Do not paste raw resume text into reports.

## 4. Structured Profile Checks

For profile parse failures, inspect:

- resume ID and whether `parsed_text` exists;
- runtime policy for structured resume parsing and timeout;
- default LLM model availability;
- active DB prompt for `resume_profile_extractor`;
- fallback extractor enabled state;
- parse-run status, parser version, input hash, error message, and current profile version.

Use placeholder snippets such as `[candidate summary omitted]` when a report needs to describe content shape.

## 5. Matching Checks

For match failures, inspect:

- application ID, job ID, candidate user ID, resume ID, and current resume profile ID;
- HR user permissions for application read and AI HR use;
- runtime policy for candidate match, semantic match, and shadow mode;
- active DB prompt for `candidate_match_evaluator` if the LLM requirement matcher is involved;
- latest evaluation version, score, recommendation, dimensions, missing requirements, and evidence count.

## 6. Frontend Checks

Candidate upload UI validates PDF/DOCX and size before presign. HR intelligence UI uses `getApplicationResumeProfile`, `parseResumeProfile`, `evaluateCandidateMatch`, and `getCandidateMatchEvaluation`. If the UI and API disagree, compare route params, business error code, silent-error handling, and JSON parsing of score/evidence fields.

## Suggested Verification

- `go test ./...` from `logic-grpc-service/` when changing parser, OSS, profile, extractor, match, repository, or migration code.
- `go test ./...` from `web-gin-service/` when changing candidate upload or recruiting intelligence handlers/middleware.
- `pnpm --filter user-frontend typecheck` for candidate upload frontend changes.
- `pnpm --filter hr-frontend typecheck` for HR intelligence frontend changes.

## Evidence to Record

Record IDs, statuses, versions, model names, prompt keys/versions, runtime policy flags, business codes, quota reset data, evaluation/evidence counts, and test commands. Exclude names, phone numbers, emails, raw resume text, raw prompt content, raw LLM output, object-storage URLs, and private object keys unless a TASK explicitly authorizes secure handling.
