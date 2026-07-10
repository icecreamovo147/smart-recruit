---
schema_version: 1
id: resume-sensitive-data
title: Resume sensitive data handling
kind: pitfall
status: active
owners:
  - recruitment-domain
  - security
tags:
  - resume
  - pii
  - sensitive-data
  - ai
applies_to:
  - logic-grpc-service/oss/**
  - logic-grpc-service/resumeparser/**
  - logic-grpc-service/service/resume_profile_service.go
  - logic-grpc-service/service/resume_profile_extractor_llm.go
  - logic-grpc-service/service/candidate_match_llm_matcher.go
  - logic-grpc-service/service/recruiting_intelligence_service.go
  - web-gin-service/handler/candidate/resume.go
  - web-gin-service/handler/hr/recruiting_intelligence.go
  - hr-frontend/src/views/hr/ApplicationIntelligenceView.vue
  - user-frontend/src/views/candidate/ResumeUploadView.vue
source_refs:
  - logic-grpc-service/model/model.go
  - logic-grpc-service/oss/storage.go
  - logic-grpc-service/service/resume_profile_service.go
  - logic-grpc-service/service/resume_profile_extractor_llm.go
  - logic-grpc-service/service/candidate_match_llm_matcher.go
  - logic-grpc-service/service/recruiting_intelligence_service.go
  - web-gin-service/handler/candidate/resume.go
  - web-gin-service/handler/hr/recruiting_intelligence.go
  - web-gin-service/middleware/resume_quota.go
last_verified: 2026-07-10
review_after: 2026-10-08
---

# Resume Sensitive Data Handling

Resume flows handle personal data and AI-derived assessment data. Active knowledge may describe fields and behavior, but it should not include real candidate content or storage credentials.

## Sensitive Surfaces

- Object-storage keys, presigned upload/download URLs, upload IDs, and object metadata.
- Raw resume files and extracted `parsed_text`.
- Structured profile fields such as name, email, phone, location, education, experience, projects, skills, summary, and raw JSON.
- Prompt inputs and LLM outputs that include resume text or candidate evidence.
- Candidate match evidence snippets, strengths, risks, missing requirements, summaries, and score breakdowns.
- Logs that include safe previews, model names, prompt keys, input hashes, durations, and error messages.

## Common Failure Modes

- Copying raw resume text, raw JSON, or evidence snippets into TASK reports or active knowledge.
- Treating presigned URLs or object keys as harmless debug strings.
- Logging or surfacing more candidate data than needed to debug an extractor or matcher failure.
- Reusing personal-data examples in tests or documentation instead of placeholders.
- Changing frontend display logic so hidden evidence, raw JSON, or LLM output becomes visible without review.
- Changing parser, extractor, or matcher prompts without reviewing how much candidate content is sent to the model.

## Safe Reporting Pattern

Use structural placeholders:

- `[resume text omitted]`
- `[candidate profile JSON omitted]`
- `[evidence snippet omitted]`
- `[presigned URL omitted]`
- `[object key omitted]`

Record non-sensitive operational details instead: resume ID, application ID, profile ID, parse-run status, parser version, input hash, model name, prompt key/version, evaluation ID/version, evidence count, business code, and command results.

## Review Triggers

- New resume file formats, parsing logic, text-cleaning rules, or max text length.
- New LLM prompts or prompt input fields for resume profile extraction or match evaluation.
- New evidence fields, raw JSON display, download URL behavior, or frontend preview behavior.
- Changes to quota, risk-block, authorization, or data-scope checks around resume and intelligence routes.
- Logging changes in parser, extractor, matcher, or HR intelligence handlers.

## Verification

Verified against current resume storage, parser, profile extraction, LLM matcher, recruiting intelligence, upload handler, HR handler, and quota middleware sources on 2026-07-10.
