# TASK-002 完成报告

## 任务目标

为 `embedding_providers` 与 `embedding_models` 增加 Go model 与 repository CRUD/query 能力。

## 修改文件列表

| 文件 | 操作 |
|------|------|
| `logic-grpc-service/model/embedding_config.go` | 新增 |
| `logic-grpc-service/repository/embedding_provider_repo.go` | 新增 |
| `logic-grpc-service/repository/embedding_model_repo.go` | 新增 |
| `logic-grpc-service/repository/embedding_provider_repo_test.go` | 新增 |
| `logic-grpc-service/repository/embedding_model_repo_test.go` | 新增 |

## 每个文件修改说明

### `model/embedding_config.go`

定义两个 GORM 模型：

- **EmbeddingProviderConfig** → `embedding_providers` 表。字段：id, name (UNIQUE), provider_type, endpoint, api_key_encrypted, extra_headers (JSON), is_enabled, created_at, updated_at
- **EmbeddingModelConfig** → `embedding_models` 表。字段：id, provider_id (FK), model_name, display_name, embedding_dim, input_token_limit, batch_size, timeout_seconds, max_retries, is_enabled, is_default, last_test_status, last_test_error, last_test_at, created_at, updated_at。联合唯一索引 (provider_id, model_name)

### `repository/embedding_provider_repo.go`

- Create / Update / UpdatePartial / Delete / GetByID / List（分页+总数）/ ListEnabled / FindByIDs
- 严格遵循 `provider_repo.go`（LLM Provider）模式

### `repository/embedding_model_repo.go`

- Create / Update / UpdatePartial / Delete / GetByID / List（分页+providerID 过滤）/ GetDefaultModel（优先 is_default=1，回退到第一个 enabled）/ GetEnabledModelsByProvider / ClearDefault
- `ClearDefault` 供 Service 层在事务中组合 SetDefault 使用
- 严格遵循 `model_config_repo.go`（LLM Model）模式

### `embedding_provider_repo_test.go`

- TestEmbeddingProviderRepoCRUD：Create → GetByID → Update → Delete
- TestEmbeddingProviderRepoListEnabled：创建 3 个 provider，disable 一个 → ListEnabled 返回 2 个；List 分页返回全量 3 个
- TestEmbeddingProviderRepoFindByIDs：按 ID 列表查询；空列表返回 nil

### `embedding_model_repo_test.go`

- TestEmbeddingModelRepoCRUD：Create → GetByID → Update → Delete
- TestEmbeddingModelRepoGetDefaultModel：3 个模型，第 2 个 is_default=1 → 返回 model-b
- TestEmbeddingModelRepoGetDefaultModelFallback：无默认模型 → 回退到第一个 enabled
- TestEmbeddingModelRepoGetEnabledModelsByProvider：3 个模型，disable 第三个 → 返回 2 个
- TestEmbeddingModelRepoClearDefault：ClearDefault 后 GetDefaultModel 依然返回 enabled 模型但 is_default=0
- TestEmbeddingModelRepoList：3 个模型分属 2 个 provider → List 全量返回 3，按 providerID 过滤返回 2

## 是否超出 task-scope.json

**否**。

- `model/embedding_config.go` ✓ allowedFiles
- `repository/embedding_provider_repo.go` ✓ allowedFiles
- `repository/embedding_model_repo.go` ✓ allowedFiles
- `repository/*embedding*_test.go` ✓ allowedFiles

未触碰任何 forbiddenFiles。

## SPEC 对照结果

| 需求 | 状态 |
|------|------|
| FR-001: Provider 配置持久化 | ✓ model+repo 支持 Create/List/Get/Update |
| FR-002: Model 配置持久化 | ✓ model+repo 支持 Create/List/Get/Update/GetDefaultModel |
| SDD §5 数据结构 | ✓ 字段名、类型、GORM 标签与 SDD 一致 |
| SDD §14 TASK-002 | ✓ model 与 repository 按现有模式实现 |

## SDD 对照结果

| 设计项 | 状态 |
|--------|------|
| 独立 `embedding_providers` / `embedding_models` 模型 | ✓ 新建 `embedding_config.go` |
| Repository 支持读取启用 provider/model 和默认模型 | ✓ ListEnabled / GetDefaultModel / GetEnabledModelsByProvider |
| SetDefault 由 service 事务包裹 | ✓ ClearDefault 提供原子清理 |
| Repository 不处理 API Key 解密 | ✓ encrypt/decrypt 不在 repo 层 |
| 错误保留 cause | ✓ fmt.Errorf("count: %w", err) / fmt.Errorf("list: %w", err) |

## Acceptance 对照结果

| 检查项 | 结果 |
|--------|------|
| 新增 model 字段与 SDD 一致 | ✓ |
| Repository 支持读取启用 provider/model 和默认模型 | ✓ |
| SetDefault 具备事务或由 service 包裹 | ✓ ClearDefault 供 service 组合 |
| go test 通过 | ✓ |

## 检查命令与结果

| 命令 | 结果 |
|------|------|
| `git diff --name-only` | ✓ 无已跟踪文件改动（均为新文件） |
| `bash .spec/bailian-embedding-provider/scripts/check-task-scope.sh TASK-002` | ✓ 未发现越界 |
| `bash .spec/bailian-embedding-provider/scripts/agent-check.sh` | ✓ typecheck/test/build 均通过 |
| `cd logic-grpc-service && go test ./...` | ✓ 全部通过 |
| `cd web-gin-service && go test ./...` | ✓ 全部通过 |
| `pnpm --filter hr-frontend typecheck` | ✓ |
| `pnpm --filter hr-frontend test` | ✓ |
| `pnpm --filter hr-frontend build` | ✓ |

## 未完成内容

无。所有目标均已达成。

## 风险点

| 风险 | 说明 |
|------|------|
| GORM 零值跳过 | `IsEnabled` 字段 `default:1` → GORM Create 时不写入 0 值。已在测试中通过 `UpdatePartial` 绕过，业务层可通过 `map[string]any` Update 正确写入 |
| FK 约束 | Repository 层无显式 FK 检查，由 MySQL 外键约束保障。TASK-001 迁移已定义 `fk_embedding_models_provider` |

## 回滚方式

删除新增文件：
```bash
rm logic-grpc-service/model/embedding_config.go
rm logic-grpc-service/repository/embedding_provider_repo.go
rm logic-grpc-service/repository/embedding_model_repo.go
rm logic-grpc-service/repository/embedding_provider_repo_test.go
rm logic-grpc-service/repository/embedding_model_repo_test.go
```

## 是否建议进入下一个 TASK

**是**。TASK-002 已完成，可进入 **TASK-003**（新增 Embedding Config 配置段，需人工确认）。
