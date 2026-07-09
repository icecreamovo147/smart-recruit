# TASK Report - TASK-FU-006

## 1. TASK ID

TASK-FU-006 — debug 视图 breakdown tooltip

## 2. Modified File List

- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`（script + 2 个 template 段）

## 3. Change Summary by File

### `SemanticRetrievalDebugView.vue`

1. **新增 `BREAKDOWN_TOOLTIPS` 映射表**（script setup）：
   ```ts
   const BREAKDOWN_TOOLTIPS: Record<string, string> = {
     vector: 'embedding cosine ∈ [0, 1]',
     lexical: 'keyword hit ratio ∈ [0, 1]',
     metadata: 'scope / category hit ∈ [0, 1]',
     relevance: 'weighted sum ∈ [0, 1]',
     'business boost': 'relevance × boost ∈ [0, 1.5]',
     'final rank': 'final_rank_score ∈ [0, 1.5]',
   }
   ```

2. **Skill item breakdown 网格改造**：每个字段的 `<span>label</span>` 包到 `<el-tooltip :content="..." placement="top">` 内。6 个字段 × 1 个 article = 6 个 tooltip。

3. **Memory item breakdown 网格同样改造**：6 个字段 × 1 个 article = 6 个 tooltip。

4. 旧展示行为完全保留：tooltip 仅在 hover 时显示，不影响首屏渲染。

## 4. Scope Check Result

- 实际修改文件：`hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`，在 `task-scope.json` 的 `allowedFiles` 内。
- `pnpm --filter hr-frontend typecheck` → 通过（vue-tsc 无报错）。
- `grep -c "el-tooltip" hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` = 24（6 字段 × 2 article × 2：opening + closing tag）。
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-006 tooltip | ✅ 6 个 breakdown 字段都有 tooltip |
| §11 AC-FU-006 | ✅ 全部 4 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.2 tooltip：6 个字段都加 `el-tooltip`；文案简洁（≤ 30 字符）。
- SDD §13 实现边界：仅修改 `views/hr/admin/SemanticRetrievalDebugView.vue`；未修改 types / api / components / package.json。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 6 个 breakdown 字段都有 tooltip | ✅ |
| AC-002 tooltip 文案简洁（≤ 30 字符） | ✅（最长 28 字符） |
| AC-003 `pnpm --filter hr-frontend typecheck` 通过 | ✅ |
| AC-004 旧展示行为不变 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `pnpm --filter hr-frontend typecheck` | 通过（vue-tsc 无输出） |
| `grep -c "el-tooltip" SemanticRetrievalDebugView.vue` | 24（12 × 2 = opening + closing） |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. Risks

- **Element Plus tooltip 默认延迟 250ms**：hover 后需等待才显示。如未来需要即时显示，可设置 `:open-delay="0"`。
- **多 tooltip 同时存在**：12 个 tooltip 在两个 article 内独立，无冲突。
- **移动端无 hover**：tooltip 默认在桌面端 hover 触发；移动端需点击（Element Plus 默认行为）。本 debug 视图是后台工具，主要用户为开发 / 运维，桌面端为主。

## 10. Follow-up Items

- TASK-FU-007：RankingConfig 收敛。
- 后续 PR：可考虑给 `relevance_mode` 角标也加 tooltip（`vector_lexical_metadata` / `lexical_metadata` 的语义）。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-007 可以开始。tooltip 已落地，与 backend 重构独立。
