# Acceptance - TASK-DLV-004

## TASK Summary

实现 EventHub、有界订阅、SSE、`Last-Event-ID` 恢复和后端完整实时链路。

## SPEC References

- UI-029 至 UI-034
- FR-012 至 FR-018
- NFR-001 至 NFR-005
- OD-001、OD-002、EH-004、EH-005
- AC-002、AC-003、AC-005、AC-006、AC-011、AC-012

## SDD References

- 2.2 实时传输难点
- 3.2、3.3 组件与后端职责
- 4.3 StreamEnvelope
- 5.3 SSE API
- 6.5 事件环和重连
- 9.1、10 可观测性

## Acceptance Criteria

- 单调 EventID、20,000 事件环、2,048 客户端队列均有边界测试。
- 新连接获得有界 snapshot；可恢复 ID 顺序补发，不可恢复 ID 显式 reset 后重新 snapshot。
- SSE 设置正确 content type、禁用缓存并定期 heartbeat。
- 服务筛选只能引用白名单 ID，未知服务返回确定的 4xx。
- 慢客户端只影响自身并收到合并 dropped；其他客户端持续接收。
- 断开、取消和服务器退出释放 goroutine、订阅和文件资源。
- 查看器日志不包含被查看正文、关联 ID 值或绝对用户路径。

## Required Checks

- `cd dev-log-viewer && go test ./internal/stream ./internal/server ./internal/tailer -race`
- `cd dev-log-viewer && go test ./...`
- `cd dev-log-viewer && go vet ./...`
- Harness scope 与 agent-check。

## Manual Verification, if needed

无；SSE 客户端并发和断线使用自动化测试。

## Out-of-Scope

WebSocket、持久化事件、业务 Gateway、静态前端嵌入。
