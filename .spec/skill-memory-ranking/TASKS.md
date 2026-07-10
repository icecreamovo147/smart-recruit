# TASKS - skill-memory-ranking

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | 新增混合打分数据模型与权重常量 | pending | skill_memory_ranking.go（新增） | acceptance/TASK-001.md |
| TASK-002 | 实现 Skill 池混合打分排序 | pending | agent_skill_selector.go / skill_memory_ranking.go | acceptance/TASK-002.md |
| TASK-003 | 实现 Memory 池混合打分排序 | pending | agent_context.go / skill_memory_ranking.go | acceptance/TASK-003.md |
| TASK-004 | 引入 top1/top2 gap 与池置信度计算 | pending | skill_memory_ranking.go / DebugSemanticRetrieval | acceptance/TASK-004.md |
| TASK-005 | 扩展 proto 新增 score breakdown 字段 | pending | recruitment.proto / recruitment.pb.go（追加字段） | acceptance/TASK-005.md |
| TASK-006 | DebugSemanticRetrieval 填充 score breakdown 与 pool_confidence | pending | agent_skill_service.go（gRPC） | acceptance/TASK-006.md |
| TASK-007 | 前端类型同步追加 optional 字段 | pending | hr-frontend/src/types/agentSkill.ts | acceptance/TASK-007.md |
| TASK-008 | 补充单测与表驱动测试 | pending | *_test.go（新增 / 更新） | acceptance/TASK-008.md |

## TASK-001 - 新增混合打分数据模型与权重常量

### Goal

新增 `logic-grpc-service/service/skill_memory_ranking.go`，定义 `RankingSignals` / `RankConfidence` 数据结构、`computeRelevanceScore` / `computeBusinessBoost` / `computeFinalRankScore` / `ComputePoolConfidence` 工具函数，以及全部权重与阈值常量。要求所有信号归一化到 [0, 1]、`final_rank_score = relevance * business_boost`，并提供单测覆盖公式、gating、置信度等关键路径。

### Scope

- 允许新建：
  - `logic-grpc-service/service/skill_memory_ranking.go`
  - `logic-grpc-service/service/skill_memory_ranking_test.go`
- 不允许修改任何业务逻辑文件、proto、pb、前端。

### Allowed Files

- `logic-grpc-service/service/skill_memory_ranking.go`（新建）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（新建）

### Forbidden Files

- 任何 `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/recruitment/pb/**`
- `web-gin-service/**`
- `hr-frontend/**`
- `logic-grpc-service/service/agent_skill_selector.go` 及其测试
- `logic-grpc-service/service/agent_context.go` 及其测试
- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/config/**`
- `logic-grpc-service/model/**`
- `logic-grpc-service/repository/**`
- `deploy/**` / `docker/**`

### Dependencies

无前置依赖。

### Acceptance Criteria

- AC-001：`skill_memory_ranking.go` 定义 `RankingSignals`、`RankConfidence` 类型与权重 / 阈值常量。
- AC-002：`computeRelevanceScore(vector, lexical, metadata) float64` 单调、合理、clamp 到 [0, 1]。
- AC-003：`computeBusinessBoost(priority, importanceSignal, confidenceSignal, isSkill) float64` clamp 到 [1.0, 1.5]。
- AC-004：`computeFinalRankScore(relevance, boost) float64` = `relevance * boost`。
- AC-005：`ComputePoolConfidence([]RankingSignals) RankConfidence` 输出 4 种枚举之一。
- AC-006：单测覆盖 ≥ 6 种典型组合；`go test ./...` 通过。

### Required Tests

- `TestComputeRelevanceScore`
- `TestComputeBusinessBoost`
- `TestComputeFinalRankScore`
- `TestMemoryImportanceGatedByRelevance`（针对 Memory gating 在本 TASK 验证 helper 行为）
- `TestComputePoolConfidence`

### Risks

- 权重常量与 SDD 文档不一致：需逐项核对 SDD §3.3。
- 命名冲突：`RankingSignals` 与未来其他模块命名冲突时，统一通过包内 `type` 解决。

### Notes

- 权重 / 阈值常量集中在文件顶部，便于后续 review 调整。
- 工具函数须为纯函数，不依赖全局状态、不读取 embedding / repository。
- 不得修改现有 `agent_skill_selector.go` / `agent_context.go`。

## TASK-002 - 实现 Skill 池混合打分排序

### Goal

让 Skill 池（`selectAgentSkillsWithSemantic` 自动排序阶段）走混合打分：新 `final_rank_score = relevance_score * business_boost`，priority 仅作为 boost 系数。保留 `selectedAgentSkill.Score int` 旧字段作为兼容映射；扩展 `selectedAgentSkill` 新增 `VectorScore / LexicalScore / MetadataScore / RelevanceScore / BusinessBoost / FinalRankScore / RelevanceMode / PoolRank / RankingConfidence` 字段。保留 `selectAgentSkills` / `selectAgentSkillsWithSemantic` 函数签名。

### Scope

- 修改 `agent_skill_selector.go`：扩展 `selectedAgentSkill` 字段、用 `scoreSkillRankingSignals` 替换自动排序的加权和、扩展 `selectedAgentSkillTraceItems` 输出 breakdown。
- 修改 `agent_skill_selector_test.go`：必要时把对 `Score` 整数的强断言更新到 `FinalRankScore`。
- 新增 `skill_memory_ranking.go` 中的 `scoreSkillRankingSignals(question, skill, semanticScore, embeddingAvailable) RankingSignals` 与 `RankSkillCandidates(...) []selectedAgentSkill` 函数（与 TASK-001 同一个文件，本 TASK 在该文件追加）。

### Allowed Files

- `logic-grpc-service/service/agent_skill_selector.go`
- `logic-grpc-service/service/agent_skill_selector_test.go`
- `logic-grpc-service/service/skill_memory_ranking.go`（追加 Skill 部分）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 Skill 测试）

### Forbidden Files

- `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/recruitment/pb/**`
- `web-gin-service/**`
- `hr-frontend/**`
- `logic-grpc-service/service/agent_context.go` 及其测试
- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/config/**` / `logic-grpc-service/model/**` / `logic-grpc-service/repository/**`
- `deploy/**` / `docker/**`

### Dependencies

- TASK-001 完成。

### Acceptance Criteria

- AC-001：`selectAgentSkillsWithSemantic` 自动排序阶段使用 `final_rank_score` 排序；手动 Skill 路径保持不变。
- AC-002：`selectedAgentSkill` 扩展字段被填充；旧 `Score int` 字段值 = `int(round(final_rank_score * 100))`。
- AC-003：`scoreAgentSkillMatch` 不被直接修改行为；可作为 `lexical_score` / `metadata_score` 归一化的内部 helper 复用。
- AC-004：现有 `TestSelectAgentSkillsManualPriorityAndAutoMatch` / `TestSelectAgentSkillsUsesPriorityAndRequiredCapabilities` / `TestSelectAgentSkillsUsesSemanticScoresOnlyToRerankRuleCandidates` 全部通过（必要时更新 `Score` 断言为 `FinalRankScore`）。
- AC-005：新增 `TestRankSkillCandidatesAllModes` 覆盖 vector / lexical / 混合 / 降级 4 种 mode。
- AC-006：`go test ./...` 通过。

### Required Tests

- 现有 `agent_skill_selector_test.go` 全部通过。
- `TestRankSkillCandidatesAllModes`
- `TestSkillPoolFallbackToLexicalMetadata`

### Risks

- 旧测试对 `Score` 整数 / 关键词命中加权和的强断言可能因新模型调整而失败；TASK 范围内更新断言。
- `selectedAgentSkillTraceItems` 输出变化可能影响前端展示，遵守 `FinalRankScore = score` 的兼容映射。

### Notes

- 不要修改 `selectAgentSkills` / `selectAgentSkillsWithSemantic` 函数签名。
- 不得修改 `model.AgentSkill` / `model.AgentSkillVersion`。
- 不得修改 proto / pb。

## TASK-003 - 实现 Memory 池混合打分排序

### Goal

让 Memory 池（`AgentContextBuilder.rankMemories` 内部）走混合打分：新 `final_rank_score = relevance_score * business_boost`，`importance` / `confidence` 在相关性 gating 阈值之上才参与 boost。保留 `rankMemories` / `retrieveMemories` 函数签名；保留 Memory 池 `maxMemories` + `maxMemoryChars` 双重截断。

### Scope

- 修改 `agent_context.go`：扩展 `rankMemories` 内部实现，新增 `scoreMemoryRankingSignals` 与 `RankMemoryCandidates(...)` 调用。
- 修改 `agent_context_memory_test.go`：必要时更新对 `score` 的强断言。
- 在 `skill_memory_ranking.go` 追加 Memory 部分的 helper（与 TASK-001 / TASK-002 同文件）。

### Allowed Files

- `logic-grpc-service/service/agent_context.go`
- `logic-grpc-service/service/agent_context_memory_test.go`
- `logic-grpc-service/service/skill_memory_ranking.go`（追加 Memory 部分）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 Memory 测试）

### Forbidden Files

- `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/recruitment/pb/**`
- `web-gin-service/**`
- `hr-frontend/**`
- `logic-grpc-service/service/agent_skill_selector.go` 及其测试
- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/config/**` / `logic-grpc-service/model/**` / `logic-grpc-service/repository/**`
- `deploy/**` / `docker/**`

### Dependencies

- TASK-001 完成。
- TASK-002 完成（保证 Skill 池同步走新打分；本 TASK 不依赖其内部实现，只依赖 `skill_memory_ranking.go` 已存在）。

### Acceptance Criteria

- AC-001：`rankMemories` 内部使用 `final_rank_score` 排序，gating 后 `importance / confidence` 才参与 boost。
- AC-002：`maxMemories` + `maxMemoryChars` 双重截断保持；现有 `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 通过。
- AC-003：现有 `keywordMemoryScore` / `memoryBaseRecallScore` / `semanticMemoryScores` 行为保持（可作为内部 helper 复用），不修改对外签名。
- AC-004：新增 `TestRankMemoryCandidatesAllModes` 覆盖 vector / lexical / 混合 / 降级 4 种 mode。
- AC-005：新增 `TestMemoryImportanceGatedByRelevance` 验证 gating。
- AC-006：`go test ./...` 通过。

### Required Tests

- 现有 `agent_context_memory_test.go` 全部通过。
- `TestRankMemoryCandidatesAllModes`
- `TestMemoryImportanceGatedByRelevance`

### Risks

- 现有 `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 排序期望与新模型可能不一致；TASK 范围内更新断言或在注释中说明 ranking 模式切换。
- 不得修改 `model.AIMemory` 表结构。

### Notes

- 不得修改 `AgentContextInput` / `AgentContext` 结构。
- 不得修改 `Build` / `retrieveMemories` / `rankMemories` 函数签名。
- Memory `importance` / `confidence` 默认在 `relevance_score < rankRelevanceGate` 时信号为 0；Gating 仅对 Memory 生效，Skill 不应用。

## TASK-004 - 引入 top1/top2 gap 与池置信度计算

### Goal

在 `skill_memory_ranking.go` 提供 `ComputePoolConfidence`，并把它接入 Skill 池 / Memory 池的 debug 输出（`DebugSemanticRetrieval`）。在 gRPC response 中暴露 `skill_pool_confidence` / `memory_pool_confidence`。

### Scope

- 修改 `agent_skill_service.go`：在 `DebugSemanticRetrieval` 与 `semanticDebugMemories` 内部计算并填充 `pool_confidence`（仍然只通过 proto 新字段透传，本 TASK 不修改 proto，由 TASK-005 / TASK-006 完成 proto / 透传）。
- 在 `skill_memory_ranking.go` 追加 helper：本 TASK 内允许在 `selectedAgentSkillTraceItems` 与 Memory 调试 trace 中写入 `pool_rank` / `ranking_confidence` 字段（仅 internal 字段；不依赖 proto）。

### Allowed Files

- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/agent_skill_selector.go`（追加 `pool_rank` / `ranking_confidence` 字段填充）
- `logic-grpc-service/service/agent_context.go`（追加 Memory 调试 trace 的新字段填充）
- `logic-grpc-service/service/skill_memory_ranking.go`
- `logic-grpc-service/service/skill_memory_ranking_test.go`

### Forbidden Files

- `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/recruitment/pb/**`
- `web-gin-service/**`
- `hr-frontend/**`
- `logic-grpc-service/config/**` / `logic-grpc-service/model/**` / `logic-grpc-service/repository/**`
- `deploy/**` / `docker/**`

### Dependencies

- TASK-002 完成。
- TASK-003 完成。

### Acceptance Criteria

- AC-001：`ComputePoolConfidence` 单测覆盖 4 种枚举。
- AC-002：`selectedAgentSkillTraceItems` / Memory 调试 trace 输出新字段；proto 透传由 TASK-005 / TASK-006 完成。
- AC-003：`DebugSemanticRetrieval` 内部计算 `skill_pool_confidence` / `memory_pool_confidence` 并填充到 response（即使 proto 字段未生成也应使用通用结构暂存）。
- AC-004：`go test ./...` 通过。

### Required Tests

- `TestComputePoolConfidence`
- `TestSkillPoolConfidenceFromRankings`
- `TestMemoryPoolConfidenceFromRankings`

### Risks

- proto 字段尚未生成时，gRPC response 透传会编译失败；本 TASK 的实施顺序应在 TASK-005 / TASK-006 之前或同步，避免长链路断点。

### Notes

- 不得修改 `DebugSemanticRetrieval` 函数签名。
- 不得修改 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` 现有字段。
- 本 TASK 允许在 `agent_skill_service.go` 引入新的 internal struct 来暂存 `pool_confidence`，不依赖 proto。

## TASK-005 - 扩展 proto 新增 score breakdown 字段

### Goal

在 `recruitment.proto` 中仅追加新字段，不修改现有字段编号：

- `SemanticSkillDebugItem`：追加 `vector_score` (10) / `lexical_score` (11) / `metadata_score` (12) / `relevance_score` (13) / `business_boost` (14) / `final_rank_score` (15) / `relevance_mode` (16) / `pool_rank` (17)。
- `SemanticMemoryDebugItem`：追加 12..19 字段。
- `DebugSemanticRetrievalResponse`：追加 `skill_pool_confidence` (7) / `memory_pool_confidence` (8)。

同步更新 `logic-grpc-service/recruitment/pb/recruitment.pb.go` 与 `web-gin-service/recruitment/pb/recruitment.pb.go`（如项目使用 protoc 生成，遵循现有 CI / scripts；如采用手工同步，本 TASK 范围内手工同步最小集合）。

### Scope

- 修改 `logic-grpc-service/proto/recruitment.proto`（仅追加，不删除 / 改编号）。
- 修改 `logic-grpc-service/recruitment/pb/recruitment.pb.go`。
- 修改 `web-gin-service/recruitment/pb/recruitment.pb.go`（如未生成，提示人工跑 protoc）。

### Allowed Files

- `logic-grpc-service/proto/recruitment.proto`
- `logic-grpc-service/recruitment/pb/recruitment.pb.go`
- `web-gin-service/recruitment/pb/recruitment.pb.go`

### Forbidden Files

- `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `web-gin-service/handler/**` / `web-gin-service/router/**` / `web-gin-service/rpc/**`
- `hr-frontend/**`
- `logic-grpc-service/service/**`（除 recruiter 注：本 TASK 不修改 service 业务代码，由 TASK-006 完成填充）
- `deploy/**` / `docker/**`

### Dependencies

- TASK-004 完成（已确定字段命名）。

### Acceptance Criteria

- AC-001：proto 字段追加成功，protoc 编译通过。
- AC-002：pb 文件同步更新（如生成）；web-gin / logic-grpc 两侧 pb 字段一致。
- AC-003：未触碰现有字段编号与类型。
- AC-004：`go build ./...` 在 logic-grpc-service 与 web-gin-service 全部通过。

### Required Tests

- 无新增测试；本 TASK 主要验证 proto 编译。

### Risks

- pb 文件手工同步可能漏字段，建议优先尝试 protoc；如不可用，逐字段手工同步。
- proto 改动属于公共契约变更，本 TASK 设置为 `requiresHumanConfirmation: true`。

### Notes

- 必须在 `task-scope.json` 中标记 `requiresHumanConfirmation: true`，开始改代码前先与用户确认。
- 不得修改 `recruitment_grpc.pb.go`。

## TASK-006 - DebugSemanticRetrieval 填充 score breakdown 与 pool_confidence

### Goal

在 `agent_skill_service.go` 的 `DebugSemanticRetrieval` 路径下，将新 score breakdown 字段与 `pool_confidence` 填充到 gRPC response。

### Scope

- 修改 `agent_skill_service.go`：`semanticDebugSkillsToPB` / `semanticDebugMemories` 填充新字段；`DebugSemanticRetrieval` 填充 `skill_pool_confidence` / `memory_pool_confidence`。
- 必要时修改 `agent_skill_selector.go` / `agent_context.go` 中对应的 trace helper，确保数据可被 `semanticDebugSkillsToPB` / `semanticDebugMemories` 读取。

### Allowed Files

- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/agent_skill_selector.go`（仅在 `semanticDebugSkillsToPB` 读取新字段需要 helper 时）
- `logic-grpc-service/service/agent_context.go`（仅在 `semanticDebugMemories` 读取新字段需要 helper 时）

### Forbidden Files

- `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/recruitment/pb/**`
- `web-gin-service/**`
- `hr-frontend/**`
- `logic-grpc-service/config/**` / `logic-grpc-service/model/**` / `logic-grpc-service/repository/**`
- `deploy/**` / `docker/**`

### Dependencies

- TASK-004 完成。
- TASK-005 完成。

### Acceptance Criteria

- AC-001：`DebugSemanticRetrieval` 响应中每个 Skill item 携带新字段（`vector_score / lexical_score / metadata_score / relevance_score / business_boost / final_rank_score / relevance_mode / pool_rank`）。
- AC-002：每个 Memory item 携带对应字段。
- AC-003：response 顶层携带 `skill_pool_confidence` / `memory_pool_confidence`。
- AC-004：现有 debug 字段（`embedding_available / fallback_reason`）保持。
- AC-005：`go test ./...` 通过。

### Required Tests

- 必要时在 `agent_skill_service_test.go` / `agent_skill_selector_test.go` / `agent_context_memory_test.go` 增加 trace 字段断言。

### Risks

- proto 字段未生成时，本 TASK 不能开始；TASK-005 必须先完成。
- 旧 response 字段含义不变；`score` = `final_rank_score`，保持兼容。

### Notes

- 不得修改 `DebugSemanticRetrieval` 入参。
- 不得修改 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` 现有字段。

## TASK-007 - 前端类型同步追加 optional 字段

### Goal

`hr-frontend/src/types/agentSkill.ts` 同步追加新字段为 optional，保持 `score` 字段类型与语义不变。

### Scope

- 修改 `hr-frontend/src/types/agentSkill.ts`：在 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `SemanticRetrievalDebugResult` 末尾追加 optional 字段。

### Allowed Files

- `hr-frontend/src/types/agentSkill.ts`

### Forbidden Files

- `hr-frontend/src/views/**`（debug 视图 UI 改动不在本 TASK 范围）
- `hr-frontend/package.json`
- `pnpm-lock.yaml` / `pnpm-workspace.yaml` / `package.json`
- `deploy/**` / `docker/**`
- `web-gin-service/**`
- `logic-grpc-service/**`

### Dependencies

- TASK-006 完成（gRPC 字段已确定）。

### Acceptance Criteria

- AC-001：`SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `SemanticRetrievalDebugResult` 类型追加 optional 字段。
- AC-002：`pnpm --filter hr-frontend typecheck` 通过。
- AC-003：旧字段类型不变。

### Required Tests

- 无新增测试；通过 `pnpm --filter hr-frontend typecheck` 验证。

### Risks

- 字段命名拼写不一致；TASK 实施前需对齐 proto 字段名（snake_case）。

### Notes

- 不得删除或修改现有字段。
- 不在 debug 视图引入新 UI；如需展示 score breakdown 留待后续 TASK。

## TASK-008 - 补充单测与表驱动测试

### Goal

新增并完善单测与表驱动测试，覆盖混合打分、降级路径、置信度、gating、分池隔离等关键路径。确保现有 `go test ./...` 与 `pnpm --filter hr-frontend typecheck` 全部通过。

### Scope

- 修改 / 追加以下测试文件：
  - `logic-grpc-service/service/skill_memory_ranking_test.go`（新增 / 追加）
  - `logic-grpc-service/service/agent_skill_selector_test.go`（必要时更新）
  - `logic-grpc-service/service/agent_context_memory_test.go`（必要时更新）
  - `logic-grpc-service/service/agent_skill_service_test.go`（必要时新增 trace 字段断言）

### Allowed Files

- `logic-grpc-service/service/skill_memory_ranking_test.go`
- `logic-grpc-service/service/agent_skill_selector_test.go`
- `logic-grpc-service/service/agent_context_memory_test.go`
- `logic-grpc-service/service/agent_skill_service_test.go`（如已存在）

### Forbidden Files

- `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`
- `logic-grpc-service/proto/**`
- `logic-grpc-service/recruitment/pb/**`
- `web-gin-service/**`
- `hr-frontend/**`
- `logic-grpc-service/service/*.go`（除测试文件外）
- `deploy/**` / `docker/**`

### Dependencies

- TASK-001 ~ TASK-007 全部完成。

### Acceptance Criteria

- AC-001：新增测试覆盖 SPEC §5 FR-010 列出的全部场景。
- AC-002：现有 `go test ./...` 全部通过。
- AC-003：现有 `pnpm --filter hr-frontend typecheck` 通过。
- AC-004：不删除已有测试；不弱化已有断言。

### Required Tests

- `TestComputeRelevanceScore`
- `TestComputeBusinessBoost`
- `TestComputeFinalRankScore`
- `TestMemoryImportanceGatedByRelevance`
- `TestComputePoolConfidence`
- `TestRankSkillCandidatesAllModes`
- `TestRankMemoryCandidatesAllModes`
- `TestSkillPoolFallbackToLexicalMetadata`
- `TestMemoryPoolFallbackToLexicalMetadata`
- `TestPoolIsolation`（Skill / Memory 互不污染）

### Risks

- 现有测试对 `Score` 整数 / 排序的强断言可能因新模型调整而失败；TASK 范围内更新断言。
- 不得 mock 出 NaN / Inf 模拟真实 embedding 异常，应通过真实 helper 路径覆盖。

### Notes

- 测试不得依赖真实 embedding provider；使用 fake / nil provider 即可。
- 测试不得使用 `any` 绕过类型；如必要可使用 `interface{}`。
- 测试不得删除已有断言；如必须修改，需在报告中说明。
