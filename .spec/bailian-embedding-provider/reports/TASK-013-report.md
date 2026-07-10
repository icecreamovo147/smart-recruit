# TASK-013 Report - 补充跨层测试与回归检查

## 完成状态

✅ **完成**

## 修改文件列表

| 文件 | 操作 | 说明 |
|------|------|------|
| `logic-grpc-service/service/embedding_text_builder_test.go` | **新建** | TestBuildAgentSkillEmbeddingText（5 cases）、TestBuildMemoryEmbeddingText（3 cases）、TestStripMarkdown、TestBuildAgentSkillEmbeddingTextMaxLen |
| `logic-grpc-service/service/embedding_config_service_test.go` | **新建** | TestEmbeddingProviderToInfoMasking、TestEmbeddingModelConfigDefaults、TestBackfillInputDefaultValues、TestEmbeddingScopeResolution、TestEmbeddingUpsertEventDefaults |
| `logic-grpc-service/service/embedding_text_builder.go` | 修改 | 修复 stripMarkdown 对 **bold** 和 _italic_ 的处理 |

## 自测命令结果

| 命令 | 结果 |
|------|------|
| `cd logic-grpc-service && go test ./...` | ✅ 全部通过 |
| `cd web-gin-service && go test ./...` | ✅ 全部通过 |
| `pnpm --filter hr-frontend typecheck` | ✅ 通过 |
| `pnpm --filter hr-frontend test` | ✅ 通过 (9 tests) |
| `pnpm --filter hr-frontend build` | ✅ 构建成功 |

## 测试覆盖情况

| 模块 | 已有测试 | 新增测试 | 状态 |
|------|---------|---------|------|
| EmbeddingTextBuilder | ❌ | ✅ 10 个用例 | ✅ |
| ConfigService helpers | ❌ | ✅ 5 个用例 | ✅ |
| ProviderBailian | ✅ | - | ✅ |
| ProviderFactory | ✅ | - | ✅ |
| EmbeddingService | ✅ | - | ✅ |
| MQ Consumer/Pub | ❌ | 注入测试 | ⚠️ 需有 MQ 基础设施 |

## 越界检查

- `embedding_text_builder.go` 修复：在 allowedFiles 范围内（属于工具函数微调）
- 仅修改测试文件和辅助函数，未触碰业务代码

## 风险与回滚

无新增风险。stripMarkdown 修复改进了 `*`、`_`、`` ` `` 的清理逻辑。
