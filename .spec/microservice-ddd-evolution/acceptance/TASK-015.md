# Acceptance - TASK-015

## TASK Summary

创建 Recruitment DDD 骨架并完成核心域盘点。

## SPEC References

- FR-001 至 FR-007
- FR-012

## SDD References

- 3.4 Per-Service Migration Pattern
- 12. Migration Risks

## Acceptance Criteria

- Recruitment DDD 骨架存在。
- job/candidate/resume/application/collaboration/taxonomy/usage stats 盘点完成。
- 不改变招聘 API 行为。

## Required Checks

- `go test ./...` in `smart-recruit-recruitment-service`
- scope check
- agent-check

## Manual Verification, if needed

确认前序服务边界未被回退。

## Out-of-Scope

Recruitment 业务迁移、shared cleanup、schema/proto 修改。
