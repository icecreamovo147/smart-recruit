# Acceptance - TASK-DLV-003

## TASK Summary

实现有界 snapshot/follow/reset 与混合日志解析，不接入 HTTP。

## SPEC References

- UI-011 至 UI-019、UI-034
- FR-005 至 FR-011
- NFR-001 至 NFR-005
- EH-002 至 EH-005
- AC-002、AC-003、AC-004、AC-011

## SDD References

- 2.1 文件读取难点
- 3.3 tailer/parser 职责
- 4.2 LogRecord
- 6.2 至 6.4 算法
- 9.1 后端降级

## Acceptance Criteria

- 默认读取最后 300 行，最大请求固定为 1000；空文件和无换行结尾有确定行为。
- 正确处理跨块 UTF-8、半行、append、truncate、rename/recreate、delete/reappear。
- reset 事件携带服务和代次，不把旧 offset 用于新文件。
- ANSI 被移除；Zap、JSON 尾部、Gin、Vite、多行堆栈和 UNKNOWN 降级均保留原文本语义。
- 提取允许的关联字段，超长记录按明确边界截断并标记。
- 某一文件错误不停止其他 tailer；轮询没有忙循环。

## Required Checks

- `cd dev-log-viewer && go test ./internal/tailer ./internal/parser -race`
- `cd dev-log-viewer && go vet ./...`
- `gofmt -l` 对 TASK Go 文件无输出。
- Harness scope 与 agent-check。

## Manual Verification, if needed

无；不得使用当前 `.dev/logs` 中可能含敏感信息的原始内容作为 fixture。

## Out-of-Scope

SSE、浏览器状态、历史索引、正则查询、日志脱敏规则引擎。
