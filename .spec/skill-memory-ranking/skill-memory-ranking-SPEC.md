# Skill / Memory 召回排序重构 SPEC

## 1. 背景

当前 Smart Recruit 的 Agent 召回存在两套独立的打分逻辑：

- `logic-grpc-service/service/agent_skill_selector.go` 中 `scoreAgentSkillMatch` 返回的整数关键词命中分，加上 `int(semanticScore * 100)`，再叠加 `skill.Priority`。
- `logic-grpc-service/service/agent_context.go` 中 `memoryBaseRecallScore` 以 `Importance*20 + Confidence*10 + scope_bonus` 加上 `semanticScore*100` 或 `keywordMemoryScore` 形成最终分。

两套逻辑都把**关键词命中、向量相似度、业务字段 (priority / importance / confidence / scope)** 全部裸加在一起，分数没有归一化，分池不清晰，召回结果在 debug 视图和业务日志中只能看到一个裸分。

问题：

1. **裸加导致量纲错配**：embedding cosine ∈ [0, 1]，被乘 100 后量级是 [0, 100]；rule 命中是 token 加权，可达数十；priority 是 `int32` 任意值；三者直接相加，谁主导排序取决于具体 query，不可控。
2. **业务字段污染相关性**：低相关性但 importance 高的 Memory 会因为裸加分排到第一。
3. **Skill / Memory 排序混合**：当前没有显式分池概念，UI 端只能在两列分别展示，但底层排序没有保证两池独立。
4. **置信度不可观测**：top1 与 top2 差距小、或者 top1 不到绝对阈值时，调用方无法判断"这次召回可不可信"。
5. **debug 单一分数字段**：现有 `SemanticSkillDebugItem.score` / `SemanticMemoryDebugItem.score` 只有一个数字，无法支撑回归定位。
6. **向量不可用时表现不稳定**：`agent_skill_selector` 走规则路径、`agent_context` 走 `keywordMemoryScore`，行为依赖具体实现细节，没有统一的降级契约。

## 2. 目标

将 Skill / Memory 召回从「裸加分」重构为「相关性 × 业务 boost」的生产级混合排序方案：

1. 引入统一的打分数据模型 `RankingSignals`，把 `vector_score / lexical_score / metadata_score / business_boost` 全部归一化到 [0, 1]。
2. 区分 `relevance_score`（相关性） 和 `business_boost`（业务加权）。
3. `final_rank_score = relevance_score * business_boost`。
4. Skill 池和 Memory 池各自独立排序，互不污染。
5. Embedding 不可用时统一降级到 `lexical + metadata`。
6. priority 仅作为轻量 boost（不再裸加）。
7. Memory `importance` / `confidence` 在相关性达标后才生效（相关性 gating）。
8. 引入 top1/top2 gap 置信度指标。
9. debug 输出展示完整 score breakdown。
10. 保持对外接口兼容：proto / 前端类型只新增可选字段，不删不改现有字段。
11. 补充单元测试和表驱动测试覆盖混合打分、降级、置信度、gating 等关键路径。

## 3. 非目标

本期不做的内容：

1. 不修改 `model.AgentSkill` / `model.AIMemory` 表结构。
2. 不替换 `EmbeddingService` / `EmbeddingProvider` 的实现。
3. 不改变 `ai_embeddings` 表的存储和检索语义。
4. 不修改 `maxAgentSkillsPerRequest = 3`、`defaultMaxMemories = 10`、`defaultMaxMemoryChars = 1500` 等现有容量上限。
5. 不引入新的第三方依赖，不修改 `package.json` / `pnpm-lock.yaml`。
6. 不重写规则召回的 tokenizer（`tokenizeAgentSkillText` / `addCJKAgentSkillTokens`）。
7. 不改变 `AIService` / `AgentSkillService` / `AgentContextBuilder` 的对外调用方式。
8. 不在 debug 视图上引入新的业务功能（如 A/B、阈值调参 UI），仅展示新指标。
9. 不在生产侧引入实时权重配置；本期 hardcode 权重常量，便于回归。
10. 不替换 `scoreAgentSkillMatch` 内部的字符级 / CJK n-gram 算法，只调整它返回的语义角色。

## 4. 用户角色与使用场景

### 用户角色

- **AI / Agent 管理员**：在语义召回调试页验证混合打分后的 Top K 结果，定位排序异常。
- **HR 业务用户**：间接受益于更稳定的召回顺序。
- **运维 / 研发**：通过 debug 日志、trace、gRPC response 复现召回行为。
- **自动化测试 / 回归测试**：使用单测与表驱动测试验证排序在多种 query / 候选组合下的行为。

### 使用场景

1. 管理员输入一个真实 HR 问题，调试页同时展示 Skill 与 Memory 两列的 rank_score、relevance_score、business_boost、top1/top2 gap、运行模式（embedding 可用 / 降级）。
2. Embedding Provider 不可用时，debug 视图显示降级模式，分数来自 lexical + metadata，但分池与置信度计算保持一致。
3. 研发在 CI 中跑 `go test ./...`，新的表驱动测试覆盖：纯向量命中、纯规则命中、混合命中、降级路径、置信度为低的情况、Memory importance gating。
4. 运营 / 研发在日志中看到 `final_rank_score`、`relevance_score`、`business_boost` 三个字段，便于排障。

## 5. 功能需求

### FR-001：统一打分数据模型

- 描述：新增内部数据结构 `RankingSignals`，用于描述一次召回的各类得分。
- 字段：
  - `vector_score` float64，范围 [0, 1]，来自 embedding cosine similarity。
  - `lexical_score` float64，范围 [0, 1]，来自关键词 / n-gram 命中。
  - `metadata_score` float64，范围 [0, 1]，来自 category / scenario / agent_type / scope 等元数据匹配。
  - `business_boost` float64，范围 [1.0, business_boost_max]，默认上界 1.5；priority / importance / confidence 在相关性 gating 之后折叠到本字段。
  - `relevance_score` float64，范围 [0, 1]，由 `vector_score / lexical_score / metadata_score` 组合得出。
  - `final_rank_score` float64，`relevance_score * business_boost`，范围 [0, business_boost_max]。
- 触发条件：每个候选 Skill / Memory 在排序时构造一组 `RankingSignals`。
- 输入：原始 `question`、候选对象、可选 embedding score、embedding 可用标志。
- 输出：归一化的 `RankingSignals`，含 `final_rank_score`。
- 边界：
  - 当 embedding 不可用时 `vector_score = 0` 但 `relevance_mode = "lexical_metadata"`。
  - 当所有信号都为 0 时 `final_rank_score = 0`，该候选在默认 gating 下被丢弃。

### FR-002：相关性 + 业务 boost 分离

- 描述：实现 `computeRelevanceScore` 和 `computeBusinessBoost`。
- `relevance_score` 公式（推荐默认值，hardcode）：
  - `relevance = w_v * vector_score + w_l * lexical_score + w_m * metadata_score`
  - 默认 `w_v = 0.6`，`w_l = 0.3`，`w_m = 0.1`，三个权重之和等于 1。
- `business_boost` 公式（推荐默认值，hardcode）：
  - `boost = 1.0 + alpha * priority_normalized + beta * importance_signal + gamma * confidence_signal`
  - `priority_normalized` = `clamp(priority / priority_norm, 0, 1)`，其中 `priority_norm = 50`。
  - `importance_signal`、`confidence_signal` 在相关性 gating 之前为 0；相关性达标后取 `Memory.Importance` / `Memory.Confidence`。
  - 默认 `alpha = 0.2`，`beta = 0.2`，`gamma = 0.1`，`boost` 上界 `business_boost_max = 1.5`。
  - 当 `boost` 计算结果超过 `business_boost_max` 时截断到上限。
- 输出：`relevance_score ∈ [0, 1]`、`business_boost ∈ [1.0, 1.5]`。
- 边界：所有权重、阈值、上限为常量，hardcode 后必须有对应单元测试保护。

### FR-003：Skill / Memory 分池排序

- 描述：Skill 召回和 Memory 召回各自排序、各自截断、互不污染。
- 行为：
  - Skill 池排序时仅在候选 Skill 内部按 `final_rank_score` 排序，取前 `maxAgentSkillsPerRequest`。
  - Memory 池排序时仅在候选 Memory 内部按 `final_rank_score` 排序，按现有 `maxMemories` + `maxMemoryChars` 双重截断。
  - 现有 `manualIDs` 行为保留：手动指定 Skill 仍按现有规则优先加入 selected，再走自动排序填补剩余名额。
- 边界：自动排序填补阶段不再混合两池。

### FR-004：Embedding 不可用降级

- 描述：当 `EmbeddingService` 不可用、超时或返回错误时，自动降级到 `lexical + metadata` 模式。
- 行为：
  - 降级时 `vector_score = 0`，`relevance_mode = "lexical_metadata"`。
  - 降级日志输出 `mode=fallback`，并写入 `embedding_unavailable_reason`。
  - debug response 中暴露 `mode` 字段。
  - 降级后 `business_boost` 行为与正常模式保持一致。
- 边界：降级时 `metadata_score` 至少基于 category / scenario / agent_type 命中计算，不允许 0。

### FR-005：Memory importance / confidence 相关性 gating

- 描述：Memory 的 `importance` 和 `confidence` 只有在 `relevance_score` 达到阈值后才参与 `business_boost`。
- 阈值：默认 `relevance_gate_threshold = 0.15`。
- 行为：
  - `relevance_score < relevance_gate_threshold`：`importance_signal = 0`，`confidence_signal = 0`。
  - `relevance_score >= relevance_gate_threshold`：`importance_signal = clamp(memory.Importance, 0, 1)`，`confidence_signal = clamp(memory.Confidence, 0, 1)`。
- 边界：gating 仅对 Memory 生效，Skill 池不应用。

### FR-006：Top1 / Top2 gap 置信度

- 描述：每个池（Skill 池 / Memory 池）独立计算 top1 与 top2 的 `final_rank_score` 差距。
- 指标：
  - `top1_score` float64
  - `top2_score` float64
  - `gap` float64 = `top1_score - top2_score`
  - `confidence` 枚举：`"high"`（`gap >= 0.10`）、`"medium"`（`0.03 <= gap < 0.10`）、`"low"`（`gap < 0.03` 或 `top1_score < relevance_gate_threshold`）。
- 输出：debug response 增加 `pool_confidence` 字段。
- 边界：
  - 池内候选数 < 2 时 `gap = top1_score`，`confidence` 按 `top1_score` 与 gating 阈值计算。
  - 池内候选数 = 0 时 `confidence = "none"`。

### FR-007：Debug 字段扩展

- 描述：在 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` 中增加 score breakdown 字段。
- 新增字段（全部 optional / proto3 scalar）：
  - `vector_score` double
  - `lexical_score` double
  - `metadata_score` double
  - `relevance_score` double
  - `business_boost` double
  - `final_rank_score` double
  - `relevance_mode` string（`"vector_lexical_metadata"` / `"lexical_metadata"`）
  - `pool_rank` int32（在池内 1-based 排名）
- 保留：原 `score` 字段，语义改为 `final_rank_score`（保持兼容，避免破坏现有调用方）。
- 边界：前端未读新字段时仍能正常显示。

### FR-008：日志可观测

- 描述：每个候选 Skill / Memory 排序时输出 `final_rank_score`、`relevance_score`、`business_boost`、`mode`、`pool_rank` 到 debug 日志。
- 边界：debug 日志在 `logger.GetRequestLogger(ctx).Debug` 级别，不污染 info 级别；embedding 文本、Memory 完整内容不写入日志。

### FR-009：接口兼容

- 描述：proto / 前端 / 外部 API 保持向后兼容。
- 行为：
  - proto 只在 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `DebugSemanticRetrievalResponse` 末尾追加新字段，字段编号不重用。
  - 前端 TypeScript 类型同步追加 optional 字段；`score` 字段类型保持 `number`，语义不变。
  - 现有 `selectedAgentSkill` / 内部 `RankingSignals` 是新增内部结构，调用方无感。
  - 现有单测保持通过（必要时调整断言以反映新分数范围，但仍要求通过）。
- 边界：现有 `go test ./...` 全部通过。

### FR-010：测试覆盖

- 描述：补充单元测试和表驱动测试。
- 必须覆盖：
  - 纯向量命中、纯规则命中、混合命中的 `final_rank_score` 计算。
  - Skill 池 / Memory 池分池排序互不污染。
  - Embedding 降级路径下分数不出现 NaN / 负数 / 超过 1.5。
  - Memory importance / confidence 在 gating 阈值前不生效。
  - top1/top2 gap 与 confidence 枚举。
  - 现有 `TestSelectAgentSkillsManualPriorityAndAutoMatch` 等测试不回归。
- 边界：测试不依赖真实 embedding provider；使用 fake / nil provider 即可。

## 6. 非功能需求

- **性能**：单次 Skill 排序或 Memory 排序的 CPU 耗时增加不超过 20%（相对当前实现）。
- **稳定性**：现有 `go test ./...`、`pnpm --filter hr-frontend typecheck` 通过。
- **可观测性**：所有排序关键路径都有 debug 日志；运行模式（vector / fallback）通过字段可见。
- **可维护性**：权重与阈值集中在同一文件（建议 `service/skill_memory_ranking.go` 或同包等价物），便于后续调整。
- **可回滚**：本期所有新逻辑集中在 1-2 个新文件，必要时可回滚新文件并保留旧实现路径。

## 7. 兼容性需求

- `logic-grpc-service/service/agent_skill_selector.go` 中的 `selectAgentSkills` / `selectAgentSkillsWithSemantic` / `scoreAgentSkillMatch` 函数签名保持不变。
- `logic-grpc-service/service/agent_context.go` 中的 `AgentContextBuilder.Build` / `retrieveMemories` / `rankMemories` 函数签名保持不变。
- `logic-grpc-service/service/agent_skill_service.go` 中的 `DebugSemanticRetrieval` 入参 / 返回结构保持不变。
- `logic-grpc-service/proto/recruitment.proto` 中 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `DebugSemanticRetrievalResponse` 仅追加新字段，不删除 / 修改现有字段编号。
- `web-gin-service` 的 `agent_skill.go` handler 入参 / 出参保持不变。
- `hr-frontend` 的 `agentSkill.ts` 类型与 `SemanticRetrievalDebugView.vue` 调用方式保持兼容；新字段为 optional。

## 8. 可观测性需求

- 排序路径在 `logger.GetRequestLogger(ctx).Debug` 级别输出 score breakdown。
- 降级路径在 `logger.GetRequestLogger(ctx).Info` 级别输出 `mode=fallback` 和 `reason`。
- 置信度在 `pool_confidence` 字段对外暴露。
- 不在日志中输出完整 Memory 内容、Skill markdown 完整内容、API Key、token。

## 9. 异常处理与降级

| 场景 | 行为 |
| --- | --- |
| Embedding provider 不可用 | 走 `lexical + metadata` 模式 |
| Embedding 搜索超时 | 同上 |
| Skill 池为空 | 跳过 Skill 返回 |
| Memory 池为空 | 跳过 Memory 返回 |
| 全部候选 `final_rank_score = 0` | 返回空结果，debug 中 `pool_confidence = "none"` |
| `relevance_score` 全为 0 | Memory `business_boost = 1.0`（无 boost） |
| `business_boost` 计算结果 > 1.5 | 截断到 1.5 |
| Embedding score 为 NaN / Inf | 当作 0 处理，不参与排序 |

## 10. 安全需求

- 不在日志中输出完整简历、Memory、Skill markdown。
- 不在 debug response 中输出 embedding vector 原值。
- 不改变 API Key 存储方式。
- 不影响现有 RBAC 与权限校验。

## 11. 验收标准

- AC-001：`computeRelevanceScore` / `computeBusinessBoost` / `computeFinalRankScore` 单测覆盖至少 6 种典型组合。
- AC-002：现有 `TestSelectAgentSkillsManualPriorityAndAutoMatch`、`TestSelectAgentSkillsUsesPriorityAndRequiredCapabilities`、`TestSelectAgentSkillsUsesSemanticScoresOnlyToRerankRuleCandidates` 不回归。
- AC-003：现有 `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 不回归。
- AC-004：降级路径下，新增 `TestSkillMemoryRankingFallbackToLexicalMetadata` 验证分数在合法范围且模式为 `lexical_metadata`。
- AC-005：新增 `TestMemoryImportanceGatedByRelevance` 验证低相关性时 `importance_signal = 0`。
- AC-006：新增 `TestTopOneTopTwoGapConfidence` 验证 high / medium / low / none 四种 confidence 枚举。
- AC-007：proto `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `DebugSemanticRetrievalResponse` 新增字段，旧字段保持。
- AC-008：`hr-frontend/src/types/agentSkill.ts` 同步更新 optional 字段。
- AC-009：`go test ./...` 通过。
- AC-010：`pnpm --filter hr-frontend typecheck` 通过。

## 12. 不在本期范围

1. 权重 / 阈值的运行时配置（hardcode 本期）。
2. 召回策略的 A/B 实验。
3. 新增 query rewriting / HyDE / reranker 等算法。
4. 替换 Embedding Provider 的实现。
5. 修改 `ai_embeddings` 表结构或检索语义。
6. 引入新依赖。

## 13. 待确认问题

- 权重 `w_v=0.6 / w_l=0.3 / w_m=0.1`、boost 上限 `1.5`、`relevance_gate_threshold=0.15`、confidence 阈值 `0.10/0.03` 是否为合理起点（待产品 / 算法确认）。
- proto 字段是否一次性扩展 `pool_confidence` 还是分两个字段 `top1_score` + `top2_score` + `gap`（建议后者更利于排障）。
- Memory gating 是否只对 `importance` 生效，还是同时对 `confidence` 生效（SPEC 暂定同时 gating；如产品要求不同需要确认）。

## 14. 开放问题

- 是否在 debug 视图新增"权重说明"小卡片，帮助管理员理解当前排序逻辑？
- 是否在日志侧引入 `pool` / `mode` 标签，便于运维筛选？
- 是否需要提供 `final_rank_score` 与旧 `score` 的兼容映射规则（SPEC 默认 `score = final_rank_score`）？
