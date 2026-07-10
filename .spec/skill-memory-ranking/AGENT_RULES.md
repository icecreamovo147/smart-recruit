# Skill / Memory 召回排序重构 Agent Rules

## 1. 基本执行规则

1. 每次只能执行一个 TASK，不允许在同一轮合并多个 TASK。
2. 执行 TASK 前必须读取：
   - `.spec/skill-memory-ranking/skill-memory-ranking-SPEC.md`
   - `.spec/skill-memory-ranking/skill-memory-ranking-SDD.md`
   - `.spec/skill-memory-ranking/TASKS.md`
   - `.spec/skill-memory-ranking/AGENT_RULES.md`
   - `.spec/skill-memory-ranking/task-scope.json`
   - `.spec/skill-memory-ranking/acceptance/<TASK-ID>.md`
3. 不允许修改任务范围之外的文件。
4. 不允许修改 `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`、全局配置文件，除非当前 TASK 明确允许。
5. 不允许大规模格式化无关文件。
6. 不允许为了通过类型检查随意使用 `any`。
7. 不允许吞掉异常；必须保留错误语义、日志和可观测性。
8. 不允许删除已有测试或弱化已有断言。
9. 不允许改变对外 API 行为；新增字段必须保持向后兼容。
10. 如果必须修改公共模块（proto / pb / config / 共享类型），必须先停止并说明原因，等待人工确认。
11. 每个 TASK 完成后必须输出完成报告。

## 2. 安全与隐私规则

- 不得提交真实 API Key、Bearer Token、Cookie 或本地凭证。
- 日志和测试 fixture 不得包含完整简历、完整 Memory 内容、完整 SKILL.md 或真实敏感文本。
- 日志不得输出完整 query 原文（特别在降级路径中）。
- 日志不得输出 embedding vector 原值。
- 不得使用 `v-html` 展示任何 debug 字段。

## 3. 代码质量规则

- Go 代码必须 `gofmt`。
- 涉及 `hr-frontend` 时必须通过 `pnpm --filter hr-frontend typecheck`。
- 涉及 `logic-grpc-service` 时必须 `cd logic-grpc-service && go test ./...`。
- 涉及 `web-gin-service` 时必须 `cd web-gin-service && go test ./...`。
- 打分 / 排序 / 置信度逻辑必须补充针对失败路径与边界场景的测试（NaN / Inf / 空池 / 全 0 分）。
- proto 字段编号必须只追加，不重用。
- 现有函数签名（`selectAgentSkills` / `selectAgentSkillsWithSemantic` / `rankMemories` / `retrieveMemories` / `Build` / `DebugSemanticRetrieval`）保持不变。

## 4. 兼容性规则

- proto 仅追加新字段，不删除 / 改编号 / 改类型；旧字段语义不变。
- 前端 TypeScript 类型仅追加 optional 字段。
- 旧 `score` 字段值 = `final_rank_score`，`selectedAgentSkill.Score int` 旧字段值 = `int(round(final_rank_score * 100))`。
- 旧 `tokenizeAgentSkillText` / `addCJKAgentSkillTokens` / `scoreAgentSkillMatch` / `memoryBaseRecallScore` / `keywordMemoryScore` / `semanticMemoryScores` 不被删除，可作为内部 helper 复用。
- 现有 `TestSelectAgentSkillsManualPriorityAndAutoMatch` / `TestSelectAgentSkillsUsesPriorityAndRequiredCapabilities` / `TestSelectAgentSkillsUsesSemanticScoresOnlyToRerankRuleCandidates` / `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 保持通过。

## 5. 权重 / 阈值常量规则

所有权重与阈值集中在 `logic-grpc-service/service/skill_memory_ranking.go` 顶部：

```go
const (
    rankWeightVector     = 0.6
    rankWeightLexical    = 0.3
    rankWeightMetadata   = 0.1
    rankBusinessBoostMax = 1.5
    rankPriorityNorm     = 50.0
    rankBoostAlpha       = 0.2
    rankBoostBeta        = 0.2
    rankBoostGamma       = 0.1
    rankRelevanceGate    = 0.15
    rankGapHigh          = 0.10
    rankGapMedium        = 0.03
)
```

任何对这些常量的调整必须：

1. 同步更新 SPEC §5 FR-002 与 SDD §3.3 / §3.4。
2. 通过 `go test ./...` 全部测试。
3. 在 TASK 完成报告中说明。

## 6. 日志规则

- 排序关键路径使用 `logger.GetRequestLogger(ctx).Debug` 输出 breakdown。
- 降级路径使用 `logger.GetRequestLogger(ctx).Info` 输出 `mode=fallback reason=...`，**不输出 query 原文**。
- 旧日志字段（`[Agent Skill] selected for ADK instruction` 等）保持。
- 不得在 info / warn 级别输出 score breakdown；仅 debug 级别。
- 不得输出完整 Memory content、Skill markdown、API Key、token。

## 7. 测试规则

- 不得使用真实 embedding provider；使用 `fakeEmbeddingProvider` 或 nil provider 即可。
- 不得使用 `any` 绕过类型检查；如必要可使用 `interface{}`。
- 不得删除已有测试；不得弱化已有断言。
- 必要时更新对 `Score` 整数 / 加权和值的强断言到 `FinalRankScore` / `final_rank_score`，并在报告中说明。
- 表驱动测试必须覆盖：纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值 / 降级。

## 8. 完成报告格式

每个 TASK 完成后必须输出：

- TASK 编号和名称。
- 实际修改文件。
- 未修改但检查过的关键文件。
- 已执行的自测命令和结果。
- 未执行命令及原因。
- 是否触碰公共模块。
- 是否存在越界修改。
- 风险、回滚方式和后续任务建议。

## 9. Hard Stop 条件

以下情形必须停止并请求用户确认：

- 需要修改 `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`。
- 需要修改 `logic-grpc-service/proto/recruitment.proto`（TASK-005 范围内已通过 `requiresHumanConfirmation` 标记，但具体字段命名 / 类型 / 编号仍需先与用户确认）。
- 需要修改 `recruitment/pb/recruitment.pb.go` 现有字段。
- 需要修改 `model/**` / `repository/**`。
- 需要修改 `config/**` 公共段。
- 需要改变 `selectAgentSkills` / `selectAgentSkillsWithSemantic` / `rankMemories` / `retrieveMemories` / `Build` / `DebugSemanticRetrieval` 函数签名。
- 需要改变 proto 现有字段编号、类型或语义。
- 需要在 debug 视图引入新功能（如 A/B 按钮、阈值调参 UI）。
- 需要新增第三方依赖。
- TASK 范围不足以完成当前需求。
- `requiresHumanConfirmation: true` 的 TASK 在开始编码前。
