# Skill / Memory 召回排序重构 SDD

## 1. 现有架构摘要

### 1.1 Skill 召回

文件：`logic-grpc-service/service/agent_skill_selector.go`

- 入口：`selectAgentSkillsWithSemantic(ctx, repo, agentType, question, manualIDs, availableCapabilities, semanticScores)`。
- 行为：
  - 加载 `ListEnabled` Skill。
  - manualIDs 优先加入 selected（要求 `IsManualInvocable=1`）。
  - 自动候选阶段：对每个 skill 调用 `scoreAgentSkillMatch`（token 命中加权，meta × 2 + content × 1），加 `int(semanticScore * 100)`，加 `int(skill.Priority)`，合并为整数 `Score`。
  - 按 `Score desc, Priority desc, ID asc` 稳定排序。
  - 截断到 `maxAgentSkillsPerRequest = 3`。
- 调试点：`semanticDebugSkillScores` 调 `EmbeddingService.Search(ObjectTypes=["agent_skill"])`，把 cosine 写到 `selectedAgentSkill` 调试输出。
- 输出：`selectedAgentSkill.Score int`、`Reason string`、debug 输出 `score float64`、`reason`。

### 1.2 Memory 召回

文件：`logic-grpc-service/service/agent_context.go`

- 入口：`AgentContextBuilder.retrieveMemories(ctx, input)` → `rankMemories(ctx, input, all)`。
- 行为：
  - 通过 `MemoryRepo.ListRecallCandidates` 拉取 scope 候选（`hr / job / application`）。
  - `memoryBaseRecallScore`：`Importance*20 + Confidence*10`，叠加 scope 命中 `+12 / +8 / +4`。
  - `semanticScores` 存在时：`score += semanticScore * 100`；否则 `score += keywordMemoryScore(query, content)`。
  - 按 `score desc, index asc` 稳定排序。
  - 按 `maxMemories=10` + `maxMemoryChars=1500` 双重截断。
- 输出：`model.AIMemory`，`DebugSemanticRetrieval` 路径下额外透出 `score / importance / confidence / reason / created_at`。

### 1.3 Debug 视图

文件：`hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`

- 字段：每个 item 仅展示一个 `score` 数字，配合 `reason` 文本。
- 统计：测试状态、Skill 命中、Memory 命中、最高匹配分数、请求耗时、Embedding 状态。

### 1.4 Proto 契约

`logic-grpc-service/proto/recruitment.proto` 中：

```proto
message SemanticSkillDebugItem {
  int64 id = 1;
  string name = 2;
  string display_name = 3;
  string category = 4;
  string scenario = 5;
  int32 priority = 6;
  double score = 7;
  string reason = 8;
  repeated string semantic_tags = 9;
}

message SemanticMemoryDebugItem {
  uint64 id = 1;
  string scope_type = 2;
  uint64 scope_id = 3;
  string memory_type = 4;
  string content = 5;
  string source = 6;
  double confidence = 7;
  double importance = 8;
  double score = 9;
  string reason = 10;
  string created_at = 11;
}

message DebugSemanticRetrievalResponse {
  int32 code = 1;
  string msg = 2;
  bool embedding_available = 3;
  string fallback_reason = 4;
  repeated SemanticSkillDebugItem skills = 5;
  repeated SemanticMemoryDebugItem memories = 6;
}
```

### 1.5 现有日志

- `agent_skill_selector.go`：debug 级别输出 selected skill 的 `id / name / reason / score`。
- `agent_context.go`：info 级别输出 `[上下文预算]`，未输出 score breakdown。
- `agent_skill_service.go`：debug / warn 级别输出 semantic search 状态、embedding_available 标志、fallback_reason。

## 2. 问题分析

| 编号 | 问题 | 影响 |
| --- | --- | --- |
| P-1 | `int(semanticScore * 100) + ruleScore + int(priority)` 三个量级裸加 | 排序结果对 priority 数字大小过度敏感，embedding 信号主导能力受 priority 数值挤压 |
| P-2 | Memory `Importance*20 + Confidence*10` 直接加分 | 低相关性但 importance 高的 Memory 排前 |
| P-3 | Skill 与 Memory 排序没有显式分池 | 现有 UI 分两列但底层排序混合 trace；debug 日志中两者用同一段代码输出 |
| P-4 | top1 / top2 差距不可观测 | 召回不可信时无法在调试页告警 |
| P-5 | `score` 字段单一数字 | debug / 排障 / 回归定位信息不足 |
| P-6 | 降级行为分散 | Skill 走 `scoreAgentSkillMatch`，Memory 走 `keywordMemoryScore`，行为差异无统一契约 |
| P-7 | Memory `confidence` 与 `importance` 在相关性极低时仍然加分 | 容易召回陈年 / 低质 Memory |

## 3. 提议设计

### 3.1 总览

新增内部打分子系统，集中在 `logic-grpc-service/service/skill_memory_ranking.go` 与 `skill_memory_ranking_test.go`：

- 数据结构 `RankingSignals` / `RelevanceScore` / `BusinessBoost` / `RankConfidence`。
- 排序函数 `RankSkillCandidates(...)` / `RankMemoryCandidates(...)`。
- 置信度计算 `ComputePoolConfidence(top1, top2)`。
- Skill 排序入口 `RankAgentSkillsWithHybrid(...)` 替代现有 `selectAgentSkillsWithSemantic` 内部对候选的加权和排序（保留旧函数作为兼容包装）。
- Memory 排序入口 `RankAgentContextMemoriesWithHybrid(...)` 替代 `rankMemories` 内部加权和排序（保留旧函数）。

不改变：

- `selectAgentSkills` / `selectAgentSkillsWithSemantic` 函数签名。
- `AgentContextBuilder.Build` / `retrieveMemories` / `rankMemories` 函数签名。
- 现有 proto message 字段；只在末尾追加新字段。
- 现有前端 `score` 字段；新字段为 optional。

### 3.2 数据结构

```go
type RankingSignals struct {
    VectorScore     float64 `json:"vector_score"`
    LexicalScore    float64 `json:"lexical_score"`
    MetadataScore   float64 `json:"metadata_score"`
    RelevanceScore  float64 `json:"relevance_score"`
    BusinessBoost   float64 `json:"business_boost"`
    FinalRankScore  float64 `json:"final_rank_score"`
    RelevanceMode   string  `json:"relevance_mode"` // "vector_lexical_metadata" | "lexical_metadata"
    Reasons         []string `json:"reasons,omitempty"`
}

type RankConfidence string

const (
    RankConfidenceHigh   RankConfidence = "high"
    RankConfidenceMedium RankConfidence = "medium"
    RankConfidenceLow    RankConfidence = "low"
    RankConfidenceNone   RankConfidence = "none"
)
```

### 3.3 权重与阈值常量

集中在 `skill_memory_ranking.go` 顶部，方便后续调整：

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

### 3.4 公式

```text
relevance_score = clamp01(rankWeightVector * vector_score
                        + rankWeightLexical * lexical_score
                        + rankWeightMetadata * metadata_score)

boost =
  1.0
  + rankBoostAlpha * priority_normalized      // Skill only
  + rankBoostBeta  * importance_signal        // Memory only, gated
  + rankBoostGamma * confidence_signal        // Memory only, gated
clamp(boost, 1.0, rankBusinessBoostMax)

final_rank_score = relevance_score * boost
```

- `priority_normalized` = `clamp01(priority / rankPriorityNorm)`。
- `importance_signal` / `confidence_signal` 在 `relevance_score >= rankRelevanceGate` 时取 `clamp01(memory.Importance)` / `clamp01(memory.Confidence)`，否则为 0。

### 3.5 Skill 排序实现

`selectAgentSkillsWithSemantic` 内部：

1. 现有 `manualIDs` 逻辑保持不变，仍走旧路径直接 selected。
2. 自动候选阶段：调用 `scoreSkillRankingSignals(question, skill, semanticScore, embeddingAvailable)`，得到 `RankingSignals`。
3. 过滤 `final_rank_score > 0` 的候选。
4. 用 `sort.SliceStable` 按 `final_rank_score desc, priority desc, id asc` 排序。
5. 截断到 `maxAgentSkillsPerRequest - len(manualSelected)`。
6. 把 `RankingSignals` 写入 `selectedAgentSkill.RankingSignals` 字段（新增内部字段，向后兼容旧调用方）。

`selectedAgentSkill` 新增字段：

```go
type selectedAgentSkill struct {
    // ... 现有字段保持不变
    Score              int
    VectorScore        float64
    LexicalScore       float64
    MetadataScore      float64
    RelevanceScore     float64
    BusinessBoost      float64
    FinalRankScore     float64
    RelevanceMode      string
    PoolRank           int
    RankingConfidence  string
}
```

注意：保留 `Score int`（旧字段）以兼容旧调试 API；新调试 API 使用 `FinalRankScore` 等新字段。

### 3.6 Memory 排序实现

`rankMemories` 内部：

1. 现有 `semanticMemoryScores` / `keywordMemoryScore` 行为保留，组合出原始的 `vector_score / lexical_score / metadata_score`：
   - `vector_score` ∈ [0, 1]，由 semantic 搜索结果或 0（降级）给出。
   - `lexical_score` ∈ [0, 1]，由 `keywordMemoryScore` 归一化（除以最大可能命中 token 数或 1.0，取大者 clip 到 1）。
   - `metadata_score` ∈ [0, 1]，scope 命中计算（application: 1.0 / job: 0.7 / hr: 0.4 / miss: 0）。
2. 调用 `scoreMemoryRankingSignals(memory, input, ...)`，得到 `RankingSignals`，应用 gating。
3. 排序：`final_rank_score desc, importance desc, id asc`。
4. 截断：与 `retrieveMemories` 现有 `maxMemories` + `maxMemoryChars` 双重截断完全一致。

### 3.7 分池与置信度

- 现有 Skill / Memory 路径天然分池（不同入口函数），只需在 debug response 中分别给出 `pool_confidence`。
- 新增 `ComputePoolConfidence(ranked []RankingSignals) RankConfidence`：
  - 空：`none`。
  - 仅 1 个：`confidence` 由 `top1.FinalRankScore >= rankRelevanceGate ? low : none` 给出。
  - 多个：`gap = top1.FinalRankScore - top2.FinalRankScore`；`gap >= rankGapHigh → high`；`gap >= rankGapMedium → medium`；否则 `low`；当 `top1.FinalRankScore < rankRelevanceGate` 强制 `low`。

### 3.8 Debug 字段

在 `recruitment.proto` 末尾追加（不允许修改现有字段编号）：

```proto
message SemanticSkillDebugItem {
  // ... 现有 1..9 字段保持
  double vector_score = 10;
  double lexical_score = 11;
  double metadata_score = 12;
  double relevance_score = 13;
  double business_boost = 14;
  double final_rank_score = 15;
  string relevance_mode = 16;
  int32 pool_rank = 17;
}

message SemanticMemoryDebugItem {
  // ... 现有 1..11 字段保持
  double vector_score = 12;
  double lexical_score = 13;
  double metadata_score = 14;
  double relevance_score = 15;
  double business_boost = 16;
  double final_rank_score = 17;
  string relevance_mode = 18;
  int32 pool_rank = 19;
}

message DebugSemanticRetrievalResponse {
  // ... 现有 1..6 字段保持
  string skill_pool_confidence = 7;
  string memory_pool_confidence = 8;
}
```

前端 `hr-frontend/src/types/agentSkill.ts` 同步追加 optional 字段；`SemanticRetrievalDebugView.vue` 不强制改动，但允许后续 PR 增加 breakdown 展示。

### 3.9 日志

- Skill 排序：debug 级别输出 `id / name / final_rank_score / relevance_score / business_boost / relevance_mode / pool_rank`。
- Memory 排序：debug 级别输出 `id / scope / final_rank_score / relevance_score / business_boost / relevance_mode / pool_rank`。
- 降级路径：info 级别输出 `mode=fallback reason=...`，**不输出 query 原文**。
- 旧日志字段保留，避免破坏现有日志解析。

### 3.10 兼容映射

- proto `score` 字段值 = `final_rank_score`（保证旧前端不破）。
- `selectedAgentSkill.Score` 旧字段值 = `int(round(final_rank_score * 100))`，仅用于兼容旧 debug 路径（不要在生产中依赖）。
- `agent_skill_selector_test.go` 中已有断言 `len(selected) == 2` 等结构性断言不受影响；可能涉及 `Score` 字段比较的断言需要更新到使用 `FinalRankScore`（TASK 范围内处理）。

## 4. 数据结构变化

- 新增内部结构（Go）：`RankingSignals`、`RankConfidence`，定义在 `service/skill_memory_ranking.go`。
- `selectedAgentSkill` 扩展：新增 8 个 float64 / string / int 字段，全部在 `service/agent_skill_selector.go` 中定义。
- proto 扩展：见 3.8。
- 前端 TypeScript 接口扩展：所有新字段为 optional。

## 5. API 与接口变化

| 接口 | 变化 |
| --- | --- |
| `gRPC: DebugSemanticRetrieval` | 入参不变；返回 message 末尾追加 7 / 8 / pool_confidence 字段 |
| `HTTP: POST /api/v1/hr/admin/agent-skills/semantic-debug` | 入参不变；返回 JSON 透传 proto 新增字段 |
| `gRPC: ListAgentSkills` / `GetAgentSkill` / `CreateAgentSkill` / ... | 不变 |
| `gRPC: ChatStream` / 内部 `AIService.selectAgentSkills` | 函数签名不变；内部实现替换为混合打分 |
| `gRPC: AgentContextBuilder.Build` | 函数签名不变；内部实现替换为混合打分 |

## 6. 算法与工作流变化

| 阶段 | 旧实现 | 新实现 |
| --- | --- | --- |
| Skill rule score | `scoreAgentSkillMatch` 返回 int | 拆分为 `lexical_score` 和 `metadata_score`，各自归一化到 [0, 1] |
| Skill semantic score | `int(semanticScore * 100)` | `vector_score = semanticScore` ∈ [0, 1] |
| Skill total | `int(ruleScore + semantic*100 + priority)` | `final_rank_score = relevance_score * business_boost` |
| Memory rule score | `Importance*20 + Confidence*10 + scope_bonus` | 拆分为 `metadata_score`（scope 命中）+ gating 后的 `importance / confidence` 信号 |
| Memory semantic | `semanticScore * 100` 或 `keywordMemoryScore` | `vector_score` / `lexical_score`，并归一化到 [0, 1] |
| Memory total | `base + semantic*100 + keyword` | `final_rank_score = relevance_score * business_boost` |
| 分池 | 默认两池独立 | 显式 `pool` 标签写入 trace |
| 置信度 | 无 | top1/top2 gap 推断 `high / medium / low / none` |

## 7. 配置设计

- 权重与阈值全部 hardcode 在 `service/skill_memory_ranking.go`。
- 不新增配置文件、不修改 `config.Config`、不修改 `config.example.yaml`。
- 后续如需热调，建议通过新增 `config.Ranking` 段接入，但本期不做。

## 8. 兼容性策略

1. proto 仅追加新字段；旧字段不变；旧 `score = final_rank_score` 兼容映射。
2. 前端 TypeScript 类型追加 optional 字段；现有 `score` 字段类型与显示逻辑不变。
3. `selectedAgentSkill` 旧 `Score int` 字段保留，赋值为 `int(round(final_rank_score * 100))`；旧单测可继续断言 `len(selected)` 等结构。
4. 旧 `scoreAgentSkillMatch` / `memoryBaseRecallScore` 函数保留，可作为内部 helper 复用其 token 命中 / scope 命中计算。
5. 旧 `selectAgentSkillsWithSemantic` / `rankMemories` 函数签名保持不变；仅在内部把"加权和"换成"hybrid 打分"。
6. 现有 `agent_skill_selector_test.go` / `agent_context_memory_test.go` 中如出现对 `Score` 整数 / 加权后值的强断言，需迁移到 `FinalRankScore`；但结构性断言（数量、manual 优先、池隔离等）保持。

## 9. 错误处理与降级

| 触发 | 行为 |
| --- | --- |
| `EmbeddingService == nil` | `vector_score = 0`，`relevance_mode = "lexical_metadata"` |
| `EmbeddingService.Search` 返回 error | 同上 |
| `cosineSimilarity` 返回 NaN / Inf | `vector_score = 0` |
| `semanticMemoryScores` 返回 error | 同上 |
| `priority > 1000` 或 `< -1000` | `priority_normalized` clip 到 [0, 1] |
| `importance / confidence` 越界 [0, 1] | clip |
| 池内全部 `final_rank_score = 0` | debug response 给出 `pool_confidence = "none"`；selected 仍可返回空 |

降级路径必须满足：

- 不抛 error 到 debug API（除非仓库错误）。
- 不输出 query 原文到日志。
- `relevance_score` 仍由 `lexical + metadata` 计算出非 0 值。
- `business_boost` 行为正常。

## 10. 可观测性设计

- `logger.GetRequestLogger(ctx).Debug` 输出每个候选的 score breakdown。
- `logger.GetRequestLogger(ctx).Info` 输出降级原因。
- debug response `embedding_available` + `fallback_reason` 已有，再补 `relevance_mode` 与 `pool_confidence`。
- 不输出 embedding vector、原始 token 列表、API Key。

## 11. 测试策略

### 11.1 单元测试（新增）

- `TestComputeRelevanceScore`：6 种组合（纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值）。
- `TestComputeBusinessBoost`：Skill priority 0/25/50/100、Memory importance / confidence gating。
- `TestComputeFinalRankScore`：验证 `final = relevance * boost` 公式。
- `TestMemoryImportanceGatedByRelevance`：验证 gating 行为。
- `TestTopOneTopTwoGapConfidence`：4 种 confidence 枚举。
- `TestLexicalScoreNormalization`：CJK / 英文 / 混合的 keyword 命中归一化。
- `TestMetadataScoreNormalization`：scope 命中映射到 [0, 1]。
- `TestVectorScoreFallbackToZero`：embedding 不可用时 `vector_score = 0`。

### 11.2 集成 / 表驱动测试

- `TestRankAgentSkillsWithHybridAllModes`：vector / lexical / 混合 / 降级。
- `TestRankAgentContextMemoriesWithHybridAllModes`：同上。
- `TestPoolIsolation`：Skill 池排序不应影响 Memory 池，Memory 池排序不应影响 Skill 池。

### 11.3 兼容性测试

- 现有 `agent_skill_selector_test.go` / `agent_context_memory_test.go` 不回归。
- 必要时更新对 `Score` 整数值的断言到 `FinalRankScore`，但结构性断言（manual 优先、数量、ID 顺序）保持。

### 11.4 前端

- `pnpm --filter hr-frontend typecheck` 通过。
- 不强制新增组件测试（debug 视图改动最小）。

## 12. 迁移风险

| 风险 | 等级 | 缓解 |
| --- | --- | --- |
| 现有 `Score int` 断言的回归 | 中 | 在实现 TASK 中同步更新断言；保留旧字段值映射 |
| proto 字段编号冲突 | 低 | 仅追加，不重用编号 |
| 真实 embedding 维度变化 | 低 | `vector_score` 全部由 cosineSimilarity 给出，量级稳定 |
| 降级路径性能 | 低 | 不引入额外 IO |
| 权重调整需要重新跑全量回归 | 中 | 权重 hardcode，集中在常量；调整时由人触发回归 |
| 旧日志解析失败 | 低 | debug 日志只追加新字段；info 级别不变 |

## 13. 实现边界

| 模块 | 是否本期修改 |
| --- | --- |
| `logic-grpc-service/service/skill_memory_ranking.go` | 新增 |
| `logic-grpc-service/service/skill_memory_ranking_test.go` | 新增 |
| `logic-grpc-service/service/agent_skill_selector.go` | 改：内部使用新打分；保留旧函数签名；扩展 `selectedAgentSkill` 字段 |
| `logic-grpc-service/service/agent_skill_selector_test.go` | 改：必要时更新对 `Score` 的断言；不删除测试 |
| `logic-grpc-service/service/agent_context.go` | 改：内部使用新打分；保留旧函数签名 |
| `logic-grpc-service/service/agent_context_memory_test.go` | 必要时更新对 Memory `score` 的断言；不删除测试 |
| `logic-grpc-service/service/agent_skill_service.go` | 改：填充 proto 新字段；不破坏 `DebugSemanticRetrieval` 接口 |
| `logic-grpc-service/proto/recruitment.proto` | 改：仅追加字段 |
| `logic-grpc-service/recruitment/pb/recruitment.pb.go` | 由 `protoc` 重生成；TASK 范围内手动同步最小集合（如已重生成，直接采用） |
| `logic-grpc-service/recruitment/pb/recruitment_grpc.pb.go` | 不变 |
| `web-gin-service/recruitment/pb/recruitment.pb.go` | 由 `protoc` 重生成；TASK 范围内同步 |
| `web-gin-service/handler/hr/agent_skill.go` | 不变（透传新字段） |
| `web-gin-service/router/router.go` | 不变 |
| `hr-frontend/src/types/agentSkill.ts` | 改：追加 optional 字段 |
| `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` | 可选改动：展示新 breakdown（不在所有 TASK 中强制） |
| `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml` | 不变 |
| `deploy/**` / `docker/**` / `start-dev.sh` / `stop-dev.sh` | 不变 |
| `model/AIMemory.go` / `model/AgentSkill.go` / `repository/**` | 不变 |

## 14. 备选方案

### 14.1 学习式 reranker

- 用 LLM 对 Top K 候选重排。
- 缺点：引入在线 LLM 成本与延迟；不适合召回主路径；与 SPEC 强调的稳定性诉求冲突。
- 结论：本期不采用。

### 14.2 多阶段 BM25 + vector fusion

- 用 BM25 作为 lexical 层。
- 缺点：引入新依赖；当前 `tokenizeAgentSkillText` 已经是轻量 BM25 近似；切换成本与收益不匹配。
- 结论：本期不采用。

### 14.3 全部信号进入单一 Sigmoid 输出

- 把所有信号直接喂给一个轻量 Sigmoid 模型。
- 缺点：训练数据不足、需新增模型与推理依赖、超出 SPEC 范围。
- 结论：本期不采用。

### 14.4 保留裸加但记录更多日志

- 改 `Score` 计算不变，仅扩展日志。
- 缺点：根本问题（P-1 / P-2）未解决；debug 信号虽增加但排序行为未稳定。
- 结论：本期不采用。

## 15. 待确认问题

- proto 字段一次性追加 7/8 号 `pool_confidence` 还是分 `top1_score` + `top2_score` + `gap` 三个字段？建议后者更利于排障。
- Memory `confidence` 是否同样需要相关性 gating？SPEC 默认 yes；如产品要求不同，需要在本期 SPEC 调整。
- 权重 `w_v=0.6 / w_l=0.3 / w_m=0.1`、boost 上限 `1.5`、gate 阈值 `0.15`、gap 阈值 `0.10/0.03` 是否合理？建议作为起点，后续按 AB 数据调整。
- `selectedAgentSkill.Score int` 旧字段是否仍保留兼容映射？SPEC 暂定 yes，避免破坏旧调试消费方。

## 16. 开放问题

- 是否在 debug 视图新增"权重说明"卡片（不影响逻辑）？
- 是否在日志中加 `pool=skill|memory` 标签，便于运维筛选？
- 是否提供 `pool_confidence` 之外的 `top1_score` / `top2_score` 字段，供高级用户直接观察？
- 是否在文档（`docs/`）中新增"召回排序"章节，作为长期维护入口？
