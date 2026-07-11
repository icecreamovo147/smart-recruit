# Acceptance - TASK-BDME-012

## TASK Summary

Recruitment Candidate Resume Application Boundary: Move candidate profile, resume, and application lifecycle ownership into Recruitment.

## SPEC References

- SPEC §5 FR-001..FR-008
- SPEC §11 AC-004..AC-008

## SDD References

- SDD §3.2 Target DDD Package Shape
- SDD §3.4 Target Ownership Matrix

## Acceptance Criteria

- Candidate recruitment profile, resumes, and applications are Recruitment-owned.
- AI-derived profiles, embeddings, matching artifacts, memory, and intelligence remain AI Agent-owned.
- Application lifecycle tests preserve current visible behavior.

## Required Checks

- git diff --name-only
- bash .spec/backend-ddd-microservices-evolution/scripts/check-task-scope.sh TASK-BDME-012
- bash .spec/backend-ddd-microservices-evolution/scripts/agent-check.sh
- cd logic-grpc-service && go test ./...
- node .knowledge/scripts/validate-knowledge.mjs if .knowledge files changed

## Manual Verification, if needed

- Manual verification is required only when automated checks cannot cover an acceptance criterion; skipped checks must be recorded with reasons.

## Out-of-Scope

- Changes outside task-scope.json allowedFiles for this TASK.
- Modifications to SPEC, SDD, TASKS.md, AGENT_RULES.md, task-scope.json, acceptance files, prompts, or Harness scripts.
- Frontend application changes.
- Package manifest, lockfile, go.mod, go.sum, dependency, CI/CD, or global configuration changes unless explicitly listed in scope and confirmed.
- Product behavior changes not stated in this TASK acceptance file.
