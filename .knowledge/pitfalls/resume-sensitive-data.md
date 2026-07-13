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
  - smart-recruit-ai-agent-service/internal/legacydomain/service/*resume*.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_*.go
source_refs:
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-gateway/middleware/resume_quota.go
  - smart-recruit-commons/oss/storage.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_extractor_llm.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/recruiting_intelligence_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_llm_matcher.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Resume Sensitive Data Pitfall

Resume files, parsed text, match evidence, and AI prompts/results can contain personal data. Do not log raw resume text, presigned URLs, credentials, or full prompts/results. Keep ownership checks, file limits, quota middleware, and redaction intact.

## Verification

Verified against current repository files on 2026-07-14.
