# Pipeline Summary - skill-memory-ranking-followups (Round 2)

## 执行概况

- 起始 TASK: TASK-FU-005
- 结束 TASK: TASK-FU-009
- 完成: 5 / 5
- 失败: 无
- 跳过: 无
- 状态: `completed`
- 注: 用户未选择"文档沉淀"项；其余 5 项全部完成

## 各 TASK 结果

| TASK | 标题 | 状态 | 审查轮数 | 报告 |
|------|------|------|----------|------|
| TASK-FU-005 | applyRankingField warn 日志 | ✅ | 1 | [report](reports/TASK-FU-005-report.md) |
| TASK-FU-006 | debug 视图 breakdown tooltip | ✅ | 1 | [report](reports/TASK-FU-006-report.md) |
| TASK-FU-007 | RankingConfig 收敛为单个 struct | ✅ | 1 | [report](reports/TASK-FU-007-report.md) |
| TASK-FU-008 | ranking-show CLI | ✅ | 1 | [report](reports/TASK-FU-008-report.md) |
| TASK-FU-009 | web-gin-service 同步 LoadRankingConfig | ✅ | 1 | [report](reports/TASK-FU-009-report.md) |

## 总修改文件

| 文件 | TASK | 类型 |
|------|------|------|
| `logic-grpc-service/service/skill_memory_ranking.go` | 005, 007, 008 | 改：warn 日志 + struct 收敛 + SnapshotRankingConfig |
| `logic-grpc-service/service/skill_memory_ranking_test.go` | 005, 007 | 改：4 个 warn 测试 + 114 处引用更新 |
| `logic-grpc-service/cmd/ranking-show/main.go` | 008 | 新增（~50 行） |
| `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` | 006 | 改：6 字段 × 2 article = 12 个 tooltip |
| `web-gin-service/config/config.go` | 009 | 改：Ranking struct + envFloat64 helper + 11 个 RANKING_* env |
| `web-gin-service/main.go` | 009 | 改：启动日志输出 ranking 段 |
| `web-gin-service/config/config_test.go` | 009 | 改：3 个 ranking 加载测试 |

## 测试 / Harness 检查结果

- **`go test ./...` (logic-grpc-service)**: 全部 `ok`（11 个包）
- **`go test ./...` (web-gin-service)**: 全部 `ok`
- **`go build ./...` (logic-grpc + web-gin)**: 成功
- **`pnpm --filter hr-frontend typecheck`**: 通过
- **`bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh`**: 全部通过
- **`ranking-show` CLI 手工验证**:
  - 11 行 hardcode 默认输出 ✅
  - env 覆盖生效 ✅
  - 越界 clamp + warn 日志 ✅

## 各 TASK 关键实现要点

### TASK-FU-005：applyRankingField warn 日志
- `applyRankingField(target, value, min, max, key)` 增加 `key` 参数
- value 越界时 `logger.L().Warn("[ranking] env value out of range, clamped", zap.String("key", ...), zap.Float64("value", ...), zap.Float64("clamped", ...), ...)`
- 4 个新测试覆盖 below_min / above_max / normal / zero

### TASK-FU-006：debug 视图 breakdown tooltip
- `BREAKDOWN_TOOLTIPS` 映射表（6 字段 × 1 文案）
- Skill + Memory item 各 6 个 `<el-tooltip>` 包裹
- 24 个 el-tooltip tag 实例（12 fields × 2 opening/closing）

### TASK-FU-007：RankingConfig 收敛为单个 struct
- 11 个 `var rankXxx` → 1 个 `var rankingConfig` struct
- sed 批量替换 47 + 67 = 114 处引用
- `LoadRankingConfig` / `ResetRankingConfigForTest` 签名不变

### TASK-FU-008：ranking-show CLI
- 新增 `cmd/ranking-show/main.go`
- `config.Load()` + `service.LoadRankingConfig()` + `service.SnapshotRankingConfig()`
- 输出 11 行 `key=%.4f`
- 新增 `SnapshotRankingConfig` helper（返回 struct 副本）

### TASK-FU-009：web-gin-service 同步
- web-gin 不依赖 logic-grpc-service，走 fallback 方案：
  - 本地 `Ranking` struct（与 logic-grpc-service 同形）
  - `envFloat64` helper
  - 11 个 RANKING_* env 解析
- main.go 启动时输出 `[ranking] config loaded` 结构化日志
- 3 个新 config 测试

## 风险与待确认项

1. **`applyRankingField` 测试需用 `observer` 临时替换 global logger**：本 TASK 用 `zap/zaptest/observer` 实现；并发测试需注意 goroutine 隔离（当前 Go test 串行运行，OK）。
2. **11 vars → 1 struct 大批量 refactor**：114 处引用一次性更新；如未来回滚需还原 11 个 var 声明 + 114 处引用。
3. **`ranking-show` CLI 修改全局 state**：CLI 是独立进程，影响有限；可在 `cmd/ranking-show/main.go` 注释中提示。
4. **web-gin 与 logic-grpc 的 Ranking 段重复定义**：两端各有一份 struct；如未来 logic-grpc 增加字段需同步 web-gin 端。后续 PR 可抽到 `pkg/config` 共享包。
5. **web-gin 端不实际使用 cfg.Ranking**：当前是日志输出 + 未来扩展准备；如 web-gin 引入排序逻辑，可直接读 cfg.Ranking。
6. **web-gin 端越界静默**：不 clamp（与 logic-grpc 端 `applyRankingField` 不同）。如未来 web-gin 也使用这些值，可加 clamp + warn。
7. **Element Plus tooltip 默认 250ms 延迟**：hover 后需等待；如需即时可设 `:open-delay="0"`。

## 后续建议

1. **`Ranking` struct 抽到 `pkg/config` 共享包**：消除 web-gin / logic-grpc 重复定义。
2. **web-gin 端也加 `applyRankingField` + warn 日志**：与 logic-grpc 行为对齐。
3. **`ranking-show` CLI 加 `--json` flag**：输出 JSON 格式便于脚本解析。
4. **Makefile 加 `make ranking-show` target**。
5. **`SnapshotRankingConfig` 改名为 `RankingSnapshot`**：命名更清晰。
6. **`LoadRankingConfig` 末尾输出 `logger.L().Info("[ranking] config loaded", ...)` 一次性打印 11 个生效值**：与 web-gin 端日志一致。
7. **relevance_mode 角标 tooltip**：debug 视图 `vector_lexical_metadata` / `lexical_metadata` 角标也加 tooltip 解释。
8. **CI 跑一次 env 覆盖的 `go test`**：`RANKING_WEIGHT_VECTOR=0.7 go test ./...` 验证 env 覆盖路径。
9. **CI 跑一次 `RANKING_BUSINESS_BOOST_MAX=2.0 go test ./...`**：验证越界 clamp + warn 日志。
10. **文档沉淀**（用户未选）：可在 `docs/` 增加"召回排序"章节，记录权重常量与阈值。

## SPEC / SDD / Acceptance 对齐情况

### SPEC
- §5 FR-FU-005 ~ FR-FU-009: 全部 5 个功能需求实现
- §7 兼容性：内部 refactor（TASK-FU-007）；其他都是 additive
- §11 AC-FU-005 ~ AC-FU-009: 全部验收标准通过

### SDD
- §3.1 warn 日志：完全对齐
- §3.2 tooltip：完全对齐
- §3.3 RankingConfig 收敛：完全对齐
- §3.4 ranking-show CLI：完全对齐
- §3.5 web-gin 同步：fallback 方案（不直接 import logic-grpc）
- §11 测试策略：warn / tooltip / 收敛 / CLI / web-gin 全部覆盖

### Acceptance
- AC-FU-005 ~ AC-FU-009: 全部通过
- 5 个 TASK 各自的 acceptance criteria 全部通过（详见各 TASK 报告）
