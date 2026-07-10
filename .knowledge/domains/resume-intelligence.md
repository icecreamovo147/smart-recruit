---
schema_version: 1
id: resume-intelligence
title: Resume intelligence and matching
kind: domain
status: active
owners:
  - recruitment-domain
  - agent-platform
tags:
  - resume
  - intelligence
  - matching
  - ai
  - sensitive-data
applies_to:
  - logic-grpc-service/oss/**
  - logic-grpc-service/resumeparser/**
  - logic-grpc-service/service/resume_profile_service.go
  - logic-grpc-service/service/resume_profile_extractor_*.go
  - logic-grpc-service/service/recruiting_intelligence_service.go
  - logic-grpc-service/service/candidate_match_*.go
  - web-gin-service/handler/candidate/resume.go
  - web-gin-service/handler/hr/recruiting_intelligence.go
  - user-frontend/src/views/candidate/ResumeUploadView.vue
  - hr-frontend/src/views/hr/ApplicationIntelligenceView.vue
source_refs:
  - logic-grpc-service/model/model.go
  - logic-grpc-service/oss/storage.go
  - logic-grpc-service/resumeparser/parser.go
  - logic-grpc-service/resumeparser/registry.go
  - logic-grpc-service/service/resume_parse_consumer.go
  - logic-grpc-service/service/resume_profile_service.go
  - logic-grpc-service/service/resume_profile_extractor_llm.go
  - logic-grpc-service/service/resume_profile_extractor_fallback.go
  - logic-grpc-service/service/recruiting_intelligence_service.go
  - logic-grpc-service/service/candidate_match_service.go
  - logic-grpc-service/service/candidate_match_llm_matcher.go
  - web-gin-service/handler/candidate/resume.go
  - web-gin-service/handler/hr/recruiting_intelligence.go
  - web-gin-service/middleware/resume_quota.go
  - web-gin-service/router/router.go
  - user-frontend/src/views/candidate/ResumeUploadView.vue
  - hr-frontend/src/views/hr/ApplicationIntelligenceView.vue
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Resume Intelligence and Matching

Resume intelligence turns a candidate-uploaded resume file into parsed text, a structured resume profile, and a candidate-job match evaluation. The chain crosses candidate upload, object storage, asynchronous text extraction, HR-triggered profile parsing, AI or heuristic extraction, candidate match scoring, evidence persistence, and HR review UI.

## Upload and Storage

- Candidate resume upload uses presign and confirm endpoints under candidate routes.
- `ResumePresignQuota` and `ResumeConfirmQuota` enforce per-user hourly and daily limits and can block suspicious presign-without-confirm behavior.
- The frontend accepts PDF and DOCX up to the current UI limit and uploads to the presigned object-storage URL before confirming through the gateway.
- OSS storage abstracts presigned PUT/GET URLs, object verification, object size verification, downloads, copies, deletes, and presign sessions.
- `ContentTypeFromFileType` maps logical file types to deterministic MIME types for signed uploads.

## Text Extraction

`ResumeParseConsumer` consumes resume parse events, downloads the stored object, extracts text, and updates the resume record. The parser registry currently covers PDF and DOCX. Text preparation cleans lines, removes likely noise, limits analysis text length, and rejects incoherent or too-short parsed text.

## Structured Profile

`ResumeProfileService.ParseResume` loads parsed text, hashes the input, calls the configured extractor, validates output, builds a versioned profile snapshot, and saves parse-run status. The LLM extractor requires a default LLM model and active DB prompt for `resume_profile_extractor`; the fallback wrapper can use a heuristic extractor when enabled.

Structured profile persistence includes parse runs, profile core fields, education, experience, project, and skill rows. Only one current profile version should be used for current matching.

## Candidate Match

`RecruitingIntelligenceService` is the HR-facing orchestrator for profile lookup, parse trigger, candidate match evaluation, and candidate comparison. It checks application/job access and AI HR permission before parse or evaluate actions.

`CandidateMatchService.EvaluateApplication` loads the application, job, candidate profile, resume, current structured profile snapshot, and then scores. It can use semantic-enhanced scoring depending on runtime policy. Evaluations are versioned and persisted with evidence rows. The LLM requirement matcher uses the default LLM model and active DB prompt for `candidate_match_evaluator`.

## Frontend Surfaces

- Candidate resume upload lives in `user-frontend/src/views/candidate/ResumeUploadView.vue`.
- HR intelligence review lives in `hr-frontend/src/views/hr/ApplicationIntelligenceView.vue`.
- HR API helpers are in `hr-frontend/src/api/recruitingIntelligence.ts`; candidate upload helpers are in `user-frontend/src/api/resume.ts`.

## Review Triggers

- Upload file type, size, content type, presign session, quota, or OSS verification changes.
- Parser registry, magic byte validation, text cleaning, max analysis length, or extraction failure behavior.
- LLM extractor prompt key, default model usage, fallback behavior, parse-run status, or profile versioning changes.
- Candidate match scoring, semantic policy, LLM requirement matcher, evidence generation, evaluation versioning, or comparison sorting changes.
- Permission or route changes for resume upload, profile parse, match evaluation, or match read APIs.

## Verification

Verified against current OSS, resume parser, resume profile, extractor, recruiting intelligence, candidate match, quota, gateway handler, router, and frontend upload/intelligence sources on 2026-07-10.
