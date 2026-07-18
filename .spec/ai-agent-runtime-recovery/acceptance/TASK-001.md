# Acceptance - TASK-001

## TASK Summary

Repair the validation baseline by resolving missing Go checksum entries and create the dev-branch reference inventory for later recovery work.

## SPEC References

- FR-009 Validation and dependency hygiene
- Compatibility Requirements 7
- Acceptance Criteria 13, 14
- Assumptions Requiring Confirmation 1, 5

## SDD References

- Problem Analysis 10
- Data Structure Changes
- Testing Strategy
- Migration Risks 8

## Acceptance Criteria

- `smart-recruit-ai-agent-service` no longer fails test startup because of missing `go.sum` entries.
- `smart-recruit-gateway` no longer fails test startup because of missing `go.sum` entries.
- `.spec/ai-agent-runtime-recovery/docs/dev-reference-inventory.md` maps dev reference files to current target areas for HR AI, Candidate AI, Agent Run, Recruiting Intelligence, MCP, Agent Skill, and Embedding.
- No production source code or runtime behavior is changed.
- Any newly exposed compile/test failures are documented in the TASK report without expanding scope.

## Required Checks

- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway`.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-001`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

None required.

## Out-of-Scope

- Runtime implementation.
- `go.mod` changes.
- Protobuf, schema, auth, frontend, or K8s changes.
