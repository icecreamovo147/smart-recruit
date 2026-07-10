# TASK Report - TASK-FU-005

## 1. TASK ID

TASK-FU-005 — applyRankingField warn 日志

## 2. Modified File List

- `logic-grpc-service/service/skill_memory_ranking.go`（修改 `applyRankingField` + `LoadRankingConfig` 11 个调用点）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 4 个 warn 日志测试）

## 3. Change Summary by File

### `skill_memory_ranking.go`

1. **新增 import**：`go.uber.org/zap` 与 `logic-grpc-service/pkg/logger`
2. **`applyRankingField` 增加 `key string` 参数**：
   ```go
   func applyRankingField(target *float64, value, min, max float64, key string) {
       if value == 0 { return }
       if value < min {
           logger.L().Warn("[ranking] env value out of range, clamped",
               zap.String("key", key),
               zap.Float64("value", value),
               zap.Float64("clamped", min),
               zap.Float64("min", min),
               zap.Float64("max", max),
           )
           *target = min
           return
       }
       if value > max {
           logger.L().Warn("[ranking] env value out of range, clamped",
               zap.String("key", key),
               zap.Float64("value", value),
               zap.Float64("clamped", max),
               zap.Float64("min", min),
               zap.Float64("max", max),
           )
           *target = max
           return
       }
       *target = value
   }
   ```
3. **`LoadRankingConfig` 11 个调用点同步传 key**：
   - "weight_vector" / "weight_lexical" / "weight_metadata"
   - "business_boost_max" / "priority_norm"
   - "boost_alpha" / "boost_beta" / "boost_gamma"
   - "relevance_gate" / "gap_high" / "gap_medium"

### `skill_memory_ranking_test.go`

- `captureRankingLogs(t)` helper：用 `zap/zaptest/observer` 临时替换 global logger
- 4 个新测试：
  - `TestApplyRankingFieldWarnBelowMin`：value=-0.5 → clamp 到 0 + 1 条 warn
  - `TestApplyRankingFieldWarnAboveMax`：value=2.5 → clamp 到 1 + 1 条 warn
  - `TestApplyRankingFieldNoWarnOnValidValue`：value=0.7 → 无 warn
  - `TestApplyRankingFieldNoWarnOnZero`：value=0 → 无 warn（视为"未设置"）

## 4. Scope Check Result

- 实际修改文件均在 `task-scope.json` 的 `allowedFiles` 内。
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过。
- `go test ./...` (logic-grpc-service) → 全部 `ok`（11 个包）。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-005 warn 日志 | ✅ value 越界时输出 zap.Warn 含 key / value / clamped 字段 |
| §11 AC-FU-005 | ✅ 全部 6 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.1 warn 日志：完全对齐。
- SDD §10 可观测性：warn 日志结构化。
- SDD §11 测试策略：3+1 个 case（normal / below_min / above_max / zero）覆盖。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `applyRankingField` 增加 `key string` 参数 | ✅ |
| AC-002 value 越界时输出 `logger.L().Warn` 含 key / value / clamped | ✅ |
| AC-003 value 在 [min, max] 区间内不 warn | ✅ |
| AC-004 value == 0 不 warn | ✅ |
| AC-005 单测覆盖 4 种 case | ✅ |
| AC-006 `go test ./...` 通过 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestApplyRankingField' ./service/... -v` | 4 个测试全部 PASS |
| `go test ./...` (logic-grpc-service) | 全部 `ok` |
| `gofmt -l` 两个文件 | 无输出（已 gofmt） |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. Risks

- **`logger.Set` 临时替换 global**：测试用 observer logger 替换，测试结束后恢复。并发测试不冲突（Go test 串行运行）。
- **生产 warn 日志噪音**：如果运维错误配置 11 个 env 都越界，会有 11 条 warn 日志；这是预期行为（运维应该被告知）。
- **key 字符串拼写**：11 个 key 名为 SDD 字段名（snake_case），与 yaml 配置一致。

## 10. Follow-up Items

- TASK-FU-006：debug 视图 breakdown tooltip。
- 后续 PR：可考虑在 `LoadRankingConfig` 末尾输出 `logger.L().Info("[ranking] config loaded", ...)` 一次性打印 11 个生效值。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-006 可以开始。warn 日志已落地，与前端 UI 独立。
