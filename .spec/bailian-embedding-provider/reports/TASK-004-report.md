# TASK-004 完成报告

## 任务目标

实现阿里云百炼 MaaS 原生 HTTP text-embedding-v4 Provider。

## 修改文件列表

| 文件 | 操作 |
|------|------|
| `logic-grpc-service/service/embedding_provider_bailian.go` | 新增 |
| `logic-grpc-service/service/embedding_provider_bailian_test.go` | 新增 |

## 每个文件修改说明

### `embedding_provider_bailian.go`

实现 `EmbeddingProvider` 接口的 `BailianTextEmbeddingProvider`：

- **请求格式**：`POST {endpoint}`，`Content-Type: application/json`，`Authorization: Bearer {api_key}`，Body `{"model":"text-embedding-v4","input":{"texts":["文本"]}}`
- **响应解析**：兼容 `output.embeddings[].embedding`（百炼原生）和 `data[].embedding`（兼容格式）
- **重试策略**：401/403 不重试，429/5xx/timeout 最多重试 2 次（`DefaultBailianMaxRetries`），间隔线性退避
- **可选项**：`WithBailianTimeout`、`WithBailianMaxRetries`、`WithBailianModel`
- **安全**：日志中不输出 API Key；错误信息截断至 200 字符；空向量/维度异常返回 `ErrEmbeddingVectorInvalid`
- **模型**：默认 `text-embedding-v4`，可通过 `WithBailianModel` 覆盖；响应中 `model` 字段优先从 response body 提取

### `embedding_provider_bailian_test.go`

13 个测试用例，覆盖所有关键路径：

| 测试 | 覆盖场景 |
|------|----------|
| TestBailianProvider_Success | 请求体/Header/向量结果正确性 |
| TestBailianProvider_DataResponse | 兼容 `data` 响应格式 + model 提取 |
| TestBailianProvider_401NoRetry | 401 不重试 |
| TestBailianProvider_403NoRetry | 403 不重试 |
| TestBailianProvider_429Retry | 429 重试后成功 |
| TestBailianProvider_5xxRetry | 500 重试耗尽返回错误 |
| TestBailianProvider_TimeoutRetry | 连接超时重试耗尽 |
| TestBailianProvider_EmptyVector | 空向量返回错误 |
| TestBailianProvider_NullVector | null 向量返回错误 |
| TestBailianProvider_InvalidJSON | 响应非 JSON 返回错误 |
| TestBailianProvider_ModelOverride | 自定义模型名传入 |
| TestBailianProvider_RequestCancelled | 已取消 Context 返回错误 |
| TestIsRetryableStatus | 11 种 HTTP 状态码的可重试判定 |

## 是否超出 task-scope.json

**否**。新增文件均匹配 allowedFiles：
- `service/embedding_provider_bailian.go` ✓
- `service/embedding_provider_bailian_test.go` ✓

未修改 `embedding_service.go`（允许但不需要改动）。未触碰任何 forbiddenFiles。

## SPEC 对照结果

| 需求 | 状态 |
|------|------|
| FR-003: 测试连接（HTTP 调用、vector 维度、耗时） | ✓ Provider 可被 TestEmbeddingModel 复用 |
| §6 接口设计：百炼 HTTP API | ✓ endpoint/header/request body 与 SDD 完全一致 |
| §9 异常：429/5xx/timeout 可重试 | ✓ |
| §9 异常：401/403 不重试 | ✓ |
| §9 异常：空向量/维度异常返回错误 | ✓ |
| §10 日志：不输出 API Key 或完整文本 | ✓ |
| §11 安全：错误返回脱敏 | ✓ 截断至 200 字符 |

## SDD 对照结果

| 设计项 | 状态 |
|--------|------|
| §6.1 请求体 `{model, input:{texts:[...]}}` | ✓ |
| §6.1 Header `Authorization: Bearer` + `Content-Type: application/json` | ✓ |
| §6.1 兼容 `output.embeddings` 和 `data` | ✓ |
| §6.1 超时默认 30s | ✓ `DefaultBailianTimeout` |
| §6.1 重试 429/5xx/timeout 最多 2 次 | ✓ `DefaultBailianMaxRetries=2` |
| §14 TASK-004: 实现 Bailian Provider | ✓ |

## Acceptance 对照结果

| 检查项 | 结果 |
|--------|------|
| fake HTTP server 单测覆盖成功请求体和 header | ✓ TestBailianProvider_Success |
| 401/403 不重试 | ✓ 各 1 个测试，callCount=1 |
| 429/5xx/timeout 按策略重试 | ✓ 各 1 个测试，验证 callCount |
| 空 vector、维度异常返回错误 | ✓ EmptyVector + NullVector |
| go test 通过 | ✓ |

## 检查命令与结果

| 命令 | 结果 |
|------|------|
| `git diff --name-only` | ✓ 无已跟踪文件改动（均为新 untracked 文件） |
| `bash .spec/bailian-embedding-provider/scripts/check-task-scope.sh TASK-004` | ✓ 未发现越界 |
| `bash .spec/bailian-embedding-provider/scripts/agent-check.sh` | ✓ typecheck/test/build 均通过 |
| `cd logic-grpc-service && go test ./...` | ✓ 全部通过 |

## 未完成内容

无。所有目标均已达成。

## 风险点

| 风险 | 状态 |
|------|------|
| 百炼真实响应结构与假设不一致 | ⚠️ 模拟了 `output.embeddings` 和 `data` 两种格式；真实响应如与 curl 示例不符需调整 `parseBailianResponse` |
| 错误日志泄漏敏感文本 | ✓ 响应 body 截断至 200 字符 |

## 回滚方式

```bash
rm logic-grpc-service/service/embedding_provider_bailian.go
rm logic-grpc-service/service/embedding_provider_bailian_test.go
```

## 是否建议进入下一个 TASK

**是**。TASK-004 已完成，可进入 **TASK-005**（实现 Embedding Provider Factory）。

## Review 结果

**通过**（第二次 Fix 后）

| 检查项 | 结果 |
|--------|------|
| gofmt | ✓ |
| `isRetryableError` 残留函数定义 | ✓ 已删除 |
| `go test ./service/...` | ✓ 全部通过 |
| agent-check | ✓ 全部通过 |
