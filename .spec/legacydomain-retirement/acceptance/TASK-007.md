# Acceptance - TASK-007

## TASK Summary

Cut AI Agent runtime over from legacy service graph to native application, infrastructure, and gRPC adapters.

## SPEC References

- FR-007, FR-008, FR-009
- EHF-002
- SSR-002, SSR-003, SSR-004

## SDD References

- Sections 3, 5, 6, 8, 9, 11, 12, 13

## Acceptance Criteria

- Active AI Agent runtime no longer constructs legacy service graph.
- Native adapters cover AI chat, candidate chat, agent runs, prompt/config, MCP, skill, embedding, recruiting intelligence, and workers.
- `interfaces/grpc/legacy_servers.go` is replaced by native adapters.
- Provider calls remain fake/env-gated in tests.
- AI Agent `legacydomain` is not deleted in this TASK.

## Required Checks

- `go test ./...` in `smart-recruit-ai-agent-service`
- `node scripts/check-backend-boundaries.mjs`
- `node scripts/check-mysql-table-ownership.mjs`
- `bash .spec/legacydomain-retirement/scripts/check-task-scope.sh TASK-007`
- `bash .spec/legacydomain-retirement/scripts/agent-check.sh`

## Manual Verification, if needed

None.

## Out-of-Scope

Deleting AI Agent legacydomain, schema changes, and public API behavior changes.
