# Acceptance - TASK-004

## TASK Summary

在 `skill_memory_ranking.go` 提供 `ComputePoolConfidence`，接入 Skill / Memory 池的 debug 输出。在 `agent_skill_service.go` / `agent_skill_selector.go` / `agent_context.go` 内部暂存 `pool_confidence` / `pool_rank` / `ranking_confidence` 字段。本 TASK 不修改 proto，由 TASK-005 / TASK-006 完成 proto 与透传。

## SPEC References

- SPEC §5 FR-006：Top1 / Top2 gap 置信度。
- SPEC §5 FR-007：Debug 字段。
- SPEC §5 FR-008：日志可观测。
- SPEC §11 AC-006。

## SDD References

- SDD §3.7 分池与置信度。
- SDD §10 可观测性设计。
- SDD §11.1 单元测试。

## Acceptance Criteria

- [ ] AC-001：`ComputePoolConfidence` 单测覆盖 4 种枚举（high / medium / low / none）。
- [ ] AC-002：边界场景：空池（none）、1 候选（low / none）、2 候选（gap 计算）、多候选（high / medium / low）。
- [ ] AC-003：`selectedAgentSkillTraceItems` / Memory 调试 trace 输出新字段（`pool_rank` / `ranking_confidence`）。
- [ ] AC-004：`DebugSemanticRetrieval` 内部计算 `skill_pool_confidence` / `memory_pool_confidence`，暂存到 internal struct。
- [ ] AC-005：`go test ./...` 通过。
- [ ] AC-006：未修改 proto / pb、配置、共享类型、`DebugSemanticRetrieval` 函数签名。

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过。
- [ ] `gofmt -l logic-grpc-service/service/agent_skill_service.go` 无输出。
- [ ] `gofmt -l logic-grpc-service/service/skill_memory_ranking.go` 无输出。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/agent_skill_service.go` / `agent_skill_selector.go` / `agent_context.go` / `skill_memory_ranking.go` / `skill_memory_ranking_test.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-004` 提示无越界。

## Manual Verification, if needed

- 阅读 `ComputePoolConfidence`，确认 4 种枚举的判定逻辑与 SDD §3.7 一致。
- 阅读 `DebugSemanticRetrieval`，确认 `skill_pool_confidence` / `memory_pool_confidence` 已计算并暂存（即使 proto 字段未生成）。
- 跑 `TestComputePoolConfidence` 所有子用例，确认行为正确。
- 跑 `TestSkillPoolConfidenceFromRankings` / `TestMemoryPoolConfidenceFromRankings`，确认从 `[]RankingSignals` 推断 confidence。

## Out-of-Scope

- 不得修改 proto / pb。
- 不得修改 `DebugSemanticRetrieval` 函数签名。
- 不得修改 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` 现有字段。
- 不得在 debug 视图引入新 UI。
