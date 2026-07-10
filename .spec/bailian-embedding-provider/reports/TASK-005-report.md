# TASK-005 Completion Report

## 任务目标

根据默认 embedding model 和 provider 配置构造真实 Provider 或降级 Provider。

## 修改文件列表

| 文件 | 操作 |
|------|------|
| `logic-grpc-service/service/embedding_provider_factory.go` | 新增 |
| `logic-grpc-service/service/embedding_provider_factory_test.go` | 新增 |

## 每个文件修改说明

### `logic-grpc-service/service/embedding_provider_factory.go` (新增)

实现 `EmbeddingProviderFactory` 结构体和构建方法：

- **`EmbeddingProviderFactory`**：持有 config、`EmbeddingModelRepo`、`EmbeddingProviderRepo`
- **`Build(ctx, encKey)`**：按以下顺序检查并构建：
  1. 检查 `config.Embedding.Enabled`，若为 `false` 返回 `UnavailableEmbeddingProvider`
  2. 通过 `modelRepo.GetDefaultModel()` 获取默认模型，失败/不存在返回 unavailable
  3. 检查模型 `IsEnabled`，disabled 返回 unavailable
  4. 通过 `providerRepo.GetByID()` 获取关联 provider，失败返回 unavailable
  5. 检查 provider `IsEnabled`，disabled 返回 unavailable
  6. 按 `ProviderType` switch 分发：
     - `"bailian"` → `buildBailian()` 构建 `BailianTextEmbeddingProvider`
     - default → 记录 warn 返回 unavailable
- **`buildBailian()`**：解密 API Key（失败降级）、构造超时/重试选项、调用 `NewBailianTextEmbeddingProvider`
- 所有失败路径均通过 `logger.L().Warn()` 记录脱敏日志，不 panic

### `logic-grpc-service/service/embedding_provider_factory_test.go` (新增)

覆盖以下场景：

| 测试 | 验证点 |
|------|--------|
| `ConfigDisabled` | config embedding.enabled=false → unavailable |
| `NoDefaultModel` | 空 DB 无默认模型 → unavailable |
| `ModelDisabled` | 默认模型 is_enabled=0 → unavailable |
| `ProviderDisabled` | 关联 provider is_enabled=0 → unavailable |
| `UnsupportedProviderType` | provider_type="openai" → unavailable |
| `DecryptFails` | 用错误 key 解密 → unavailable，不 panic |
| `DecryptInvalidData` | 无效加密数据 → unavailable（非 panic） |
| `SuccessBailian` | 完整正确配置 → `*BailianTextEmbeddingProvider` |
| `ZeroEncryptionKey` | 零值 encryption key → unavailable |

## 是否超出 task-scope.json

**否**。仅修改了 `allowedFiles` 中声明的两个文件：
- `logic-grpc-service/service/embedding_provider_factory.go`
- `logic-grpc-service/service/embedding_provider_factory_test.go`

未触碰任何 `forbiddenFiles`。

## SPEC 对照结果

| SPEC 条目 | 状态 | 说明 |
|-----------|------|------|
| FR-004 服务初始化接入真实 Provider | ✅ | Factory 可作为初始化入口，TASK-006 接入 services.go |
| 异常场景：Provider 未配置 → 降级 | ✅ | 缺失/disabled → UnavailableEmbeddingProvider |
| 异常场景：解密失败 → 降级 | ✅ | 解密失败记录 warn，返回 unavailable |
| 安全要求：API Key 不进入日志 | ✅ | 仅记录 provider_id，不输出 key |
| 非功能：Provider factory 可扩展 | ✅ | switch 分发，新增 case 即可扩展其他厂商 |

## SDD 对照结果

| SDD 条目 | 状态 | 说明 |
|----------|------|------|
| 4.4 Provider Factory 模块 | ✅ | 实现读取默认模型、解密 key、构造 provider |
| 5.1 Go Config Embedding 段 | ✅ | Factory 读取 `cfg.Embedding.Enabled` |
| 11 安全：API Key 加密存储 | ✅ | 解密仅用于构造 provider，不入日志 |
| 14 TASK-005 目标 | ✅ | 配置缺失→降级；配置正确→返回 Bailian provider |

## Acceptance 对照结果

| 检查项 | 状态 | 说明 |
|--------|------|------|
| `go test ./...` 通过 | ✅ | 4.357s，全部通过 |
| 配置缺失返回 unavailable | ✅ | TestNoDefaultModel, TestConfigDisabled 等 |
| 配置正确返回 Bailian provider | ✅ | TestSuccessBailian |
| 解密失败不 panic，不泄漏密钥 | ✅ | TestDecryptFails, TestDecryptInvalidData |
| 日志不泄漏 API Key | ✅ | 只记录 provider_id，不记录 key 明文 |
| 不阻塞主业务流程 | ✅ | Provider 不可用时返回 UnavailableEmbeddingProvider |

## 检查命令与结果

```bash
# go test
$ cd logic-grpc-service && go test ./...
ok  logic-grpc-service/service  4.357s

# check-task-scope.sh
$ bash .spec/bailian-embedding-provider/scripts/check-task-scope.sh TASK-005
→ 新增文件: embedding_provider_factory.go, embedding_provider_factory_test.go
→ 未触碰 forbiddenFiles

# agent-check.sh
$ bash .spec/bailian-embedding-provider/scripts/agent-check.sh
SKIP: hr-frontend lint (未配置脚本)
OK: hr-frontend typecheck
OK: hr-frontend test
OK: hr-frontend build
OK: logic-grpc-service go test ./...
OK: web-gin-service go test ./...
agent-check completed
```

## fix-check-failures 修复记录

### 失败原因

self-review 发现 3 个问题：

| 问题 | 位置 | 严重级 | 描述 |
|------|------|--------|------|
| 1 | `embedding_provider_factory_test.go:237` | Cosmetic | 测试名 `DecryptPanics` 有误导性（实际测不 panic） |
| 2 | `embedding_provider_factory_test.go:284` | Cosmetic | `_ = bailianProvider` 多余 |
| 3 | `embedding_provider_factory.go:88` | Low | 无负值 `MaxRetries` 防护，理论上会导致 provider 0 次请求 |

### 修复内容

| 问题 | 修改文件 | 修复方式 |
|------|----------|----------|
| 1 | `embedding_provider_factory_test.go` | 重命名 `DecryptPanics` → `DecryptInvalidData` |
| 2 | `embedding_provider_factory_test.go` | 将 `bailianProvider, ok :=` + `_ = bailianProvider` 改为 `if _, ok := ` 单行 |
| 3 | `embedding_provider_factory.go` | 在 `buildBailian` 中 `maxRetries` 后追加 `if maxRetries < 0 { maxRetries = 0 }` |

### 是否在 TASK-005 范围内

**是**。仅修改 `allowedFiles` 中的 `embedding_provider_factory.go` 和 `embedding_provider_factory_test.go`。

### 重新运行的命令与结果

```bash
$ cd logic-grpc-service && go test ./...
ok  logic-grpc-service/service  5.255s

$ bash .spec/bailian-embedding-provider/scripts/agent-check.sh
OK: 全部通过
```

### 是否仍有残留问题

**否**。三个问题均已修复，`gofmt` 合规。

## 未完成内容

无。TASK-005 所有实现要点和验收标准均已满足。

## 风险点

- 密钥解密依赖 `crypto.Decrypt`，需要 `ENCRYPTION_KEY` 环境变量正确配置；未配置时由 `newLlmConfigServiceWithFallback` 模式处理（TASK-006 需人工确认接入方式）
- factory 的 switch 分发当前只支持 `"bailian"`，新增 provider type 需要修改 factory 代码——当前设计符合 SPEC/SDD 的"后续扩展"保留

## 是否建议进入下一个 TASK

**是**。建议进入 TASK-006（需人工确认），将 factory 接入 `services.go` 初始化流程。
