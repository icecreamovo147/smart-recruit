# Acceptance - TASK-022

## TASK Summary

创建 AI Agent DDD 骨架并盘点复杂依赖。

## SPEC References

- FR-001 至 FR-007
- FR-014
- SSR-005

## SDD References

- 3.3 Required Migration Order
- 12. Migration Risks

## Acceptance Criteria

- AI Agent DDD 骨架存在。
- chat/agent run/prompt/MCP/skill/embedding/intelligence/provider/audit 依赖盘点完成。
- 不改变 AI API 行为。

## Required Checks

- `go test ./...` in `smart-recruit-ai-agent-service`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认。

## Out-of-Scope

真实 provider 调用、proto/schema/security 变更。
