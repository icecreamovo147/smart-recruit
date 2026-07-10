# Skill / Memory 召回排序 — 后续硬化 Agent Rules

## 1. 基本执行规则

1. 每次只能执行一个 TASK，不允许在同一轮合并多个 TASK。
2. 执行 TASK 前必须读取：
   - `.spec/skill-memory-ranking-followups/skill-memory-ranking-followups-SPEC.md`
   - `.spec/skill-memory-ranking-followups/skill-memory-ranking-followups-SDD.md`
   - `.spec/skill-memory-ranking-followups/TASKS.md`
   - `.spec/skill-memory-ranking-followups/AGENT_RULES.md`
   - `.spec/skill-memory-ranking-followups/task-scope.json`
   - `.spec/skill-memory-ranking-followups/acceptance/<TASK-ID>.md`
3. 不允许修改任务范围之外的文件。
4. 不允许修改 `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`、全局配置文件，除非当前 TASK 明确允许。
5. 不允许大规模格式化无关文件。
6. 不允许为了通过类型检查随意使用 `any`。
7. 不允许吞掉异常；必须保留错误语义、日志和可观测性。
8. 不允许删除已有测试或弱化已有断言。
9. 不允许改变 TASK-001..008 已落地的算法公式 / proto 字段 / 函数签名。
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
- proto 字段编号必须只追加，不重用。
- TASK-001..008 已落地的算法公式（`relevance = w_v·vec + w_l·lex + w_m·meta` 等）不得修改。

## 4. 兼容性规则

- `config.Ranking` 字段为 additive；空值时退回到 hardcode 默认（不破坏现有行为）。
- `DebugSemanticRetrieval` / `RankSkillCandidates` / `RankMemoryCandidates` 公开签名不变。
- 现有 2 个 TASK-006 既有测试需保留；如需调整读取方式，须在报告中说明。
- 旧 4 个包级 helper（`stashDebugPoolConfidence` / `lastDebugPoolConfidenceForTest` / `stashLastDebugMemoryRankings` / `getLastDebugMemoryRankings`）在本 SPEC 内被 ctx helper 替换；调用方同步替换。
- proto / pb 不允许修改（TASK-005 已完成；CI 自检）。
- 前端 TypeScript 类型不允许修改（TASK-007 已完成）。

## 5. 权重 / 阈值常量规则

所有权重 / 阈值集中在 `logic-grpc-service/service/skill_memory_ranking.go` 顶部；TASK-FU-004 把 `const` 改为 `var`，默认值与 SDD §3.3 完全一致：

```go
var (
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

任何对这些 var 的调整必须：
1. 同步更新 `config.Ranking` 段默认值。
2. 通过 `go test ./...` 全部测试。
3. 在 TASK 完成报告中说明。

## 6. 日志规则

- `LoadRankingConfig` 使用 `logger.L().Info` 输出当前生效的权重。
- 越界 clamp 使用 `logger.L().Warn` 输出原因。
- debug 视图的 pool_confidence 高亮通过 UI 表现，不增加日志。

## 7. 测试规则

- 不得使用真实 embedding provider；使用 `fakeEmbeddingProvider` 或 nil provider 即可。
- 不得使用 `any` 绕过类型检查；如必要可使用 `interface{}`。
- 不得删除已有测试；不得弱化已有断言。
- 现有 5 个 TASK-001..004 既有测试（如 `TestComputeRelevanceScore` / `TestComputePoolConfidence`）必须通过。
- CI protoc 步骤失败时输出 diff 行号。

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
- 需要修改 TASK-005 已落地的 proto 字段编号 / 类型 / 语义。
- 需要修改 TASK-007 已落地的前端类型。
- 需要修改 TASK-001..008 已落地的算法公式。
- 需要修改 `model/**` / `repository/**`。
- 需要改变 `DebugSemanticRetrieval` / `RankSkillCandidates` / `RankMemoryCandidates` 函数签名。
- 需要在 debug 视图引入新功能（如 A/B 按钮、阈值调参 UI）。
- 需要新增第三方依赖。
- TASK 范围不足以完成当前需求。
- `requiresHumanConfirmation: true` 的 TASK 在开始编码前。
