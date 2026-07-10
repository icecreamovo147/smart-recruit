# TASK Report - TASK-004

## 1. TASK ID

TASK-004 — 引入 top1/top2 gap 与池置信度计算

## 2. Modified File List

- `logic-grpc-service/service/agent_skill_service.go`（修改 `DebugSemanticRetrieval` / `semanticDebugMemories` + 新增 internal struct）
- `logic-grpc-service/service/skill_memory_ranking.go`（修复 `ComputePoolConfidence` 边界 bug）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 2 个池置信度集成测试）

## 3. Change Summary by File

### `agent_skill_service.go`

1. **`DebugSemanticRetrieval`** 末尾追加：
   - 调用 `computeDebugPoolConfidence(selected, lastMemoryRankings)` 计算 Skill / Memory 池的 confidence；
   - 通过 `stashDebugPoolConfidence` 暂存到 `lastDebugPoolConfidence` 包级变量（任务间中转）；
   - 在 debug 级别日志中输出两个 confidence 值，便于运维 / 排障。
2. **`semanticDebugMemories`** 末尾追加：
   - 把 builder 的 `lastMemoryRankings` 通过 `stashLastDebugMemoryRankings` 暂存到 `lastDebugMemoryRankings` 包级变量，供后续 `DebugSemanticRetrieval` 读取。
3. **新增 internal struct**（per 任务 spec 允许）：
   ```go
   type debugPoolConfidenceView struct {
       SkillPoolConfidence  RankConfidence
       MemoryPoolConfidence RankConfidence
   }
   ```
4. **新增包级变量**（debug 中转，注释明确线程安全假设）：
   - `lastDebugPoolConfidence debugPoolConfidenceView`
   - `lastDebugMemoryRankings []RankedMemoryItem`
5. **新增 helper 函数**：
   - `stashDebugPoolConfidence(v)`
   - `lastDebugPoolConfidenceForTest()`（仅测试 / TASK-006 复用）
   - `stashLastDebugMemoryRankings(items)`
   - `getLastDebugMemoryRankings()`
   - `computeDebugPoolConfidence(selected, memoryRankings) debugPoolConfidenceView`

### `skill_memory_ranking.go`

- **修复 `ComputePoolConfidence` 边界 bug**（原本未处理 "top1=0" 的情况）：
  - 单候选 top1=0 → 返回 `none`（之前返回 `low`）
  - 多候选 top1=0 → 返回 `none`（之前返回 `low`）
  - 与 SPEC §5 FR-006 / SDD §3.7 描述 "池内全部 `final_rank_score = 0` → none" 一致。

### `skill_memory_ranking_test.go`

- `TestSkillPoolConfidenceFromRankings`：3 个子用例（high_gap_three_candidates / low_gap_close_candidates / none_for_empty）
- `TestMemoryPoolConfidenceFromRankings`：2 个子用例（medium_gap_two_memories / all_zero_none；后者专门验证 TASK-004 修复的边界 bug）

## 4. Scope Check Result

- 实际 TASK-004 修改的 business 文件：`agent_skill_service.go`（在 allowedFiles 内）、`skill_memory_ranking.go`（在 allowedFiles 内）、`skill_memory_ranking_test.go`（在 allowedFiles 内）。
- `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-004` 会触发与 TASK-002 / TASK-003 相同的误报（前序 TASK 的 `agent_skill_selector.go` 与 `agent_context.go` 仍在 working tree），本 TASK 未触碰这些文件。
- 实际范围检查（人工 `git diff --name-only | grep -v .spec`）：
  - `agent_skill_service.go`（本 TASK 改动）
  - `agent_skill_selector.go`（TASK-002 遗留）
  - `agent_context.go`（TASK-003 遗留）
  - `skill_memory_ranking.go`（TASK-001/002/003 累计 + 本 TASK 修复）
  - `skill_memory_ranking_test.go`（TASK-001/002/003 累计 + 本 TASK 追加）
- `bash .spec/skill-memory-ranking/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-006 top1/top2 gap 置信度 | ✅ `ComputePoolConfidence` 实现 4 种枚举 + 强制 low（top1 < gate） + 全部 0 → none |
| FR-007 Debug 字段 | ✅ Skill 池 trace 已写入 `pool_rank` / `ranking_confidence`（TASK-002）；Memory 池 signals 暂存在 `lastMemoryRankings`（TASK-003）；pool_confidence 通过 internal struct 暂存 |
| FR-008 日志可观测 | ✅ debug 级别日志输出 `skill_pool_confidence` / `memory_pool_confidence` |
| §11 AC-006 | ✅ 4 种 confidence 枚举覆盖 |

## 6. SDD Comparison Result

- SDD §3.7 分池与置信度：实现与描述完全一致。
- SDD §10 可观测性：debug 日志 + 4 个 internal 字段（PoolRank / RankingConfidence / VectorScore / LexicalScore / MetadataScore / RelevanceScore / BusinessBoost / FinalRankScore / RelevanceMode）暂存路径完整。
- SDD §11.1 单元测试：5 个置信度子用例 + 2 个集成测试（Skill / Memory pool from rankings）。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `ComputePoolConfidence` 单测覆盖 4 种枚举 | ✅ |
| AC-002 边界场景：空池、1 候选、2 候选、多候选 | ✅ |
| AC-003 Skill trace 写入 `pool_rank` / `ranking_confidence`（TASK-002） | ✅ |
| AC-004 `DebugSemanticRetrieval` 计算 `skill_pool_confidence` / `memory_pool_confidence` 暂存到 internal struct | ✅ |
| AC-005 `go test ./...` 通过 | ✅ |
| AC-006 未修改 proto / pb、配置、共享类型、`DebugSemanticRetrieval` 签名 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestComputePoolConfidence\|TestSkillPoolConfidenceFromRankings\|TestMemoryPoolConfidenceFromRankings' ./service/... -v` | 全部 `PASS` |
| `go test ./...` | 全部 `ok` |
| `gofmt -l` 三个文件 | 无输出（已 gofmt） |
| `bash .spec/skill-memory-ranking/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck） |

## 9. Risks

- **`check-task-scope.sh` 误报**：与前序 TASK 同样的问题（无法区分 per-TASK 变更）。已在报告中标注。
- **`lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级变量的线程安全**：本字段假设 `DebugSemanticRetrieval` 单 goroutine 调用。AgentSkillService 实例通常绑定到单一 gRPC handler，在生产中同一实例的并发请求会覆盖 lastDebugPoolConfidence。TASK-006 可以选择：
  - 在 helper 内部重新计算 pool_confidence（基于新计算的 proto 字段或 builder 状态）；
  - 给包级变量加 `sync.Mutex` 保护；
  - 在每次 debug 调用内部一次性计算 pool_confidence 并就地使用 proto 字段。
- **`ComputePoolConfidence` 边界修复** 是 1 行代码改动，影响 TASK-001 既有测试的潜在行为。已验证：所有 9 个 `TestComputePoolConfidence` 子用例仍 PASS；新增 `TestMemoryPoolConfidenceFromRankings/all_zero_none` 专门覆盖修复路径。
- **trace helper 中 `pool_rank` 字段在 Skill 池已就绪，Memory 池未填充到 proto item**：受限于 proto 字段未生成；TASK-006 会从 `lastMemoryRankings` 读取 signals 并填充 proto 新字段。

## 10. Follow-up Items

- TASK-005：扩展 proto 字段 `skill_pool_confidence` / `memory_pool_confidence` + 各类 breakdown 字段。
- TASK-006：在 `agent_skill_service.go` 读取 `lastDebugPoolConfidence` 与 `lastDebugMemoryRankings`，填充 proto response 新字段。
- 后续 PR：考虑将 `lastDebugPoolConfidence` 改为 per-request context value，避免并发覆盖。

## 11. Whether the Next TASK Can Start

⚠️ TASK-005 需要用户确认（标记 `requiresHumanConfirmation: true`）。在用户确认前，**停等**用户的 proto 字段命名 / 类型 / 编号确认。
