# Acceptance - TASK-005

## TASK Summary

Restore Recruiting Intelligence generation for resume profile parsing and candidate match evaluation.

## SPEC References

- FR-005 Recruiting Intelligence generation restoration
- Security and Safety Requirements 3, 6
- Acceptance Criteria 7, 8, 9

## SDD References

- Proposed Design 3.5
- Algorithm or Workflow Changes 6.4
- Error Handling and Fallback Design
- Testing Strategy

## Acceptance Criteria

- `ParseResumeProfile` generates and persists a resume profile snapshot for fresh data.
- `EvaluateCandidateMatch` generates and persists candidate match evaluation and evidence for fresh data.
- `agent_run_id` association is preserved when provided.
- Existing read APIs for profile/evaluation/comparison remain compatible.
- Authorization and HR AI permission checks remain intact.
- Raw candidate/resume sensitive data is not written into TASK reports.

## Required Checks

- Targeted AI Agent service tests for parse/evaluate success and authorization failures.
- Gateway handler tests if HTTP mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway` if gateway is touched.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-005`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

If no local resume/application data exists, document skipped live parse/evaluate smoke and provide fake persistence/provider test evidence.

## Out-of-Scope

- Schema migrations without confirmation.
- Protobuf changes.
- Candidate AI chat runtime.
