# Acceptance - TASK-004

## TASK Summary

Restore Candidate AI runtime behavior and the missing non-streaming candidate chat compatibility route.

## SPEC References

- FR-004 Candidate AI restoration
- FR-010 Compatibility of public contracts
- Security and Safety Requirements 2
- Acceptance Criteria 4, 5

## SDD References

- Proposed Design 3.4
- API and Interface Changes
- Algorithm or Workflow Changes 6.3
- Compatibility Strategy
- Testing Strategy

## Acceptance Criteria

- Candidate ChatStream uses candidate-scoped tools for candidate-owned applications, resumes, jobs, interviews, and offers.
- Candidate responses persist assistant content and restore usage audit/auth context behavior where dev did so.
- Suggested questions are extracted or generated through fallback.
- `POST /api/v1/candidate/ai/chat` is registered and returns the current frontend response shape.
- Candidate data isolation is preserved.

## Required Checks

- Targeted AI Agent service tests for candidate runtime, tool fallback, suggested questions, and audit writes.
- Gateway route/handler test for `POST /api/v1/candidate/ai/chat`.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway`.
- `pnpm --filter user-frontend typecheck` if user frontend is touched.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-004`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

Candidate AI live smoke may be skipped if no local candidate session/data is available; record the reason and test substitute.

## Out-of-Scope

- HR runtime changes.
- Proto changes unless explicitly confirmed.
- New candidate UX.
