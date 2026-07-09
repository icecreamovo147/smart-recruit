# TASK Report - TASK-003

## 1. TASK ID

TASK-003 — 实现 Memory 池混合打分排序

## 2. Modified File List

- `logic-grpc-service/service/agent_context.go`（修改 `rankMemories` 实现 + 在 `AgentContextBuilder` 上新增 `lastMemoryRankings` 内部字段）
- `logic-grpc-service/service/skill_memory_ranking.go`（追加 Memory 部分）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 Memory 测试）

## 3. Change Summary by File

### `agent_context.go`

1. **扩展 `AgentContextBuilder` 结构体**（line 61）：在末尾新增一个内部字段
   ```go
   // lastMemoryRankings 记录最近一次 rankMemories 调用的完整打分 breakdown，
   // 仅供 DebugSemanticRetrieval 路径读取。生产请求不依赖此字段。
   lastMemoryRankings []RankedMemoryItem
   ```
2. **重写 `rankMemories`**（line 254）：
   - 删除旧 `rankedMemory` 内部类型 / `memoryBaseRecallScore` 内联叠加逻辑；
   - 改调 `RankMemoryCandidates`（新增于 `skill_memory_ranking.go`）；
   - 排序键：`final_rank_score desc, importance desc, id asc`；
   - 把 `RankedMemoryItem` 写入 `b.lastMemoryRankings`，供后续 debug API（TASK-006）读取；
   - 函数签名保持不变（`func (b *AgentContextBuilder) rankMemories(ctx, input, memories) []model.AIMemory`）。
3. **保留既有 helper 不修改**：`semanticMemoryScores` / `memoryBaseRecallScore` / `keywordMemoryScore` 全部保留原签名，作为 `scoreMemoryRankingSignals` 的内部 helper 复用。
4. **删除 import `sort`**：重构后 `rankMemories` 不再使用 `sort` 包。

### `skill_memory_ranking.go`（追加 Memory 部分）

- `scoreMemoryRankingSignals(memory, input, semanticScore, embeddingAvailable, queryTokenCount) RankingSignals`：
  - `vector_score` ∈ [0, 1]，来自 `semanticScores[id]`，embedding 不可用 / 越界 → 0；
  - `lexical_score` = `keywordMemoryScore(query, content) / nTokens`（clip 到 1.0）；
  - `metadata_score` 由 `normalizeMemoryMetadataScore` 给出（application=1.0 / job=0.7 / hr=0.4 / miss=0）；
  - `relevance_score` 由本文件纯函数计算；
  - `importanceSignal` / `confidenceSignal` 在 `relevance_score >= rankRelevanceGate` 时生效，否则为 0；
  - `business_boost` = 1 + 0.2·importance + 0.1·confidence，clip 到 [1.0, 1.5]；
  - `final_rank_score` = `relevance_score * business_boost`。
- `normalizeMemoryMetadataScore(memory, input) float64`：scope 命中映射。
- `RankedMemoryItem` 内部类型：携带 `Memory` + `Signals`。
- `RankMemoryCandidates(query, memories, input, semanticScores, embeddingAvailable) []RankedMemoryItem`：混合打分 → 过滤 `final == 0` → 稳定排序。
- `memoryRankScoreCompatMap(items) map[uint64]int`：返回旧 `Score int` 兼容映射（= round(final·100)），供后续 debug API 复用。

### `skill_memory_ranking_test.go`（追加）

- `TestScoreMemoryRankingSignalsAllModes`：4 个子用例（vector_high_with_semantic / lexical_only_fallback / hybrid / nan_vector_handled）
- `TestRankMemoryCandidatesAllModes`：4 个子用例（embedding_unavailable_lexical_metadata / application_memory_first_with_semantic / zero_score_candidates_dropped / sort_by_final_rank_score_desc）
- `TestMemoryImportanceGatedByRelevance_RankMemoryCandidates`：2 个子用例（low_relevance_gates_boost / high_relevance_uses_boost）
- `TestMemoryPoolFallbackToLexicalMetadata`：降级路径端到端验证

## 4. Scope Check Result

- 实际 TASK-003 修改的 business 文件：`logic-grpc-service/service/agent_context.go`、`logic-grpc-service/service/skill_memory_ranking.go`、`logic-grpc-service/service/skill_memory_ranking_test.go`，全部在 `task-scope.json` 的 `allowedFiles` 列表内。
- `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-003` 输出 `forbidden files were modified: agent_skill_selector.go` —— **这是误报**：该文件是 TASK-002 的修改，仍在 working tree 中未被 commit；脚本无法区分 per-TASK 变更。实际 TASK-003 未触碰该文件。
- 实际范围检查（人工 `git diff --name-only | grep -v .spec`）：
  - `agent_context.go`（本 TASK 改动）
  - `agent_skill_selector.go`（TASK-002 遗留，未在 TASK-003 触碰）
- `bash .spec/skill-memory-ranking/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-001 `RankingSignals` 字段 | ✅ Memory 候选自动填充全部字段 |
| FR-002 boost / relevance 公式 | ✅ `computeRelevanceScore` / `computeBusinessBoost` 直接复用 |
| FR-003 Memory 池独立排序 | ✅ `rankMemories` 仅在候选 Memory 内部按 `final_rank_score` 排序；与 Skill 池完全独立 |
| FR-004 降级路径 lexical + metadata | ✅ `embeddingAvailable=false` 时 `vector_score=0`，`mode=lexical_metadata` |
| FR-005 Memory importance / confidence 相关性 gating | ✅ `memoryImportanceSignal` / `memoryConfidenceSignal` 在 `relevance < rankRelevanceGate` 时为 0 |
| FR-007 Debug 字段 | ✅ Memory breakdown 字段已计算并暂存在 `b.lastMemoryRankings`，待 TASK-006 透传 |
| FR-008 日志可观测 | ✅ Memory 路径下 debug 级别日志保留旧 `[上下文预算]` 字段；新 breakdown 在 debug 路径内部使用 |
| FR-009 接口兼容 | ✅ `Build` / `retrieveMemories` / `rankMemories` 签名不变；`AgentContext` 字段不变；`model.AIMemory` 不变 |
| §11 AC-003 | ✅ 现有 `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 通过 |

## 6. SDD Comparison Result

- SDD §3.2 数据结构：`RankingSignals` / `RankConfidence` 共用，新增 `RankedMemoryItem` 仅内部使用。
- SDD §3.6 Memory 排序实现：vector/lexical/metadata 拆解、gating、score 公式与 SDD 完全一致。
- SDD §3.7 分池与置信度：Memory 池排序独立；`ComputePoolConfidence` 由 TASK-004 接入。
- SDD §8 兼容性：旧 `memoryBaseRecallScore` / `keywordMemoryScore` / `semanticMemoryScores` 全部保留为 helper，外部调用方行为不变。
- SDD §11.1 单元测试：5 个 Memory 相关测试用例 + 2 个 TestMemoryImportanceGated 子用例 + 1 个降级测试。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `rankMemories` 内部使用 `final_rank_score` 排序，gating 后 `importance / confidence` 才参与 boost | ✅ |
| AC-002 `maxMemories` + `maxMemoryChars` 双重截断保持；`TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 通过 | ✅ |
| AC-003 `keywordMemoryScore` / `memoryBaseRecallScore` / `semanticMemoryScores` 行为保持，不修改对外签名 | ✅ |
| AC-004 `TestRankMemoryCandidatesAllModes` 覆盖 4 种 mode | ✅ |
| AC-005 `TestMemoryImportanceGatedByRelevance` 验证 gating | ✅（命名为 `TestMemoryImportanceGatedByRelevance_RankMemoryCandidates` 避免与 TASK-001 的 helper 测试同名） |
| AC-006 `go test ./...` 通过 | ✅ |
| AC-007 未修改 `model.AIMemory` | ✅ |
| AC-008 未修改 `Build` / `retrieveMemories` / `rankMemories` 签名 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestScoreMemoryRankingSignals\|TestRankMemoryCandidates\|TestMemoryImportanceGated\|TestMemoryPoolFallback' ./service/...` | `ok 0.198s` |
| `go test ./...` | 全部 `ok` |
| `gofmt -l` 三个文件 | 无输出（已 gofmt） |
| `bash .spec/skill-memory-ranking/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck） |

## 9. Risks

- **`check-task-scope.sh` 误报**：脚本无法区分 per-TASK 变更；前序 TASK 的未 commit 改动会触发本 TASK 的 `forbidden` 误报。已在报告中标注；实际 TASK-003 修改的 business 文件全部在 `allowedFiles` 内。
- **`AgentContextBuilder.lastMemoryRankings` 内存保留**：debug 路径在并发场景下不严格 thread-safe（同一 builder 共享）。当前 `Build` 是单 goroutine 调用，可接受；后续若引入并发 debug API，需要加锁或在 debug 路径内一次性调用 `rankMemories` 并就地读取。
- **`memoryBaseRecallScore` 仍被引用**：`semanticDebugMemories` 仍使用 `memoryBaseRecallScore` 计算旧 `score` 字段值；TASK-006 应当改用 `memoryRankScoreCompatMap`。
- **测试命名冲突**：与 TASK-001 的 `TestMemoryImportanceGatedByRelevance` 同名 → 改为 `TestMemoryImportanceGatedByRelevance_RankMemoryCandidates`；helper 单元测试保留原名。

## 10. Follow-up Items

- TASK-004：把 `ComputePoolConfidence` 接入 Skill / Memory 池的 debug 输出；在 `agent_skill_service.go` 引入 internal struct 暂存 `pool_confidence`。
- TASK-006：在 `agent_skill_service.go` 的 `semanticDebugMemories` 中读取 `b.lastMemoryRankings`，填充 proto 新字段。
- 后续可考虑为 `b.lastMemoryRankings` 增加并发保护（如果 debug API 引入并发）。

## 11. Whether the Next TASK Can Start

✅ TASK-004 可以开始。`scoreMemoryRankingSignals` / `RankMemoryCandidates` / `b.lastMemoryRankings` / `ComputePoolConfidence` 已就位，TASK-004 可在 Skill 池的 `selectedAgentSkill` 与 Memory 池的 `lastMemoryRankings` 之上分别计算 `pool_confidence`。
