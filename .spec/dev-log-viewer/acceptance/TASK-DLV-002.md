# Acceptance - TASK-DLV-002

## TASK Summary

实现固定服务目录、PID/日志状态、loopback 配置、`/healthz` 和 `/api/v1/services`。

## SPEC References

- UI-006 至 UI-010
- FR-001 至 FR-004
- OD-003、EH-001、EH-002
- SS-001 至 SS-004
- AC-001、AC-011、AC-012

## SDD References

- 3.3 后端职责
- 4 数据结构变更
- 5.1、5.2 API
- 6.1、6.2 工作流
- 7 配置设计

## Acceptance Criteria

- 目录准确返回 SPEC UI-007 的 12 个服务、分组、端口、PID 和日志状态。
- PID 缺失、空、非数字、陈旧和权限错误均有确定结果，且只称为 process state。
- 根目录由显式配置解析；API 响应不泄露绝对路径。
- HTTP 地址拒绝非 loopback 配置。
- `/healthz` 只表示查看器进程；服务 API 不接受任意路径参数。
- 单个状态检查失败不导致整个目录接口失败。

## Required Checks

- `cd dev-log-viewer && go test ./internal/catalog ./internal/config ./internal/server ./cmd/dev-log-viewer`
- `cd dev-log-viewer && go vet ./...`
- `gofmt -l` 对 TASK Go 文件无输出。
- Harness scope 与 agent-check。

## Manual Verification, if needed

无；状态使用临时目录和受控进程 fixture 自动验证。

## Out-of-Scope

端口健康探测、日志内容读取、SSE、UI、任意自定义服务。
