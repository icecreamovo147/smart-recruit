# Acceptance - TASK-002

## TASK Summary

Restore HR AI Chat and ChatStream runtime semantics using dev-compatible tool calling, context usage, tool traces, and fallback behavior.

## SPEC References

- FR-001 HR AI runtime restoration
- FR-006 Agent, Prompt, Skill, and Capability runtime integration
- Observability and Debug Requirements 1, 2, 3, 7
- Error Handling and Fallback Requirements 1, 2, 3
- Acceptance Criteria 1, 2

## SDD References

- Proposed Design 3.1, 3.2
- Algorithm or Workflow Changes 6.1
- Compatibility Strategy
- Error Handling and Fallback Design
- Testing Strategy

## Acceptance Criteria

- HR Chat does not simply pass the raw prompt to provider `complete()`; it builds recruitment-aware context and uses approved tools.
- HR ChatStream emits compatible stream events for model/status/tool/context/fallback/done states.
- Tool traces are persisted and retrievable for the session.
- Context usage is computed and returned/emitted where applicable.
- Deterministic fallback is used when provider output fails after useful tool results.
- The TASK report cites the dev reference implementation used for parity.

## Required Checks

- Targeted AI Agent service tests for HR Chat and ChatStream.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-002`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

If live provider credentials are unavailable, document skipped live HR AI smoke checks and provide fake-provider test evidence.

## Out-of-Scope

- Candidate AI runtime.
- Durable Agent Run runtime.
- MCP live runner.
- Protobuf, schema, K8s, or repository-root docs changes.
