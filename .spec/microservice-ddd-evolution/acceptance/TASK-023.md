# Acceptance - TASK-023

## TASK Summary

迁移 AI Agent chat/agent-run/prompt domain/application。

## SPEC References

- FR-002
- FR-003
- FR-014

## SDD References

- 6. Algorithm or Workflow Changes
- 9. Error Handling and Fallback Design

## Acceptance Criteria

- Chat/session/agent run/prompt/model config 本地化。
- stream、cancel、confirm、audit、provider fallback 兼容。
- 覆盖 agent run state/durable/prompt 测试。

## Required Checks

- AI Agent targeted tests
- `go test ./...` in `smart-recruit-ai-agent-service`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认。

## Out-of-Scope

新增 provider 行为、schema/proto 修改。
