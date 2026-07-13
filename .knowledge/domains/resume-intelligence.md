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
  - smart-recruit-ai-agent-service/internal/legacydomain/service/*resume*.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_*.go
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
source_refs:
  - smart-recruit-commons/oss/storage.go
  - smart-recruit-commons/resumeparser/parser.go
  - smart-recruit-commons/resumeparser/registry.go
  - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
  - smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_extractor_llm.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/resume_profile_extractor_fallback.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/recruiting_intelligence_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_service.go
  - smart-recruit-ai-agent-service/internal/legacydomain/service/candidate_match_llm_matcher.go
  - smart-recruit-gateway/handler/candidate/resume.go
  - smart-recruit-gateway/handler/hr/recruiting_intelligence.go
  - smart-recruit-gateway/middleware/resume_quota.go
last_verified: 2026-07-14
review_after: 2026-10-14
---

# Resume Intelligence and Candidate Matching

Resume upload and ownership validation are Recruitment/Gateway concerns, shared OSS and parser support lives in Commons, and AI-driven profile extraction plus matching currently live in AI Agent service adapters. Protect candidate-sensitive data and avoid raw resume text in logs, metrics, reports, or prompts beyond scoped processing.

## Verification

Verified against current repository files on 2026-07-14.
