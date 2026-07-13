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
  - smart-recruit-ai-agent-service/internal/legacydomain/service/*resume*.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_*.go
  - smart-recruit-commons/resumeparser/**
source_refs:
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-gateway/middleware/resume_quota.go
  - smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service.go
  - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
  - smart-recruit-commons/resumeparser/parser.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_extractor_llm.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/recruiting_intelligence_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_service.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Debug Resume Intelligence

Verify gateway quota/body limits, Recruitment resume ownership and upload-session validation, Commons OSS/parser behavior, AI Agent profile extraction and candidate matching, and privacy-safe logs/reports.

## Verification

Verified against current repository files on 2026-07-14.
