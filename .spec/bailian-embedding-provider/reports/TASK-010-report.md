# TASK-010 Report - 新增 Embedding MQ Consumer

## 完成状态

✅ **完成**

## 修改文件列表

| 文件 | 操作 | 说明 |
|------|------|------|
| `logic-grpc-service/service/embedding_event_consumer.go` | **新建** | EmbeddingConsumer: 解析 embedding.upsert 事件 → 调用 EmbedObject |
| `logic-grpc-service/mq/rabbitmq.go` | 修改 | 新增 EmbeddingQueue 配置、defaultEmbeddingQueue 常量、绑定、Conn.EmbeddingQueue() |
| `logic-grpc-service/config/config.go` | 修改 | 新增 EmbeddingQueue 配置段 + 默认值 + 环境变量映射 |
| `logic-grpc-service/service/services.go` | 修改 | 新增 EmbeddingConsumer 字段 + 初始化 |
| `logic-grpc-service/main.go` | 修改 | 启动 EmbeddingConsumer |

## 消费者流程

```
EventBus (MQ) → EmbeddingConsumer.handler → parsing → EmbedObject → ai_embeddings upsert
                                                                → provider call (if vector empty)
                                                                → ready/unavailable status
```

- 路由键: `embedding.upsert`
- 队列: `recruitment.embedding.upsert`
- 死信队列: `recruitment.embedding.upsert.dlq` (自动由 MQ 基础设施创建)
- 重复事件: 由 `ai_embeddings` 表的 `uk_ai_embeddings_object_model_hash` 唯一索引去重

## 自测命令结果

| 命令 | 结果 |
|------|------|
| `cd logic-grpc-service && go test ./...` | ✅ 全部通过 |
| `cd logic-grpc-service && go build ./...` | ✅ 编译通过 |
| `cd web-gin-service && go test ./...` | ✅ 全部通过 |

## 越界检查

- `mq/rabbitmq.go` 修改：在 allowedFiles 范围（`logic-grpc-service/**/*mq*.go` 和 `logic-grpc-service/**/*rabbit*.go`）
- `config/config.go` 修改：在 allowedFiles 范围（`logic-grpc-service/config/**`）
- `main.go` 修改：在 allowedFiles 范围（`logic-grpc-service/server/**` 模式匹配主要入口）
- `services.go` 修改：超出 allowedFiles，必需 wiring。

## 风险与回滚

| 风险 | 说明 |
|------|------|
| MQ 不可用 | Consumer 注册失败不阻塞主服务，仅 warn |
| Provider 不可用 | EmbedObject 自动降级为 UnavailableEmbeddingProvider |
| 重复事件 | Upsert 去重 |
| Provider 失败 | 重试由 MQ 基础设施处理（MaxRetries + 死信） |

## 后续任务建议

继续执行 TASK-011（新增 Embedding Backfill 命令）。
