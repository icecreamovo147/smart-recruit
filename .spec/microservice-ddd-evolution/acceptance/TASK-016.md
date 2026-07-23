# Acceptance - TASK-016

## TASK Summary

迁移 Recruitment job/candidate/resume domain/application。

## SPEC References

- FR-002
- FR-003
- FR-012

## SDD References

- 6. Algorithm or Workflow Changes
- 11. Testing Strategy

## Acceptance Criteria

- Job/Candidate/Resume domain/application 本地化。
- OSS presign、resume confirm、usage log、outbox 语义兼容。
- 覆盖 job lifecycle 和 resume upload confirm 测试。

## Required Checks

- Recruitment targeted tests
- `go test ./...` in `smart-recruit-recruitment-service`
- scope check
- agent-check

## Manual Verification, if needed

若 AI Agent 读依赖需要新 API，停止确认。

## Out-of-Scope

Application/collaboration 迁移、schema/proto 修改。
