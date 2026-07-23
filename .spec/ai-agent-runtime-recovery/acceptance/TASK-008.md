# Acceptance - TASK-008

## TASK Summary

Restore embedding model runtime validation, Agent Skill embedding backfill, semantic retrieval debug, and semantic Agent Skill selection.

## SPEC References

- FR-008 Embedding and semantic retrieval restoration
- Observability and Debug Requirements 6
- Error Handling and Fallback Requirements 6
- Acceptance Criteria 12

## SDD References

- Proposed Design 3.8
- Algorithm or Workflow Changes 6.6
- Configuration Design
- Error Handling and Fallback Design
- Testing Strategy

## Acceptance Criteria

- Embedding model test calls the configured runtime provider/model or returns explicit non-success runtime error.
- Agent Skill embedding backfill supports `agent_skill` objects.
- Agent Skill changes trigger or publish embedding upsert/invalidation behavior consistent with dev reference.
- Semantic Retrieval Debug returns provider/model/dim, latency, scores, candidates, and fallback reason.
- Agent Skill selection uses semantic scores when available and explicit rule fallback otherwise.

## Required Checks

- Targeted AI Agent service tests for embedding model test, backfill, semantic debug, selection, and fallback.
- Gateway or frontend tests if mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `GOWORK=off go test ./...` from `smart-recruit-gateway` if gateway is touched.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-008`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

Live embedding smoke may be skipped if provider credentials are unavailable; document fake-provider evidence.

## Out-of-Scope

- New dependencies without confirmation.
- Database migrations.
- MCP tool execution.
