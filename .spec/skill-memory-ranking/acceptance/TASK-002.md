# Acceptance - TASK-002

## TASK Summary

让 `selectAgentSkillsWithSemantic` 自动排序阶段走混合打分。扩展 `selectedAgentSkill` 字段，保留 `selectAgentSkills` / `selectAgentSkillsWithSemantic` 函数签名。`Score int` 旧字段保留为 `int(round(final_rank_score * 100))` 兼容映射。

## SPEC References

- SPEC §5 FR-001 ~ FR-003（Skill 池）。
- SPEC §5 FR-007：Debug 字段。
- SPEC §5 FR-009：接口兼容。
- SPEC §5 FR-010：测试覆盖。
- SPEC §7 兼容性需求。
- SPEC §11 AC-002 / AC-009。

## SDD References

- SDD §3.5 Skill 排序实现。
- SDD §3.10 兼容映射。
- SDD §4 数据结构变化。
- SDD §5 API 与接口变化。
- SDD §8 兼容性策略。
- SDD §11.2 集成 / 表驱动测试。

## Acceptance Criteria

- [ ] AC-001：`selectAgentSkillsWithSemantic` 自动排序阶段使用 `final_rank_score` 排序；手动 Skill 路径行为不变。
- [ ] AC-002：`selectedAgentSkill` 扩展字段（`VectorScore / LexicalScore / MetadataScore / RelevanceScore / BusinessBoost / FinalRankScore / RelevanceMode / PoolRank / RankingConfidence`）被填充。
- [ ] AC-003：`Score int` 旧字段值 = `int(round(final_rank_score * 100))`。
- [ ] AC-004：`scoreAgentSkillMatch` 不被删除，可作为内部 helper 复用；`tokenizeAgentSkillText` / `addCJKAgentSkillTokens` 不被修改。
- [ ] AC-005：现有 `TestSelectAgentSkillsManualPriorityAndAutoMatch` / `TestSelectAgentSkillsUsesPriorityAndRequiredCapabilities` / `TestSelectAgentSkillsUsesSemanticScoresOnlyToRerankRuleCandidates` 通过；必要时更新对 `Score` 整数的强断言为 `FinalRankScore`。
- [ ] AC-006：新增 `TestRankSkillCandidatesAllModes` 覆盖 vector / lexical / 混合 / 降级 4 种 mode。
- [ ] AC-007：新增 `TestSkillPoolFallbackToLexicalMetadata` 验证降级路径。
- [ ] AC-008：`go test ./...` 通过。
- [ ] AC-009：未修改 `model.AgentSkill` / `model.AgentSkillVersion`、proto / pb、配置、共享类型。
- [ ] AC-010：未引入新依赖。

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过。
- [ ] `gofmt -l logic-grpc-service/service/agent_skill_selector.go` 无输出。
- [ ] `gofmt -l logic-grpc-service/service/skill_memory_ranking.go` 无输出。
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/agent_skill_selector.go` / `agent_skill_selector_test.go` / `skill_memory_ranking.go` / `skill_memory_ranking_test.go`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-002` 提示无越界。

## Manual Verification, if needed

- 阅读 `agent_skill_selector.go` 中 `selectAgentSkillsWithSemantic` 自动排序阶段，确认排序键为 `final_rank_score desc`。
- 阅读 `selectedAgentSkill` 字段定义，确认新字段全部为值类型。
- 阅读 `selectedAgentSkillTraceItems` 输出，确认 `pool_rank` / `ranking_confidence` 被填充。
- 跑 `TestRankSkillCandidatesAllModes` 4 个子用例，确认 4 种 mode 行为符合 SDD §3.5。

## Out-of-Scope

- 不得修改 `model.AgentSkill` / `model.AgentSkillVersion`。
- 不得修改 proto / pb。
- 不得修改 Memory 池（由 TASK-003 处理）。
- 不得修改 `DebugSemanticRetrieval`（由 TASK-006 处理）。
