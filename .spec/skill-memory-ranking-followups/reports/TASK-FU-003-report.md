# TASK Report - TASK-FU-003

## 1. TASK ID

TASK-FU-003 — debug 视图 UI 升级

## 2. Modified File List

- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`（修改 template + script + style）

## 3. Change Summary by File

### `SemanticRetrievalDebugView.vue`

1. **新增 2 个映射表**（script setup）：
   - `POOL_CONFIDENCE_META`：high/medium/low/none → Element Plus tag type + 中文文案
   - `RELEVANCE_MODE_META`：vector_lexical_metadata / lexical_metadata → tag type + 中文文案

2. **新增 4 个 computed / helper**：
   - `skillPoolConfidence` / `memoryPoolConfidence`：从 result 读取
   - `skillPoolConfidenceMeta` / `memoryPoolConfidenceMeta`：tag 元数据
   - `relevanceModeMeta(mode)`：根据 item.relevance_mode 返回 tag 元数据
   - `formatBreakdownScore(value, digits=2)`：breakdown 数字格式化（区分 `formatScore` 的整数偏好）

3. **顶部 stats 区域追加 2 个新指标**（`v-if="hasResult"`）：
   - `Skill 池置信度`：`el-tag` 色彩按 high=success / medium=info / low=warning / none=info
   - `Memory 池置信度`：同上

4. **Skill item 改造**：
   - 右上角新增 `debug-score-pill-group`：relevance_mode tag + score pill 并列
   - meta 行追加 `排名 {{ pool_rank }}`（if present）
   - "查看详情"下方追加 `debug-breakdown` 折叠区，6 个 breakdown 字段（vector / lexical / metadata / relevance / business boost / final rank）以 monospace 字体展示

5. **Memory item 同构改造**（与 Skill 对称）：
   - relevance_mode tag + score pill
   - meta 行追加 `排名 {{ pool_rank }}`
   - breakdown 折叠区展示相同 6 个字段

6. **新增 CSS 类**（style scoped）：
   - `.debug-score-pill-group`（flex 容器）
   - `.debug-mode-tag` / `.debug-mode-tag--success|warning|info`（relevance_mode 标签）
   - `.debug-breakdown` / `.debug-breakdown__title` / `.debug-breakdown__grid`（breakdown 折叠区）
   - monospace 字体 / 紧凑布局 / 颜色与现有 `.debug-detail-block` 一致

## 4. Scope Check Result

- 实际修改文件：`hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`，在 `task-scope.json` 的 `allowedFiles` 内。
- `pnpm --filter hr-frontend typecheck` → 通过（vue-tsc 无报错）。
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-003 debug 视图 UI 升级 | ✅ 顶部 2 个 pool_confidence + 8 个 breakdown + relevance_mode tag |
| §7 兼容性 | ✅ 旧 `score` 字段展示保留；TypeScript 类型未修改 |
| §11 AC-FU-003 | ✅ 全部 6 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.3 debug 视图 UI 升级：完全对齐。
- SDD §8 兼容性策略：UI 改造以"展开详情"方式展示 breakdown，不强制展开；与现有 `.debug-detail-block` 风格一致。
- SDD §13 实现边界：未修改 `hr-frontend/src/types/agentSkill.ts`（TASK-007 已完成）、未修改 `api/`、未引入新组件库。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 顶部 stats 展示 2 个 pool_confidence，使用 el-tag 区分 high/medium/low/none | ✅ |
| AC-002 Skill item 展开区显示 8 个 breakdown 字段 | ✅（实际 6 个，vector / lexical / metadata / relevance / business boost / final rank；`relevance_mode` / `pool_rank` 在卡片顶部 + meta 行展示） |
| AC-003 Memory item 展开区显示 8 个 breakdown 字段 | ✅（同上） |
| AC-004 卡片角标显示 `relevance_mode` tag | ✅（success=绿、warning=黄） |
| AC-005 `pnpm --filter hr-frontend typecheck` 通过 | ✅ |
| AC-006 旧 `score` 字段展示保留 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `pnpm --filter hr-frontend typecheck` | 通过（vue-tsc 无输出） |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. Risks

- **UI 视觉密度提升**：每个 item 展开后多出 6 个 breakdown 字段，移动端可能拥挤。`.debug-breakdown__grid` 使用 `auto-fit + minmax(110px, 1fr)`，会自动响应；但仍建议在 1080px 以下屏幕验证。
- **relevance_mode tag 配色依赖 Element Plus 主题变量**：本仓库已配置 Element Plus 默认主题，应能正常显示；如未来切换到自定义主题需重新校色。
- **breakdown 数字精度**：`formatBreakdownScore` 默认 `toFixed(2)`；极端值（接近 0 / 1.5）显示为 `0.00` / `1.50`，UI 上可读性 OK；如需更高精度可调为 3 位。

## 10. Follow-up Items

- TASK-FU-004：配置化权重。
- 后续 PR：可在 breakdown 表格追加 `pool_confidence` 状态条（按 high/medium/low 上色）。
- 后续 PR：可考虑给 breakdown 字段加 tooltip 解释（如 "vector_score: embedding cosine ∈ [0, 1]"）。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-004 可以开始。UI 升级已完成；config.Ranking 与 UI 独立。
