# Acceptance - TASK-027

## TASK Summary

迁移 Worker workload application/runtime。

## SPEC References

- FR-015
- NFR-007
- SSR-006

## SDD References

- 6. Algorithm or Workflow Changes
- 8. Compatibility Strategy

## Acceptance Criteria

- Workload 通过 profile/toggle 分类运行。
- 后台写入遵守 owner contract。
- 覆盖 workload parse/toggle/graceful shutdown 测试。

## Required Checks

- Worker targeted tests
- `go test ./...` in `smart-recruit-worker-service`
- scope check
- agent-check

## Manual Verification, if needed

确认不新增绕过 owner service/domain 的写路径。

## Out-of-Scope

新增 workload、proto/schema 修改。
