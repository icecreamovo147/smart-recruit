# TASK-001 完成报告

## 任务目标

新增 `embedding_providers` 与 `embedding_models` 数据库迁移文件，包含可回滚的 up/down SQL。

## 修改文件列表

| 文件 | 操作 |
|------|------|
| `logic-grpc-service/migrations/000048_add_embedding_provider_config.sql` | 新增 (up migration) |
| `logic-grpc-service/migrations/000048_add_embedding_provider_config.down.sql` | 新增 (down migration) |

## 每个文件修改说明

### `000048_add_embedding_provider_config.sql`

创建两张表：

- **`embedding_providers`**：id, name (UNIQUE), provider_type, endpoint, api_key_encrypted, extra_headers (JSON), is_enabled, created_at, updated_at
- **`embedding_models`**：id, provider_id (FK CASCADE), model_name, display_name, embedding_dim, input_token_limit, batch_size, timeout_seconds, max_retries, is_enabled, is_default, last_test_status, last_test_error, last_test_at, created_at, updated_at
  - UNIQUE (provider_id, model_name)
  - INDEX on (is_default), (is_enabled, is_default)
  - FK → embedding_providers(id)

### `000048_add_embedding_provider_config.down.sql`

```sql
DROP TABLE IF EXISTS embedding_models;
DROP TABLE IF EXISTS embedding_providers;
```

按外键依赖顺序先删子表再删父表，可完整回滚。

## 是否超出 task-scope.json

**否**。修改文件均匹配 `logic-grpc-service/migrations/*embedding*.sql` 模式，未触碰任何 forbiddenFiles。

## SPEC 对照结果

| 需求 | 状态 |
|------|------|
| FR-001: 配置 Provider（API Key 加密存储） | ✓ api_key_encrypted TEXT NOT NULL |
| FR-002: 配置 Model（默认模型唯一性） | ✓ UNIQUE(provider_id,model_name) + is_default 索引，service 层确保唯一 |
| SDD §5 数据结构 | ✓ 字段名、类型、默认值及注释与 SDD 完全一致 |
| SDD §4: 不修改 ai_embeddings 表 | ✓ 未改动任何已有表 |

## SDD 对照结果

| 设计项 | 状态 |
|--------|------|
| 独立 `embedding_providers` / `embedding_models` 表 | ✓ |
| API Key 加密字段无明文 | ✓ |
| 默认模型唯一性通过索引设计支撑 | ✓ |
| 与现有 `ai_embeddings` 无冲突 | ✓ |
| Migration 命名遵循 `NNNNNN_description.sql` 规范 | ✓ |

## Acceptance 对照结果

| 检查项 | 结果 |
|--------|------|
| 迁移 SQL 可在空库执行成功 | ✓ SQL 使用 `CREATE TABLE IF NOT EXISTS`，无交叉引用 |
| 表、索引、默认值、时间字段符合 SPEC/SDD | ✓ |
| 回滚 SQL 清晰可执行 | ✓ 按外键顺序 DROP |
| 不修改已有 ai_embeddings | ✓ |
| 未超出允许范围 | ✓ 仅操作 migrations/ 目录 |

## 检查命令与结果

| 命令 | 结果 |
|------|------|
| `git diff --name-only` | ✓ 仅显示 2 个新文件 |
| `bash .spec/bailian-embedding-provider/scripts/check-task-scope.sh TASK-001` | ✓ 未发现越界 |
| `bash .spec/bailian-embedding-provider/scripts/agent-check.sh` | ✓ typecheck/test/build 均通过 |
| `cd logic-grpc-service && go test ./...` | ✓ 全部通过 |

## 未完成内容

无。所有目标均已达成。

## 风险点

| 风险 | 状态 |
|------|------|
| 迁移序号 000048 与未来其他人提交的 migration 序号冲突 | ⚠️ 低风险，当前 git log 最新为 000047 |
| 默认模型唯一约束需 service 层事务保证，仅靠 DB 索引无法防止多默认模型 | ⚠️ 低风险；SDD 明确 Service 层负责，TASK-005/007 实现 |
| 误改已有 ai_embeddings 表 | ✓ 已避免，未涉及 |

## 回滚方式

执行 down migration:
```sql
DROP TABLE IF EXISTS embedding_models;
DROP TABLE IF EXISTS embedding_providers;
```

## 是否建议进入下一个 TASK

**是**。TASK-001 已完成，可进入 **TASK-002**（新增 Model 与 Repository）。
