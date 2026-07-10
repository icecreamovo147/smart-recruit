# TASK Report - TASK-FU-004

## 1. TASK ID

TASK-FU-004 — 配置化权重（config.Ranking）

## 2. Modified File List

- `logic-grpc-service/config/config.go`（新增 `Ranking` struct + 11 个 env 覆盖 + `setFloat64` helper）
- `logic-grpc-service/config/config.example.yaml`（增加 `ranking` 段）
- `logic-grpc-service/service/skill_memory_ranking.go`（const → var + `LoadRankingConfig` + `ResetRankingConfigForTest`）
- `logic-grpc-service/main.go`（启动时调 `LoadRankingConfig`）
- `logic-grpc-service/service/skill_memory_ranking_test.go`（追加 5 个 config 测试）

## 3. Change Summary by File

### `config/config.go`

1. **新增 `Ranking` struct**（紧跟 `Embedding`）：
   - 11 个 `float64` 字段，每个带 `yaml` tag
   - 与 SDD §3.3 字段名完全对齐：weight_vector / weight_lexical / weight_metadata / business_boost_max / priority_norm / boost_alpha / boost_beta / boost_gamma / relevance_gate / gap_high / gap_medium
2. **Config.Ranking 字段**：`Ranking Ranking` 嵌入主 `Config`；yaml tag `ranking`。
3. **env 覆盖**（11 个 `setFloat64`）：
   - `RANKING_WEIGHT_VECTOR` / `RANKING_WEIGHT_LEXICAL` / `RANKING_WEIGHT_METADATA` / `RANKING_BUSINESS_BOOST_MAX` / `RANKING_PRIORITY_NORM` / `RANKING_BOOST_ALPHA` / `RANKING_BOOST_BETA` / `RANKING_BOOST_GAMMA` / `RANKING_RELEVANCE_GATE` / `RANKING_GAP_HIGH` / `RANKING_GAP_MEDIUM`
4. **新增 `setFloat64` helper**：解析失败时保留 target 原值（hardcode 默认）。

### `config/config.example.yaml`

- 在 `embedding` 段后、`frontend_base_url` 之前增加 `ranking` 段；11 个字段 + 注释 + env 覆盖提示。

### `service/skill_memory_ranking.go`

1. **`rankingDefaults` 匿名 struct**：11 个 hardcode 默认值（与原 const 数值完全一致）。
2. **11 个 `const` → `var`**：以 `rankingDefaults.XXX` 初始化，运行时可被 `LoadRankingConfig` 覆盖。
3. **`RankingConfig` struct**（service 包本地）：与 `config.Ranking` 同形；用作 `LoadRankingConfig` 入参，避免 service 包直接依赖 config 包。
4. **`LoadRankingConfig(cfg RankingConfig)`**：11 个 `applyRankingField` 调用；非 0 值覆盖；越界 clamp 到合法范围。
5. **`applyRankingField(target, value, min, max)`**：0 → 跳过；value < min → min；value > max → max；否则 value。
6. **`ResetRankingConfigForTest()`**：恢复 11 个 var 到 hardcode 默认，供测试 setUp / tearDown 使用。

### `main.go`

- 在 `config.Load()` 之后立即调 `service.LoadRankingConfig(...)`，把 `config.Ranking` 的 11 个字段映射到 `service.RankingConfig`。

### `skill_memory_ranking_test.go`

- `TestLoadRankingConfigDefaults`（empty_config_keeps_hardcode）
- `TestLoadRankingConfigOverride`（weight_vector_override / multiple_overrides）
- `TestLoadRankingConfigClamp`（clamp_weight_to_unit_interval / clamp_business_boost_max）
- `TestResetRankingConfigForTest`（reset_restores_all_11_vars）
- `TestRankingConfigIntegration`（config_to_ranking_to_var，验证端到端：env → config → service.RankingConfig → var → computeRelevanceScore）

## 4. Scope Check Result

- 实际修改文件均在 `task-scope.json` 的 `allowedFiles` 内：
  - `logic-grpc-service/config/config.go`
  - `logic-grpc-service/config/config.example.yaml`
  - `logic-grpc-service/service/skill_memory_ranking.go`
  - `logic-grpc-service/service/skill_memory_ranking_test.go`
  - `logic-grpc-service/main.go`（`cmd/**/main.go` 模式）
- `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` → 全部通过（gofmt / go build / go test / web-gin build / hr-frontend typecheck / ci.yml yaml parse）。
- `python3 -c 'yaml.safe_load(config.example.yaml)'` → 成功；`ranking` 段 11 个字段全部存在。

## 5. SPEC Comparison Result

| SPEC 引用 | 实现情况 |
| --- | --- |
| FR-FU-004 配置化权重 | ✅ config.Ranking + env + yaml + LoadRankingConfig 完整实现 |
| §7 兼容性 | ✅ 11 个 var 默认值与原 const 完全一致；空值回退到 hardcode |
| §11 AC-FU-004 | ✅ 全部 6 条 AC 满足 |

## 6. SDD Comparison Result

- SDD §3.4 配置化权重：完全对齐。
- SDD §7 配置设计：env `RANKING_*` + yaml `ranking.<field>` 双重覆盖；硬编码默认。
- SDD §8 兼容性策略：`config.Ranking` 字段为 additive；空值回退。
- SDD §11 测试策略：5 个测试（defaults / override / clamp / reset / integration）。

## 7. Acceptance Comparison Result

| AC | 结果 |
| --- | --- |
| AC-001 `config.Ranking` 段定义完整；11 个字段都有 yaml + env tag | ✅ |
| AC-002 `LoadRankingConfig` 在 nil / 空 / 部分填充 / 越界 4 种输入下行为正确 | ✅（defaults / override / clamp 测试覆盖） |
| AC-003 `ResetRankingConfigForTest` 恢复 11 个 var 到 hardcode 默认 | ✅ |
| AC-004 `go test ./...` 全部通过 | ✅ |
| AC-005 现有 5 个 TASK-001..004 既有测试通过 | ✅（`TestComputeRelevanceScore` / `TestComputePoolConfidence` 等无回归） |
| AC-006 `config.example.yaml` 增加 `ranking` 段注释 | ✅（11 字段 + env 覆盖提示） |

## 8. Test Commands and Results

| 命令 | 结果 |
| --- | --- |
| `go test -run 'TestLoadRankingConfig\|TestResetRankingConfigForTest\|TestRankingConfigIntegration' ./service/...` | 全部 `PASS` |
| `go test ./...` (logic-grpc-service) | 全部 `ok` |
| `gofmt -l` 4 个文件 | 无输出（已 gofmt） |
| `python3 -c 'yaml.safe_load(config.example.yaml)'` | 成功 |
| `bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh` | 全部通过 |

## 9. Risks

- **`service` 包定义 `RankingConfig`**：与 `config.Ranking` 字段名相同但结构独立。`main.go` 转换路径（`config.Ranking` → `service.RankingConfig`）需手动同步 11 个字段；如未来 config 段字段增加，需同步更新。已在 main.go 显式列出 11 个字段。
- **`applyRankingField` 0 值跳过**：如果用户希望显式设置 `weight_vector = 0`（即不参与排序），会被视为"未设置"而保持默认值。如未来需要支持显式 0，可改为 `*float64` 指针。
- **越界 clamp 静默**：当前不 warn 日志（仅 clamp）。如未来需要审计，可在 `applyRankingField` 中加 `logger.L().Warn`。
- **启动时一次性 init**：env / yaml 解析在 `main.go` 启动时完成；运行时不支持热加载（与 SPEC §3.4 / §5 一致）。

## 10. Follow-up Items

- 后续 PR：可考虑把 `applyRankingField` 的越界事件写入 `logger.L().Warn`。
- 后续 PR：可考虑 `make ranking-show` 命令行工具打印当前生效的权重。
- 后续 PR：可考虑 CI 中跑一次 `RANKING_WEIGHT_VECTOR=0.7 go test ./...` 验证 env 覆盖路径。
- 后续 PR：可考虑把 11 个 var 收敛到一个 `RankingConfig` struct 内部，避免散落的 11 个包级变量。

## 11. Whether the Next TASK Can Start

✅ 4 个 TASK-FU 全部完成；可以生成 followup pipeline-summary.md 收尾。
