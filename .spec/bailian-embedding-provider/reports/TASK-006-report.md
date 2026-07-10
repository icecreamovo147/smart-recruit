# TASK-006 Completion Report

## 任务目标

将 `services.go` 中硬编码的 `UnavailableEmbeddingProvider{}` 替换为 Factory 构建结果，使服务启动时根据配置/DB 构造真实 provider 或降级 provider。

## 修改文件列表

| 文件 | 操作 |
|------|------|
| `logic-grpc-service/service/services.go` | 修改 |

## 每个文件修改说明

### `logic-grpc-service/service/services.go`

- **新增 import**：`"context"`（用于 `context.Background()` 传递给 Factory.Build）
- **替换初始化逻辑**（原第 112 行）：
  - 原：`embeddingSvc := NewEmbeddingService(..., UnavailableEmbeddingProvider{})`
  - 新：创建 `embeddingModelRepo` 和 `embeddingProviderRepo`，尝试加载 `ENCRYPTION_KEY`
    - 加载成功 → 构造 `EmbeddingProviderFactory` 并调用 `Build` 获取真实或降级 provider
    - 加载失败 → 记录 warn，保持 `UnavailableEmbeddingProvider{}`
  - 最终调用 `NewEmbeddingService(..., embeddingProvider)`

核心代码段（`services.go:114-125`）：

```go
embeddingModelRepo := repository.NewEmbeddingModelRepo(db)
embeddingProviderRepo := repository.NewEmbeddingProviderRepo(db)

var embeddingProvider EmbeddingProvider = UnavailableEmbeddingProvider{}
encKey, encErr := crypto.LoadEncryptionKey()
if encErr == nil {
    factory := NewEmbeddingProviderFactory(cfg, embeddingModelRepo, embeddingProviderRepo)
    embeddingProvider = factory.Build(context.Background(), encKey)
} else {
    logger.L().Warn("[embedding] encryption key not set, using unavailable provider", zap.Error(encErr))
}
embeddingSvc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), embeddingProvider).WithRuntimePolicy(runtimePolicy)
```

## 是否超出 task-scope.json

**否**。仅修改了 `allowedFiles` 中的 `logic-grpc-service/service/services.go`。
未触碰任何 `forbiddenFiles`（proto、pb、`ai_embedding_repo.go`、web-gin、hr-frontend 等）。

## SPEC 对照结果

| SPEC 条目 | 状态 | 说明 |
|-----------|------|------|
| FR-004 服务初始化接入真实 Provider | ✅ | 启动时加载默认 model，构造真实 provider |
| 边界情况：配置缺失 → 降级 | ✅ | ENCRYPTION_KEY 未设置或无默认模型时降级 unavailable |
| 异常场景：Provider 未配置 → 不阻塞 | ✅ | 降级后服务正常启动 |
| 安全要求：API Key 不入日志 | ✅ | 仅记录是否加载 key，不输出 key 明文 |

## SDD 对照结果

| SDD 条目 | 状态 | 说明 |
|----------|------|------|
| 8 初始化流程：读取 config → 查默认模型 → 查 provider → 解密 → 构造 → 注入 | ✅ | 完全对齐 |
| 8 回滚：设置 embedding.enabled=false 或删除默认 model → 自动回退 | ✅ | factory.Build 内部处理 |
| 14 TASK-006 目标 | ✅ | 无配置时仍可启动，有默认模型时注入真实 provider |
| 兼容性：EmbeddingService 调用方式不变 | ✅ | 仅构造方式变化，对外接口不变 |

## Acceptance 对照结果

| 检查项 | 状态 | 说明 |
|--------|------|------|
| `go test ./...` 通过 | ✅ | `ok logic-grpc-service/service` |
| 无配置时服务仍可启动 | ✅ | encKey 加载失败 → warn + UnavailableEmbeddingProvider |
| 有默认模型时注入真实 provider | ✅ | factory.Build 返回 Bailian provider |
| DebugSemanticRetrieval 可读取 provider 状态 | ✅ | 注入方式不变，provider 状态透传 |
| 不越界修改 | ✅ | 仅 `services.go` |
| 日志不泄漏 API Key | ✅ | 仅记录 key 加载状态 |

## 检查命令与结果

```bash
# go test
$ cd logic-grpc-service && go test ./...
ok  logic-grpc-service/service  (cached)

# check-task-scope.sh
$ bash .spec/bailian-embedding-provider/scripts/check-task-scope.sh TASK-006
→ 修改文件: services.go (allowed)
→ 未触碰 forbiddenFiles

# agent-check.sh
$ bash .spec/bailian-embedding-provider/scripts/agent-check.sh
OK: hr-frontend typecheck
OK: hr-frontend test
OK: hr-frontend build
OK: logic-grpc-service go test ./...
OK: web-gin-service go test ./...
```

## 未完成内容

无。TASK-006 所有实现要点和验收标准均已满足。

## 风险点

- 公共 `Services` 初始化是全局入口，修改影响所有 gRPC 服务。本次修改仅替换了 embedding provider 的构造方式，未改动其他 service 初始化顺序，风险可控
- 回滚方式：恢复 `services.go` 中 `embeddingSvc` 为 `NewEmbeddingService(..., UnavailableEmbeddingProvider{})` 原写法即可

## 是否建议进入下一个 TASK

**是**。建议进入 TASK-007（需人工确认），新增 Embedding 配置与测试连接接口。
