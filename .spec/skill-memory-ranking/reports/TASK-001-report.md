# TASK Report - TASK-001

## 1. TASK ID

TASK-001 — 新增混合打分数据模型与权重常量

## 2. Modified File List

- `logic-grpc-service/service/skill_memory_ranking.go`（新增）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（新增）

## 3. Change Summary by File

### `skill_memory_ranking.go`（新增，~150 行）

- **权重与阈值常量**（文件顶部，与 SDD §3.3 严格一致）：
  - `rankWeightVector = 0.6`，`rankWeightLexical = 0.3`，`rankWeightMetadata = 0.1`
  - `rankBusinessBoostMax = 1.5`
  - `rankPriorityNorm = 50.0`
  - `rankBoostAlpha = 0.2`，`rankBoostBeta = 0.2`，`rankBoostGamma = 0.1`
  - `rankRelevanceGate = 0.15`
  - `rankGapHigh = 0.10`，`rankGapMedium = 0.03`
- **数据结构**：
  - `RankingSignals`（vector/lexical/metadata/relevance/boost/final/mode/reasons）
  - `RankConfidence` 字符串枚举：`high` / `medium` / `low` / `none`
- **纯函数工具**：
  - `clamp01(v float64) float64` — 把 NaN/Inf 视作 0，并裁剪到 [0,1]
  - `safeSignal(v float64) float64` — 等价于 `clamp01`
  - `computeRelevanceScore(vector, lexical, metadata) float64` — 加权和，输出 ∈ [0,1]
  - `computeBusinessBoost(priority, importanceSignal, confidenceSignal, isSkill) float64` — 仅 Skill 用 priority，Memory 用 importance/confidence；输出 ∈ [1.0, 1.5]
  - `computeFinalRankScore(relevance, boost) float64` — `relevance * boost`，输出 ∈ [0, 1.5]
  - `memoryImportanceSignal(relevance, importance) float64` / `memoryConfidenceSignal(relevance, confidence) float64` — Memory 相关性 gating 助手
  - `ComputePoolConfidence([]RankingSignals) RankConfidence` — 4 种枚举 + NaN 过滤 + 乱序输入排序

### `skill_memory_ranking_test.go`（新增，~200 行）

- 表驱动测试覆盖 5 个核心函数：
  - `TestComputeRelevanceScore`：10 个 case（全 0、纯 vector/lexical/metadata、混合、clamp 越界、NaN）
  - `TestComputeBusinessBoost`：10 个 case（Skill priority 0/50/100/1000/-10/NaN；Memory gated/active/out-of-range）
  - `TestComputeFinalRankScore`：4 个 case（零相关、满相关、混合、NaN boost）
  - `TestMemoryImportanceGatedByRelevance`：4 个 case（低于阈值/等于/高于/越界）
  - `TestComputePoolConfidence`：9 个 case（空/单元素低于门/单元素高于门/高/中/低 gap/强制 low/NaN 过滤/乱序输入）

## 4. Scope Check Result

- `git diff --name-only` 仅包含 `logic-grpc-service/service/skill_memory_ranking.go` 与 `logic-grpc-service/service/skill_memory_ranking_test.go`。
- `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-001` → `All changed files are within scope for TASK-001.`

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-001 `RankingSignals` 字段、归一化 | ✅ 全部字段、JSON tag 与 SPEC §5 一致 |
| FR-002 `relevance_score` / `business_boost` 公式与权重 | ✅ 权重常量 0.6/0.3/0.1、boost 上限 1.5、α/β/γ = 0.2/0.2/0.1 完全一致 |
| FR-005 Memory gating 阈值 0.15 | ✅ `rankRelevanceGate = 0.15` |
| FR-006 top1/top2 gap 阈值 0.10/0.03 | ✅ `rankGapHigh / rankGapMedium` |
| §6 性能 / 稳定性 | ✅ 所有函数为纯函数，无 IO、无全局状态 |
| §11 AC-001 / AC-009 | ✅ 单测覆盖 ≥ 6 种组合（实际 ≥ 30 个 case）；`go test ./...` 通过 |

## 6. SDD Comparison Result

- SDD §3.2 数据结构：✅ `RankingSignals` / `RankConfidence` 与 SDD 字段一一对应。
- SDD §3.3 权重 / 阈值常量：✅ 11 个常量全部 hardcode 在文件顶部，值与 SDD 完全一致。
- SDD §3.4 公式：✅ `relevance = clamp01(w_v*vec + w_l*lex + w_m*meta)`；`boost = clamp(1 + α*p_norm + β*imp + γ*conf, 1.0, 1.5)`；`final = relevance * boost`。
- SDD §3.7 分池与置信度：✅ 4 种枚举 + 强制 low + NaN 过滤。
- SDD §11.1 单元测试：✅ 5 个核心函数均有表驱动测试。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `RankingSignals` / `RankConfidence` 与必要字段 | ✅ |
| AC-002 权重 / 阈值常量与 SDD §3.3 一致 | ✅ |
| AC-003 `computeRelevanceScore` 纯函数且 ∈ [0,1] | ✅ |
| AC-004 `computeBusinessBoost` 纯函数且 ∈ [1.0, 1.5] | ✅ |
| AC-005 `computeFinalRankScore = relevance * boost` 纯函数 | ✅ |
| AC-006 `ComputePoolConfidence` 输出 4 种枚举 | ✅ |
| AC-007 单测覆盖 ≥ 6 种典型组合；`go test ./...` 通过 | ✅（实际 30+ case） |
| AC-008 未修改任何业务代码 / proto / pb / 配置 / 共享类型 | ✅ |
| AC-009 未引入新依赖，未修改 `package.json` | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestComputeRelevanceScore\|TestComputeBusinessBoost\|TestComputeFinalRankScore\|TestMemoryImportanceGatedByRelevance\|TestComputePoolConfidence' ./service/...` | `ok logic-grpc-service/service 0.017s` |
| `go test ./...` | 全部包 `ok`，无回归 |
| `go build ./...` | 无输出（成功） |
| `gofmt -l logic-grpc-service/service/skill_memory_ranking.go logic-grpc-service/service/skill_memory_ranking_test.go` | 无输出（已 gofmt） |
| `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-001` | `All changed files are within scope for TASK-001.` |
| `bash .spec/skill-memory-ranking/scripts/agent-check.sh` | 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck） |

## 9. Risks

- 无新依赖、无公共 API 变化、无 proto 改动。
- 唯一潜在风险：后续 TASK 在 `agent_skill_selector.go` / `agent_context.go` 中调用本文件的 helper 时需要保持函数导出与参数一致；当前 `clamp01` / `safeSignal` 未导出，但实现侧使用同一份逻辑，TASK-002/003 不需要它们。
- `memoryImportanceSignal` / `memoryConfidenceSignal` 在 TASK-003 中会被 `scoreMemoryRankingSignals` 复用，本 TASK 已为它准备好 helper。

## 10. Follow-up Items

- TASK-002：在 `skill_memory_ranking.go` 追加 `scoreSkillRankingSignals` 与 `RankSkillCandidates`，并改造 `selectAgentSkillsWithSemantic`。
- TASK-003：追加 `scoreMemoryRankingSignals` 与 `RankMemoryCandidates`，改造 `rankMemories`。
- 权重与阈值均为常量；若需热调，需走 `config.Ranking` 段接入（不在本期范围）。

## 11. Whether the Next TASK Can Start

✅ TASK-002 可以开始。`RankingSignals` / `RankConfidence` / 权重常量 / 三个核心打分函数已就位，TASK-002 可直接基于本 TASK 实现的 helper 继续。
