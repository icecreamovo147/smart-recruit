# TASK Report - TASK-FU-007

## 1. TASK ID

TASK-FU-007 — RankingConfig 收敛为单个 struct

## 2. Modified File List

- `logic-grpc-service/service/skill_memory_ranking.go`（11 个 `var` 替换为 1 个 `rankingConfig` struct + 47 处引用更新）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（67 处引用更新）

## 3. Change Summary by File

### `skill_memory_ranking.go`

1. **11 个 `var` 替换为 1 个 `rankingConfig` struct**：
   ```go
   var (
       rankingConfig = struct {
           WeightVector     float64
           WeightLexical    float64
           WeightMetadata   float64
           BusinessBoostMax float64
           PriorityNorm     float64
           BoostAlpha       float64
           BoostBeta        float64
           BoostGamma       float64
           RelevanceGate    float64
           GapHigh          float64
           GapMedium        float64
       }{
           WeightVector:     rankingDefaults.WeightVector,
           // ... 11 fields
       }
   )
   ```
2. **47 处引用更新**（sed 批量替换）：
   - `rankWeightVector` → `rankingConfig.WeightVector`
   - `rankWeightLexical` → `rankingConfig.WeightLexical`
   - `rankWeightMetadata` → `rankingConfig.WeightMetadata`
   - `rankBusinessBoostMax` → `rankingConfig.BusinessBoostMax`
   - `rankPriorityNorm` → `rankingConfig.PriorityNorm`
   - `rankBoostAlpha` → `rankingConfig.BoostAlpha`
   - `rankBoostBeta` → `rankingConfig.BoostBeta`
   - `rankBoostGamma` → `rankingConfig.BoostGamma`
   - `rankRelevanceGate` → `rankingConfig.RelevanceGate`
   - `rankGapHigh` → `rankingConfig.GapHigh`
   - `rankGapMedium` → `rankingConfig.GapMedium`
3. `LoadRankingConfig` / `ResetRankingConfigForTest` 签名不变（仅操作 `rankingConfig` struct）。

### `skill_memory_ranking_test.go`

- 67 处引用同步更新（同上 11 个 sed 规则）。
- 现有测试断言（如 `if rankingConfig.WeightVector != ...`）保留并通过。

## 4. Scope Check Result

- 实际修改文件均在 `task-scope.json` 的 `allowedFiles` 内。
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过。
- `go test ./...` (logic-grpc-service) → 全部 `ok`（11 个包）。
- `grep -E "^\s*rank[A-Z]\w*\s*=" skill_memory_ranking.go` → 0 行（var 定义已删除）。
- `grep -c "rankingConfig\." skill_memory_ranking.go` → 47；`skill_memory_ranking_test.go` → 67。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-007 收敛 | ✅ 11 个 var → 1 个 struct |
| §11 AC-FU-007 | ✅ 全部 5 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.3 RankingConfig 收敛：完全对齐。
- SDD §8 兼容性策略：内部 refactor，不影响外部 API。
- SDD §11 测试策略：所有现有测试无回归。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 11 个 `var rankXxx` 替换为 `var rankingConfig` struct | ✅ |
| AC-002 所有 `rankXxx` 引用改为 `rankingConfig.Xxx` | ✅（47 + 67 = 114 处） |
| AC-003 `LoadRankingConfig` / `ResetRankingConfigForTest` 签名不变 | ✅ |
| AC-004 所有现有测试通过 | ✅ |
| AC-005 `go test ./...` 通过 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test ./...` (logic-grpc-service) | 全部 `ok` |
| `gofmt -l` 两个文件 | 无输出（已 gofmt） |
| `grep -E "^\s*rank[A-Z]\w*\s*=" skill_memory_ranking.go` | 0 行 |
| `grep -c "rankingConfig\."` | 47 + 67 = 114 |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. Risks

- **大批量 rename**：用 `sed` 批量替换；114 处引用一次性更新；如未来需要回滚，需还原 11 个 var 声明 + 114 处引用。
- **未来测试新增 var 引用**：如开发者新增测试并使用 `rankXxx`，编译会失败（无该符号）；应改用 `rankingConfig.Xxx`。
- **行内 `var` 声明 vs 现有 `var (...)` 块**：原 11 个 `var` 在一个块内，重构后仍是 1 个 `var` 块；diff 集中在 struct 定义与 47 处引用。

## 10. Follow-up Items

- TASK-FU-008：ranking-show CLI（依赖本 TASK 的 `rankingConfig` struct）。
- 后续 PR：可考虑把 `rankingConfig` 暴露为 `func Ranking() RankingSnapshot` getter，避免外部直接修改 struct。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-008 可以开始。`rankingConfig` struct 已就绪，CLI 可直接读 11 个字段。
