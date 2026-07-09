# Skill / Memory 召回排序 — 第二轮硬化 SPEC

## 1. 背景

`.spec/skill-memory-ranking-followups/` 下的 4 个 TASK（TASK-FU-001..004）已完成 context-based debug 状态、CI 流水线、debug 视图 UI 升级、配置化权重。本 SPEC 覆盖后续硬化建议中的 5 项：

1. **warn 日志**：`applyRankingField` 在越界 clamp 时输出 `logger.L().Warn` 日志。
2. **debug 视图 breakdown tooltip**：在 `SemanticRetrievalDebugView.vue` 给 breakdown 字段标签加 `el-tooltip` 解释。
3. **RankingConfig 收敛**：把 11 个包级 `var` 收敛到单一 `RankingConfig` struct 内部，避免散落。
4. **ranking-show CLI**：新增 `cmd/ranking-show/main.go` 打印当前生效的 11 个权重 / 阈值（env 覆盖后）。
5. **web-gin-service 同步**：在 web-gin 启动时也调 `LoadRankingConfig`，保持与 logic-grpc 行为一致。

## 2. 目标

1. `applyRankingField` 在越界 / 异常输入时输出结构化 warn 日志，便于运维定位。
2. debug 视图 breakdown 字段标签带 tooltip 解释含义与取值范围；UX 提升。
3. 11 个散落的 `var rankXxx` 收敛到 `var rankingConfig` 单一 struct，命名空间更整洁。
4. `cmd/ranking-show` 命令行工具输出当前生效的 11 个权重（env 覆盖 + hardcode 默认）。
5. web-gin-service 启动时同步加载 `config.Ranking` 段，与 logic-grpc 行为一致。

## 3. 非目标

1. 不修改 TASK-001..008 / TASK-FU-001..004 已落地的算法公式 / proto 字段 / 函数签名。
2. 不引入新第三方依赖。
3. 不修改 `selectAgentSkills` / `selectAgentSkillsWithSemantic` / `rankMemories` / `DebugSemanticRetrieval` 公开签名。
4. 不修改 `cmd/ranking-show` 的输出格式（保持简单可读）。
5. web-gin-service 当前不直接使用这些权重；本期只做"启动同步"以保证后续依赖时配置一致。

## 4. 用户角色与使用场景

- **运维**：通过 `make ranking-show` 一行命令查看当前生效权重。
- **HR 业务用户**：在 debug 视图 breakdown 字段 hover 即可看到字段解释。
- **开发者**：通过 warn 日志快速定位 env 配置错误。
- **算法 / 产品**：通过 `RankingConfig` 收敛后的单一 struct 统一管理权重常量。

## 5. 功能需求

### FR-FU-005：applyRankingField warn 日志

- 描述：在 `skill_memory_ranking.go` 的 `applyRankingField` 中，检测到越界（value < min 或 value > max）时输出 `logger.L().Warn` 日志，包含 key、原值、clamp 结果。
- 输入：env 值越界（如 `RANKING_BUSINESS_BOOST_MAX=2.0`）。
- 输出：`[ranking] env value out of range, clamped` warn 日志；clamp 到合法范围。
- 边界：value == 0 不 warn（视为"未设置"）；value 在 [min, max] 区间内不 warn。

### FR-FU-006：debug 视图 breakdown tooltip

- 描述：在 `SemanticRetrievalDebugView.vue` 的 breakdown 网格中，每个字段标签（`vector` / `lexical` / `metadata` / `relevance` / `business boost` / `final rank`）加 `el-tooltip`，hover 显示字段含义。
- 边界：tooltip 文案简洁（≤ 30 字符），仅展示含义与取值范围。

### FR-FU-007：RankingConfig 收敛

- 描述：把 11 个包级 `var rankXxx` 替换为单一 `var rankingConfig` 结构（11 字段）。
- 引用更新：所有 `rankWeightVector` 等引用改为 `rankingConfig.WeightVector`。
- `LoadRankingConfig(cfg)` 改为写入 `rankingConfig` struct。
- 边界：保持 `ResetRankingConfigForTest()` 行为。

### FR-FU-008：ranking-show CLI

- 描述：新增 `logic-grpc-service/cmd/ranking-show/main.go`，打印当前生效的 11 个权重。
- 实现：
  - 读取 11 个 `RANKING_*` env 变量
  - 应用 hardcode 默认（与 `LoadRankingConfig` 行为一致）
  - 输出 11 行 key=value 表格
- 边界：不修改 `LoadRankingConfig`；仅读取。

### FR-FU-009：web-gin-service 同步

- 描述：在 `web-gin-service/main.go` 启动时调 `LoadRankingConfig`，把 `config.Ranking` 段的 11 个字段映射到 `service.RankingConfig`。
- 边界：web-gin 不直接使用这些权重；只保持配置加载行为一致。
- 注：web-gin 是否依赖 `logic-grpc-service/service` 包？需检查；如无依赖，可通过共享 proto 或简化 helper 实现。

## 6. 非功能需求

- **性能**：warn 日志在常规路径下不触发（值在 [0, 1] / [1.0, 1.5] 等合法范围内）；debug 视图 tooltip 不影响首次渲染。
- **稳定性**：现有 `go test ./...` 全部通过；前端 `pnpm --filter hr-frontend typecheck` 全部通过。
- **可观测性**：warn 日志结构化（key / value / clamped）；`ranking-show` 输出可被脚本解析（key=value 格式）。
- **可维护性**：11 个权重收敛到单一 struct 后，命名空间更整洁；debug 视图字段含义对用户透明。

## 7. 兼容性需求

- 11 个 var → 1 个 struct 转换是破坏性变更，但 `skill_memory_ranking.go` 是包内文件，不影响外部调用方。
- 公共 API（`LoadRankingConfig` / `ResetRankingConfigForTest`）签名不变。
- debug 视图 tooltip 是 additive 改动，不影响现有展示。
- `cmd/ranking-show` 是新子命令，不影响现有 `cmd/` 入口。
- web-gin-service 启动同步是 additive 行为。

## 8. 可观测性需求

- warn 日志结构化：
  ```
  [ranking] env value out of range, clamped
    key=RANKING_BUSINESS_BOOST_MAX value=2.0 clamped=1.5
  ```
- `cmd/ranking-show` 输出格式：
  ```
  weight_vector=0.6
  weight_lexical=0.3
  ...
  ```
- tooltip 文案可在 debug 视图 hover 时看到。

## 9. 异常处理与降级

| 场景 | 行为 |
| --- | --- |
| env 值越界 | warn + clamp |
| env 解析失败（非数字） | setFloat64 保留 target 原值；不 warn |
| web-gin 不依赖 logic-grpc-service | 通过 proto 或简化 helper 实现同步；如无法实现，本 TASK 降级为仅文档记录 |
| cmd/ranking-show 在没有 env 时 | 输出 hardcode 默认值 |

## 10. 安全需求

- warn 日志不输出完整 query / 简历 / Memory 内容（已遵守原 SPEC §10）。
- cmd/ranking-show 不输出敏感 env 值。
- web-gin 同步不引入新攻击面。

## 11. 验收标准

- AC-FU-005：warn 日志测试通过；正常路径无 warn。
- AC-FU-006：debug 视图 typecheck 通过；6 个字段都有 tooltip。
- AC-FU-007：`rankingConfig` struct 替代 11 个 var；既有测试全部通过。
- AC-FU-008：`go run ./cmd/ranking-show/` 输出 11 行 key=value；硬编码默认与 SDD §3.3 一致。
- AC-FU-009：web-gin-service 启动时调 `LoadRankingConfig`；`go build ./...` 成功。

## 12. 不在本期范围

- 权重热加载（运行时改 config）。
- 召回策略的 A/B 实验框架。
- 文档沉淀（用户未选）。
- proto 文件 lint / breaking-change 检查。

## 13. 待确认问题

- `applyRankingField` 的 warn 日志是否需要 `slog` / `zap` 风格（项目现有用 `zap`）。
- `cmd/ranking-show` 输出格式：`key=value` 单行 vs 表格 vs JSON。
- web-gin 同步的具体实现：直接 import logic-grpc-service/service 还是通过复制 helper。

## 14. 开放问题

- 是否在 `cmd/ranking-show` 加 `--json` flag 输出 JSON 格式。
- 是否在 debug 视图 breakdown 表格加 `final_rank` 的可视化条形（按比例显示）。
