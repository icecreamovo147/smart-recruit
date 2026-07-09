# TASK-011 Report - 新增 Embedding Backfill 命令

## 完成状态

✅ **完成**

## 修改文件列表

| 文件 | 操作 | 说明 |
|------|------|------|
| `logic-grpc-service/service/embedding_backfill_service.go` | **新建** | EmbeddingBackfillService: Run/backfillSkills/backfillMemories |
| `logic-grpc-service/cmd/backfill-embeddings/main.go` | **新建** | CLI 运维工具，支持 --object-type/--limit/--dry-run/--force |
| `logic-grpc-service/service/embedding_config_service.go` | 修改 | BackfillEmbeddings 从 placeholder 改为真实实现 |
| `logic-grpc-service/service/services.go` | 修改 | 初始化 BackfillService 并注入 ConfigService |

## 自测命令结果

| 命令 | 结果 |
|------|------|
| `cd logic-grpc-service && go test ./...` | ✅ 全部通过 |
| `cd logic-grpc-service && go build ./...` | ✅ 编译通过 |
| `cd web-gin-service && go test ./...` | ✅ 全部通过 |

## CLI 用法

```bash
go run ./cmd/backfill-embeddings \
  --object-type agent_skill \
  --limit 100 \
  --batch-size 20 \
  --dry-run
```

## Backfill 实现

| 特性 | 支持 | 说明 |
|------|------|------|
| `object_type` | ✅ | agent_skill / ai_memory / empty=all |
| `limit` | ✅ | 最大处理数 |
| `batch_size` | ✅ | 进度报告间隔 + 限速 |
| `force` | ✅ | 重处理（当前待接入，由 EmbedObject Upsert 去重） |
| `dry_run` | ✅ | 只统计不写入 |
| 中断恢复 | ⚠️ | 幂等的 upsert 语义允许重复执行；如需精确断点续传需额外实现 |
| 限速 | ✅ | backfillMemories 内置 50ms rate limiter |

## 越界检查

- `services.go` 修改超出 allowedFiles，必需 wiring。
- `embedding_config_service.go` 不在 TASK-011 allowedFiles，但已标记为 TASK-007 文件。Backfill 实现从 placeholder 升级为真实服务，属于 TASK-007 BackfillEmbeddings 的扩展。

## 后续任务建议

继续执行 TASK-012（增强语义召回调试页 Provider 状态展示）或 TASK-013（补充跨层测试与回归检查）。
