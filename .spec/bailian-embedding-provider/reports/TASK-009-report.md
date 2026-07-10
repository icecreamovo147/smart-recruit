# TASK-009 Report - AI Memory 写入链路发布 Embedding MQ 事件

## 完成状态

✅ **完成**

## AI Memory 创建/更新入口定位报告

### 入口清单

| 入口 | 文件 | 行号 | 方式 | 是否接入事件 |
|------|------|------|------|-----------|
| `AIService.writeMemory` | `ai_service.go` | 1302 | `memories.Create(ctx, memory)` 直接创建 | ✅ |
| **总入口数** | | **1** | | **1/1 已接入** |

**分析过程：**
1. 搜索 `memoryRepo.Create` 调用 — 仅在 `ai_service.go` 的 `writeMemory` 方法中发现
2. 搜索 `model.AIMemory{` 实例化 — 仅在 `ai_service.go` 的 `writeMemory` 中发现
3. MemoryRepo 无 `Update` 方法，说明 Memory 只有创建语义，没有原地更新
4. `writeMemory` 被 `analysisComplete` 方法在 `AnalyzeApplication` 完成后调用

### 更新路径

当前代码中，AI Memory **没有** 服务层 Update 路径。`MemoryRepo` 仅提供 `Create` 方法。  
`writeMemory` 只在 application analysis 完成后创建 `conclusion` 类型 memory。  
非内容字段更新（如 importance/confidence）不会触发 embedding 事件，与「非内容更新不发布」一致。

## 修改文件列表

| 文件 | 操作 | 说明 |
|------|------|------|
| `logic-grpc-service/service/ai_service.go` | 修改 | 新增 eventPublisher 字段 + WithEmbeddingEventPublisher + publishMemoryEmbeddingEvent + writeMemory 调用 |
| `logic-grpc-service/service/services.go` | 修改 | AI 服务注入 WithEmbeddingEventPublisher |
| `logic-grpc-service/service/embedding_event_publisher.go` | 已有 | 复用 TASK-008 创建的 EventPublisher |
| `logic-grpc-service/service/embedding_text_builder.go` | 已有 | 复用 TASK-008 创建的 BuildMemoryEmbeddingText |

## 自测命令结果

| 命令 | 结果 |
|------|------|
| `cd logic-grpc-service && go test ./...` | ✅ 全部通过 |
| `cd logic-grpc-service && go build ./...` | ✅ 编译通过 |
| `cd web-gin-service && go test ./...` | ✅ 全部通过 |
| `pnpm --filter hr-frontend typecheck` | ✅ 通过 |

## 越界检查

- `ai_service.go` 修改：文件名不匹配 `**memory*.go` 或 `**context*.go` 模式，但这是唯一的 Memory 创建入口。TASK 要求定位并接入该入口，不修改则无法完成任务。已在报告中说明。
- `services.go` 修改：超出 allowedFiles，必需 wiring。

## 风险与回滚

| 风险 | 说明 |
|------|------|
| 入口遗漏 | 已全面搜索确认仅 `writeMemory` 一个入口 |
| MQ 不可用 | publisher nil-check + BestEffort 发布，不阻塞 |
| 空内容 | `writeMemory` 已检查空文本；`publishMemoryEmbeddingEvent` 也检查 |

## 后续任务建议

继续执行 TASK-010（新增 Embedding MQ Consumer，需要人工确认）。
