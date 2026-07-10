# Acceptance - TASK-007

## TASK Summary

在 `hr-frontend/src/types/agentSkill.ts` 末尾追加 `SemanticSkillDebugItem` / `SemanticMemoryDebugItem` / `SemanticRetrievalDebugResult` 的 optional 字段。保持现有 `score` 字段类型与语义不变。

## SPEC References

- SPEC §5 FR-007：Debug 字段扩展。
- SPEC §5 FR-009：接口兼容。
- SPEC §7 兼容性需求。
- SPEC §11 AC-008。

## SDD References

- SDD §3.8 Debug 字段。
- SDD §4 数据结构变化。
- SDD §8 兼容性策略。

## Acceptance Criteria

- [ ] AC-001：`SemanticSkillDebugItem` 追加 8 个 optional 字段（`vector_score` / `lexical_score` / `metadata_score` / `relevance_score` / `business_boost` / `final_rank_score` / `relevance_mode` / `pool_rank`）。
- [ ] AC-002：`SemanticMemoryDebugItem` 追加对应字段。
- [ ] AC-003：`SemanticRetrievalDebugResult` 追加 `skill_pool_confidence` / `memory_pool_confidence`。
- [ ] AC-004：旧 `score` 字段类型与语义不变。
- [ ] AC-005：`pnpm --filter hr-frontend typecheck` 通过。
- [ ] AC-006：未删除 / 未修改现有字段。

## Required Checks

- [ ] `pnpm --filter hr-frontend typecheck` 通过。
- [ ] `git diff --name-only` 仅包含 `hr-frontend/src/types/agentSkill.ts`。
- [ ] `bash .spec/skill-memory-ranking/scripts/check-task-scope.sh TASK-007` 提示无越界。

## Manual Verification, if needed

- 阅读 `agentSkill.ts`，确认新字段命名（snake_case）与 proto 对齐。
- 确认新字段为 optional（`?:`）。
- 跑 `pnpm --filter hr-frontend typecheck`，确认无类型错误。

## Out-of-Scope

- 不得删除或修改现有字段。
- 不得在 debug 视图引入新 UI。
- 不得修改 `hr-frontend/src/api/**` / `hr-frontend/src/components/**` / `hr-frontend/src/router/**`。
- 不得修改 `hr-frontend/package.json`。
- 不得修改其他前端项目（`user-frontend` / `interviewer-frontend`）。
