# Acceptance - TASK-003

## TASK Summary

让 `AgentContextBuilder.rankMemories` 内部走混合打分。`importance` / `confidence` 在相关性 gating 阈值之上才参与 boost。保留 `Build` / `retrieveMemories` / `rankMemories` 函数签名。`maxMemories` + `maxMemoryChars` 双重截断保持。

## SPEC References

- SPEC §5 FR-001 ~ FR-003（Memory 池）。
- SPEC §5 FR-005：Memory importance / confidence 相关性 gating。
- SPEC §5 FR-007：Debug 字段。
- SPEC §5 FR-009：接口兼容。
- SPEC §5 FR-010：测试覆盖。
- SPEC §11 AC-003 / AC-005 / AC-009。

## SDD References

- SDD §3.6 Memory 排序实现。
- SDD §3.10 兼容映射。
- SDD §11.1 单元测试。
- SDD §11.2 集成 / 表驱动测试。

## Acceptance Criteria

- [ ] AC-001：`rankMemories` 内部使用 `final_rank_score` 排序；gating 后 `importance / confidence` 才参与 boost。
- [ ] AC-002：`maxMemories` + `maxMemoryChars` 双重截断保持。
- [ ] AC-003：现有 `keywordMemoryScore` / `memoryBaseRecallScore` / `semanticMemoryScores` 行为保持；可作为内部 helper 复用。
- [ ] AC-004：现有 `TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry` 通过；必要时更新对 `score` 的强断言。
- [ ] AC-005：新增 `TestRankMemoryCandidatesAllModes` 覆盖 vector / lexical / 混合 / 降级 4 种 mode。
- [ ] AC-006：新增 `TestMemoryImportanceGatedByRelevance` 验证 gating 行为。
- [ ] AC-007：`go test ./...` 通过。
- [ ] AC-008：未修改 `model.AIMemory`、proto / pb、配置、共享类型。
- [ ] AC-009：未引入新依赖。

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过。
- [ ] `gofmt -l logic-grpc-service/service/agent_context.go` 无输出。
- [ ] `gofmt -l logic-grpc-service/service/skill_memory_ranking.go` 无输出。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/agent_context.go` / `agent_context_memory_test.go` / `skill_memory_ranking.go` / `skill_memory_ranking_test.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-003` 提示无越界。

## Manual Verification, if needed

- 阅读 `agent_context.go` 中 `rankMemories`，确认排序键为 `final_rank_score desc, importance desc, id asc`。
- 阅读 `scoreMemoryRankingSignals`，确认 gating 在 `relevance_score < rankRelevanceGate` 时信号为 0。
- 跑 `TestMemoryImportanceGatedByRelevance` 至少 3 个子用例：完全 gating / 临界值 / 完全放行。
- 跑 `TestRankMemoryCandidatesAllModes` 4 个子用例，确认 4 种 mode 行为符合 SDD §3.6。

## Out-of-Scope

- 不得修改 `model.AIMemory`。
- 不得修改 proto / pb。
- 不得修改 Skill 池（由 TASK-002 处理）。
- 不得修改 `DebugSemanticRetrieval`（由 TASK-006 处理）。
- 不得修改 `Build` / `retrieveMemories` / `rankMemories` 函数签名。
