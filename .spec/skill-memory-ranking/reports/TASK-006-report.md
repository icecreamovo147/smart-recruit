# TASK Report - TASK-006

## 1. TASK ID

TASK-006 — DebugSemanticRetrieval 填充 score breakdown 与 pool_confidence

## 2. Modified File List

- `logic-grpc-service/service/agent_skill_service.go`（修改 `DebugSemanticRetrieval` / `semanticDebugMemories` / `semanticDebugSkillsToPB` + 新增 `memoryPoolRankOf` helper）
- `logic-grpc-service/service/agent_skill_service_test.go`（追加 2 个 breakdown 测试）

## 3. Change Summary by File

### `agent_skill_service.go`

1. **`semanticDebugSkillsToPB`**：
   - 追加新字段填充：`VectorScore` / `LexicalScore` / `MetadataScore` / `RelevanceScore` / `BusinessBoost` / `FinalRankScore` / `RelevanceMode` / `PoolRank`。
   - 旧 `Score` 字段值改为 `skill.FinalRankScore`（SDD §3.10 兼容映射：旧 `score` = `final_rank_score`）。
2. **`semanticDebugMemories`**：
   - 改用 `b.lastMemoryRankings`（已由 TASK-003 注入）构造 `signalsByID` 映射；
   - 追加新字段填充（与 Skill 对称）；
   - 旧 `Score` 字段值改为 `signals.FinalRankScore`；
   - `PoolRank` 由新增 `memoryPoolRankOf` helper 计算（1-based 排名）。
3. **`memoryPoolRankOf(rankings, id) int` 新增 helper**：在 `rankings` 中按位置查找 memory ID，返回 1-based 排名；未找到返回 0。
4. **`DebugSemanticRetrieval`**：
   - `SkillPoolConfidence` / `MemoryPoolConfidence` 从 `poolView`（`computeDebugPoolConfidence` 产物）写入 proto response；
   - debug 日志同时输出 `skill_pool_confidence` / `memory_pool_confidence`。
5. **删除冗余旧代码**：`semanticDebugMemories` 内 `score := memoryBaseRecallScore(memory, input) + semanticScore*100` 不再写入 proto `Score`；保留 `reason` 字段的 "semantic similarity and scope/importance" / "scope/importance fallback" 切换用于诊断。

### `agent_skill_service_test.go`

- `TestAgentSkillServiceDebugSemanticRetrievalPopulatesScoreBreakdown`：使用 fake embedding provider 验证：
  - `Score == FinalRankScore`（兼容映射）
  - breakdown 字段（vector / lexical / metadata / relevance / business_boost / final_rank_score / relevance_mode / pool_rank）非 0 / 非空
  - `business_boost ∈ [1.0, 1.5]`
  - top-level `SkillPoolConfidence` / `MemoryPoolConfidence` 已填充
- `TestAgentSkillServiceDebugSemanticRetrievalFallbackPopulatesBreakdown`：embedding 不可用时验证：
  - `vector_score == 0`
  - `relevance_mode == "lexical_metadata"`
  - `business_boost ∈ [1.0, 1.5]`
  - `SkillPoolConfidence` 仍被填充（fallback 行为可观测）

## 4. Scope Check Result

- 实际 TASK-006 修改的 business 文件：`agent_skill_service.go` / `agent_skill_service_test.go`，均在 `task-scope.json` 的 `allowedFiles` 列表内。
- `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-006` 同样会触发前序 TASK 的误报；本 TASK 未触碰 `agent_skill_selector.go` 或 `agent_context.go` 的业务代码。
- 实际范围检查（人工 `git diff --name-only | grep -v .spec`）：
  - `agent_skill_service.go`（本 TASK 改动）
  - `agent_skill_service_test.go`（本 TASK 改动）
  - `agent_skill_selector.go`（TASK-002 遗留）
  - `agent_context.go`（TASK-003 遗留）
  - `recruitment.pb.go` × 2（TASK-005 遗留）
- `bash .spec/skill-memory-ranking/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-007 Debug 字段 | ✅ Skill / Memory item 均填充全部新字段；response 顶层填充 `skill_pool_confidence` / `memory_pool_confidence` |
| FR-008 日志可观测 | ✅ 原有日志字段（embedding_available / fallback_reason 等）保持；新增 debug 级别输出 pool_confidence |
| FR-009 接口兼容 | ✅ `score` 字段值 = `final_rank_score`；旧字段类型不变；proto 仅追加新字段 |
| §5 AC-007 | ✅ 新增字段全部填充 |
| §11 AC-002 | ✅ 现有 `TestAgentSkillServiceDebugSemanticRetrievalFallback` / `...EmbeddingAvailableWhenServiceHealthy` 不回归 |

## 6. SDD Comparison Result

- SDD §3.5 Skill 排序实现：trace items 通过 `selectedAgentSkill` → `pb.SemanticSkillDebugItem` 完整映射（含 pool_rank）。
- SDD §3.6 Memory 排序实现：trace items 通过 `lastDebugMemoryRankings` → `pb.SemanticMemoryDebugItem` 完整映射（含 pool_rank via `memoryPoolRankOf`）。
- SDD §3.8 Debug 字段：proto 新字段 1:1 填充；类型 / 编号严格匹配。
- SDD §3.10 兼容映射：proto `score` 字段值 = `final_rank_score`；本 TASK 修改 `Score` 字段从 `float64(skill.Score)` 改为 `skill.FinalRankScore`，与 SDD 一致。
- SDD §8 兼容性策略：旧字段类型 / 语义保持；新增字段为 optional。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 Skill item 携带新字段 | ✅ |
| AC-002 Memory item 携带新字段 | ✅ |
| AC-003 response 顶层携带 `skill_pool_confidence` / `memory_pool_confidence` | ✅ |
| AC-004 现有 `embedding_available` / `fallback_reason` 保持 | ✅ |
| AC-005 `Score == FinalRankScore` | ✅ |
| AC-006 降级路径 `relevance_mode = "lexical_metadata"` 且 `vector_score = 0` | ✅（新增专门测试） |
| AC-007 `go test ./...` 通过 | ✅ |
| AC-008 未修改 `DebugSemanticRetrieval` 签名 / 现有 proto 字段 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestAgentSkillServiceDebugSemanticRetrieval' ./service/...` | `ok 0.194s` |
| `go test ./...` | 全部 `ok` |
| `gofmt -l` 两个文件 | 无输出（已 gofmt） |
| `bash .spec/skill-memory-ranking/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck） |

## 9. Risks

- **`Score` 字段值变化**：proto `Score` 字段从 `float64(skill.Score) = final_rank_score * 100` 改为 `skill.FinalRankScore`。前数值范围 0..150；新数值范围 0..1.5。前端如果按整数阈值（例如 `score > 100`）断言会受影响。前端类型更新由 TASK-007 处理。
- **`memoryPoolRankOf` 性能**：当前实现是 O(n) 线性扫描；`n = limit`（最多 20）。生产可接受。如未来 `limit` 上调，可考虑排序索引。
- **`lastDebugPoolConfidence` / `lastDebugMemoryRankings` 并发安全**：TASK-004 已记录；AgentSkillService 假设单 goroutine 调用 `DebugSemanticRetrieval`。
- **`semanticDebugMemories` 中 `score` 变量定义已删除**：旧 `score := memoryBaseRecallScore(...)` 不再使用，理由字段仍按 semantic / fallback 切换保留（用于诊断 `reason`）。原 `score` 的整数输出语义不再适用，所有数值现在由 hybrid 打分提供。

## 10. Follow-up Items

- TASK-007：前端 `agentSkill.ts` 同步追加 optional 字段。
- TASK-008：补充表驱动测试覆盖纯 vector / 纯 lexical / 混合 / 降级 4 种 mode。
- 后续 PR：考虑为 `lastDebugPoolConfidence` 加 `sync.Mutex` 或 per-request context value。

## 11. Whether the Next TASK Can Start

✅ TASK-007 可以开始。proto 字段已就位、gRPC response 填充完成；前端 `agentSkill.ts` 可直接消费新字段。
