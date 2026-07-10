# Acceptance - TASK-008

## TASK Summary

新增 / 更新单测与表驱动测试，覆盖混合打分、降级路径、置信度、gating、分池隔离等关键路径。保证 `go test ./...` 与 `pnpm --filter hr-frontend typecheck` 全部通过。

## SPEC References

- SPEC §5 FR-010：测试覆盖。
- SPEC §11 AC-001 ~ AC-006 / AC-009 / AC-010。

## SDD References

- SDD §11 测试策略。
- SDD §8 兼容性策略。

## Acceptance Criteria

- [ ] AC-001：新增测试覆盖 SPEC §5 FR-010 列出的全部场景。
- [ ] AC-002：现有 `go test ./...` 全部通过。
- [ ] AC-003：现有 `pnpm --filter hr-frontend typecheck` 通过。
- [ ] AC-004：不删除已有测试；不弱化已有断言；必要时更新对 `Score` 的强断言到 `FinalRankScore`。
- [ ] AC-005：测试不依赖真实 embedding provider。
- [ ] AC-006：测试覆盖以下用例：
  - 纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值
  - Memory gating 在阈值前不生效
  - top1/top2 gap 高 / 中 / 低 / 无
  - Skill / Memory 池互不污染
  - Embedding 降级路径

## Required Tests

- [ ] `TestComputeRelevanceScore`
- [ ] `TestComputeBusinessBoost`
- [ ] `TestComputeFinalRankScore`
- [ ] `TestMemoryImportanceGatedByRelevance`
- [ ] `TestComputePoolConfidence`
- [ ] `TestRankSkillCandidatesAllModes`
- [ ] `TestRankMemoryCandidatesAllModes`
- [ ] `TestSkillPoolFallbackToLexicalMetadata`
- [ ] `TestMemoryPoolFallbackToLexicalMetadata`
- [ ] `TestPoolIsolation`

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过。
- [ ] `pnpm --filter hr-frontend typecheck` 通过。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/*_test.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-008` 提示无越界。

## Manual Verification, if needed

- 阅读 `skill_memory_ranking_test.go`，确认覆盖：纯 vector / 纯 lexical / 纯 metadata / 混合 / 全 0 / 极端值 / Memory gating / 4 种 confidence 枚举。
- 阅读 `agent_skill_selector_test.go`，确认现有测试仍通过；如更新断言，确认逻辑等价。
- 阅读 `agent_context_memory_test.go`，确认 Memory 排序期望与新模型一致。
- 跑 `go test -run TestRankSkillCandidatesAllModes ./...`，确认 4 种 mode 行为正确。
- 跑 `go test -run TestMemoryImportanceGatedByRelevance ./...`，确认 gating 行为正确。
- 跑 `go test -run TestPoolIsolation ./...`，确认 Skill / Memory 池互不污染。

## Out-of-Scope

- 不得删除已有测试。
- 不得弱化已有断言（如必须修改，需在报告中说明）。
- 不得使用真实 embedding provider。
- 不得使用 `any` 绕过类型检查。
- 不得修改业务代码（`*.go`，除 `*_test.go`）。
