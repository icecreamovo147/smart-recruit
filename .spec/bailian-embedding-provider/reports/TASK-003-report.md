# TASK-003 完成报告

## 任务目标

在 logic-grpc-service 配置中增加 Embedding 配置段和默认值。

## 修改文件列表

| 文件 | 操作 |
|------|------|
| `logic-grpc-service/config/config.go` | 修改 |
| `logic-grpc-service/config/config.example.yaml` | 修改 |
| `logic-grpc-service/config/config_test.go` | 修改 |

## 每个文件修改说明

### `config/config.go`

- 在 `Config` 结构体中新增 `Embedding` 嵌套结构体，包含 6 个字段（SDD §5）：
  - `Enabled *bool`（yaml: enabled）— 默认 `true`
  - `DefaultModelID int64`（yaml: default_model_id）— 默认 `0`
  - `FallbackToRuleRetrieval *bool`（yaml: fallback_to_rule_retrieval）— 默认 `true`
  - `RequestTimeout Duration`（yaml: request_timeout）— 默认 `30s`
  - `MaxConcurrency int`（yaml: max_concurrency）— 默认 `8`
  - `SlowRequestThreshold Duration`（yaml: slow_request_threshold）— 默认 `2s`
- 在 `Load()` 中添加默认值逻辑（与 AI/MCP 等已有段一致的 `defaultBool` + `<= 0` 检查）
- 在 `applyEnvOverrides()` 中添加 6 个环境变量覆盖：
  - `EMBEDDING_ENABLED` / `EMBEDDING_DEFAULT_MODEL_ID` / `EMBEDDING_FALLBACK_TO_RULE_RETRIEVAL` / `EMBEDDING_REQUEST_TIMEOUT` / `EMBEDDING_MAX_CONCURRENCY` / `EMBEDDING_SLOW_REQUEST_THRESHOLD`
- 新增 `setInt64` 辅助函数用于 `DefaultModelID` 的 env 覆盖

### `config/config.example.yaml`

- 在 `frontend_base_url` 前新增 `embedding:` 配置段，包含所有 6 个字段及其默认值和环境变量注释

### `config/config_test.go`

- `TestEmbeddingConfigDefaultValues`：验证默认值全部正确
- `TestEmbeddingConfigEnvOverrides`：验证环境变量覆盖全部生效

## 是否超出 task-scope.json

**否**。修改的文件均匹配 allowedFiles：
- `config/config.go` ✓
- `config/*.go`（含 config_test.go）✓
- `config/**/*.yaml`（config.example.yaml）✓

未触碰任何 forbiddenFiles（未改 service/、repository/、model/、web-gin-service/ 等）。

## SPEC 对照结果

| 需求 | 状态 |
|------|------|
| FR-004 初始化接入：服务启动时读取配置 | ✓ config struct + defaults 已就绪，供 TASK-006 在 services.go 中读取 |
| SPEC §8 非功能：超时 30s、并发 8 | ✓ 默认值与 SPEC 一致 |

## SDD 对照结果

| 设计项 | 状态 |
|--------|------|
| §5 Go Config 结构体 | ✓ 字段名、类型、yaml tag 与 SDD 完全一致 |
| §5 默认值（enabled=true, fallback=true, timeout=30s, max_concurrency=8, slow=2s） | ✓ |
| 无配置时服务可启动并走降级 | ✓ 所有字段有安全默认值，不引入必填项 |
| 不引入必填环境变量 | ✓ 无配置时 Load() 正常返回 |

## Acceptance 对照结果

| 检查项 | 结果 |
|--------|------|
| 无 embedding 配置时配置解析成功 | ✓ 所有字段有零值默认处理 |
| 配置字段可被 services 初始化读取 | ✓ Config 结构体导出，TASK-006 可直接访问 cfg.Embedding |
| go test 通过 | ✓ |

## 检查命令与结果

| 命令 | 结果 |
|------|------|
| `git diff --name-only` | ✓ 仅 config/ 目录下 3 个文件 |
| `bash .spec/bailian-embedding-provider/scripts/check-task-scope.sh TASK-003` | ✓ 未发现越界 |
| `bash .spec/bailian-embedding-provider/scripts/agent-check.sh` | ✓ typecheck/test/build 均通过 |
| `cd logic-grpc-service && go test ./...` | ✓ 全部通过 |
| `cd web-gin-service && go test ./...` | ✓ 全部通过 |

## 未完成内容

无。所有目标均已达成。

## 风险点

| 风险 | 状态 |
|------|------|
| 公共 Config 结构影响服务启动 | ✓ 所有字段有安全默认值，无配置时正常启动 |
| `DefaultModelID=0` 含义 | ⚠️ 0 表示"未设置"，由 TASK-005/006 在 Factory 中处理降级 |
| `Embedding.Enabled` 默认 true | ⚠️ 如果 DB 中无默认模型，实际 provider 仍会降级为 Unavailable，不会直接启用真实调用 |

## 回滚方式

```bash
git checkout logic-grpc-service/config/config.go
git checkout logic-grpc-service/config/config.example.yaml
git checkout logic-grpc-service/config/config_test.go
```

## 是否建议进入下一个 TASK

**是**。TASK-003 已完成，可进入 **TASK-004**（实现 Bailian Text Embedding Provider）。
