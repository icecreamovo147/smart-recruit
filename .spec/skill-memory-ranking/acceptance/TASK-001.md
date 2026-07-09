# Acceptance - TASK-001

## TASK Summary

新增 `logic-grpc-service/service/skill_memory_ranking.go` 与对应单测，定义 `RankingSignals` / `RankConfidence` 数据结构、权重 / 阈值常量，以及 `computeRelevanceScore` / `computeBusinessBoost` / `computeFinalRankScore` / `ComputePoolConfidence` 工具函数。本 TASK 不修改任何业务代码或 proto。

## SPEC References

- SPEC §5 FR-001：统一打分数据模型。
- SPEC §5 FR-002：相关性 + 业务 boost 分离。
- SPEC §5 FR-005：Memory importance / confidence 相关性 gating。
- SPEC §5 FR-006：Top1 / Top2 gap 置信度。
- SPEC §6 性能 / 稳定性。
- SPEC §11 AC-001 / AC-009。

## SDD References

- SDD §3.1 概述。
- SDD §3.2 数据结构。
- SDD §3.3 权重 / 阈值常量。
- SDD §3.4 公式。
- SDD §3.7 分池与置信度。
- SDD §11.1 单元测试。

## Acceptance Criteria

- [ ] AC-001：`skill_memory_ranking.go` 定义 `RankingSignals` 与 `RankConfidence`，含必要字段。
- [ ] AC-002：权重 / 阈值常量集中在文件顶部，与 SDD §3.3 完全一致。
- [ ] AC-003：`computeRelevanceScore(vector, lexical, metadata) float64` 输出 ∈ [0, 1]，且纯函数。
- [ ] AC-004：`computeBusinessBoost(priority, importanceSignal, confidenceSignal, isSkill) float64` 输出 ∈ [1.0, 1.5]，且纯函数。
- [ ] AC-005：`computeFinalRankScore(relevance, boost) float64` = `relevance * boost`，且纯函数。
- [ ] AC-006：`ComputePoolConfidence([]RankingSignals) RankConfidence` 输出 4 种枚举之一。
- [ ] AC-007：单测覆盖 ≥ 6 种典型组合；`go test ./...` 通过。
- [ ] AC-008：未修改任何业务代码、proto、pb、配置、共享类型。
- [ ] AC-009：未引入新依赖、未修改 `package.json` / `pnpm-lock.yaml` / `pnpm-workspace.yaml`。

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过。
- [ ] `gofmt -l logic-grpc-service/service/skill_memory_ranking.go` 无输出。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/skill_memory_ranking.go` 与 `logic-grpc-service/service/skill_memory_ranking_test.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-001` 提示无越界。

## Manual Verification, if needed

- 阅读 `skill_memory_ranking.go` 顶部常量，与 SDD §3.3 逐项核对。
- 阅读 `skill_memory_ranking_test.go`，确认覆盖：纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值 / Memory gating / 4 种 confidence 枚举。
- 阅读 `computeBusinessBoost` 边界：`priority=0 / 50 / 100 / 1000`、`importance=0 / 0.5 / 1`、`confidence=0 / 0.5 / 1`、`isSkill` 与 `!isSkill`。

## Out-of-Scope

- 不得修改任何业务代码（`agent_skill_selector.go` / `agent_context.go` / `agent_skill_service.go`）。
- 不得修改 proto / pb / 配置 / 共享类型。
- 不得引入新依赖。
- 不得修改现有 `tokenizeAgentSkillText` / `scoreAgentSkillMatch` / `memoryBaseRecallScore` 等 helper。
