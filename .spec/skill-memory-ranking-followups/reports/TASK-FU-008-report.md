# TASK Report - TASK-FU-008

## 1. TASK ID

TASK-FU-008 — ranking-show CLI

## 2. Modified File List

- `logic-grpc-service/cmd/ranking-show/main.go`（新增）
- `logic-grpc-service/service/skill_memory_ranking.go`（新增 `SnapshotRankingConfig` helper）

## 3. Change Summary by File

### `cmd/ranking-show/main.go`（新增 ~50 行）

1. **package + doc comment**：说明用途、用法、env 覆盖行为
2. **main() 函数**：
   - `config.Load()`（best-effort；dev 模式 fallback 到 `config.example.yaml`；err 不 fatal，warn 后继续）
   - `service.LoadRankingConfig(service.RankingConfig{...})` 把 11 个字段映射过去
   - `service.SnapshotRankingConfig()` 拿当前生效 struct 副本
   - 11 行 `printLine(key, value)` 输出 `key=%.4f`
3. **`printLine` helper**：`fmt.Printf("%s=%.4f\n", key, value)`

### `service/skill_memory_ranking.go`

- **新增 `SnapshotRankingConfig() RankingConfig`**：返回当前生效的 rankingConfig struct 副本（不暴露可变引用）。

## 4. Scope Check Result

- 实际修改文件均在 `task-scope.json` 的 `allowedFiles` 内。
- `go build ./...` (logic-grpc-service) → 成功（含新 cmd/ranking-show）。
- `go test ./...` → 全部 `ok`。
- CLI 手工验证：
  - `ALLOW_INSECURE_DEV_CONFIG=true /tmp/ranking-show` → 输出 11 行 hardcode 默认
  - `RANKING_WEIGHT_VECTOR=0.7 RANKING_BUSINESS_BOOST_MAX=1.3 ...` → 输出 11 行含覆盖值
  - `RANKING_BUSINESS_BOOST_MAX=2.5 ...` → 输出 warn 日志 + 1.5（clamp）
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-008 ranking-show CLI | ✅ 新增 `cmd/ranking-show/main.go` 打印 11 行 key=value |
| §11 AC-FU-008 | ✅ 全部 5 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.4 ranking-show CLI：完全对齐。
- SDD §8 兼容性策略：新子命令不影响现有 cmd/ 入口。
- SDD §11 测试策略：手工执行验证（无需单测）。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `go run ./cmd/ranking-show/` 输出 11 行 | ✅ |
| AC-002 每行格式 `key=value`（4 位小数） | ✅（`%.4f`） |
| AC-003 env 覆盖时输出覆盖值 | ✅（`RANKING_WEIGHT_VECTOR=0.7` → 0.7000） |
| AC-004 未设 env 时输出 hardcode 默认 | ✅ |
| AC-005 `go build ./...` 通过 | ✅ |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go build -o /tmp/ranking-show ./cmd/ranking-show/` | 成功 |
| `ALLOW_INSECURE_DEV_CONFIG=true /tmp/ranking-show` | 11 行 hardcode 默认 |
| `RANKING_WEIGHT_VECTOR=0.7 RANKING_BUSINESS_BOOST_MAX=1.3 ...` | 11 行含覆盖值 |
| `RANKING_BUSINESS_BOOST_MAX=2.5 ...` | warn 日志 + 1.5（clamp） |
| `go test ./...` (logic-grpc-service) | 全部 `ok` |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. Risks

- **CLI 修改全局 state**：`LoadRankingConfig` 会写入 package-level `rankingConfig`；如果在生产中误调，可能影响其他 goroutine 的排序结果。CLI 是独立进程，影响有限；可在 `cmd/ranking-show/main.go` 注释中提示。
- **`SnapshotRankingConfig` 返回值是按值拷贝**：safe。
- **`config.Load()` 在 dev 模式 fallback**：与生产 logic-grpc 行为一致。

## 10. Follow-up Items

- TASK-FU-009：web-gin-service 同步 `LoadRankingConfig`。
- 后续 PR：可考虑加 `--json` flag 输出 JSON 格式。
- 后续 PR：可考虑在 `Makefile` 加 `make ranking-show` target。

## 11. Whether the Next TASK Can Start

✅ TASK-FU-009 可以开始。CLI 已就绪；web-gin 同步是独立 backend 改动。
