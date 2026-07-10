# Acceptance - TASK-FU-003

## TASK Summary

在 `SemanticRetrievalDebugView.vue` 展示 TASK-005/006 新增的 8 个 breakdown 字段（vector / lexical / metadata / relevance / business_boost / final_rank_score / relevance_mode / pool_rank）以及顶部 2 个 pool_confidence（skill_pool_confidence / memory_pool_confidence）。

## SPEC References

- SPEC §5 FR-FU-003：debug 视图 UI 升级
- SPEC §7 兼容性
- SPEC §11 AC-FU-003

## SDD References

- SDD §3.3 debug 视图 UI 升级
- SDD §11 测试策略

## Acceptance Criteria

- [ ] AC-001：顶部 stats 展示 `Skill 池置信度` / `Memory 池置信度`，使用 `el-tag` 区分 high/medium/low/none
- [ ] AC-002：Skill item 展开区显示 8 个 breakdown 字段
- [ ] AC-003：Memory item 展开区显示 8 个 breakdown 字段
- [ ] AC-004：卡片角标显示 `relevance_mode` tag
- [ ] AC-005：`pnpm --filter hr-frontend typecheck` 通过
- [ ] AC-006：旧 `score` 字段展示保留

## Required Checks

- [ ] `pnpm --filter hr-frontend typecheck` 通过
- [ ] `grep -c "vector_score" hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` >= 1
- [ ] `grep -c "skill_pool_confidence" hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` >= 1
- [ ] `git diff --name-only` 仅包含 `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`

## Manual Verification, if needed

- 在 dev 模式下打开 `SemanticRetrievalDebugView.vue`，运行测试 query，确认：
  - 顶部 stats 显示 2 个新 pool_confidence 指标
  - Skill / Memory item 点击"查看详情"展示 8 个 breakdown
  - 卡片角标显示 relevance_mode tag

## Out-of-Scope

- 不得修改 TypeScript 类型（TASK-007 已完成）。
- 不得修改 API 层。
- 不得引入新组件库（继续用 Element Plus）。
- 不得在 debug 视图引入 A/B 实验、阈值调参 UI。
