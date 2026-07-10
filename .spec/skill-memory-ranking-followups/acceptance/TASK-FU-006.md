# Acceptance - TASK-FU-006

## TASK Summary

debug 视图的 6 个 breakdown 字段加 `el-tooltip` 解释。

## SPEC References

- SPEC §5 FR-FU-006
- SPEC §11 AC-FU-006

## SDD References

- SDD §3.2 debug 视图 tooltip

## Acceptance Criteria

- [ ] AC-001：6 个 breakdown 字段都有 tooltip
- [ ] AC-002：tooltip 文案简洁（≤ 30 字符）
- [ ] AC-003：`pnpm --filter hr-frontend typecheck` 通过
- [ ] AC-004：旧展示行为不变

## Required Checks

- [ ] `pnpm --filter hr-frontend typecheck` 通过
- [ ] `grep -c "el-tooltip" hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` >= 6
- [ ] `git diff --name-only` 仅包含 `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`

## Manual Verification, if needed

- 在 dev 模式下打开 `SemanticRetrievalDebugView.vue`，运行测试 query，hover breakdown 字段确认 tooltip 出现。

## Out-of-Scope

- 不得修改 TypeScript 类型。
- 不得引入新组件库。
