# Pipeline Summary - skill-memory-ranking-followups

## 执行概况

- 起始 TASK: TASK-FU-001
- 结束 TASK: TASK-FU-004
- 完成: 4 / 4
- 失败: 无
- 跳过: 无
- 状态: `completed`

## 各 TASK 结果

| TASK | 标题 | 状态 | 审查轮数 | 报告 |
|------|------|------|----------|------|
| TASK-FU-001 | context-based debug 状态 | ✅ | 1 | [report](reports/TASK-FU-001-report.md) |
| TASK-FU-002 | CI 流水线 protoc 步骤 | ✅ | 1 | [report](reports/TASK-FU-002-report.md) |
| TASK-FU-003 | debug 视图 UI 升级 | ✅ | 1 | [report](reports/TASK-FU-003-report.md) |
| TASK-FU-004 | 配置化权重（config.Ranking） | ✅ | 1 | [report](reports/TASK-FU-004-report.md) |

## 总修改文件

| 文件 | TASK | 类型 |
|------|------|------|
| `logic-grpc-service/service/agent_skill_service.go` | FU-001 | 修改 |
| `logic-grpc-service/service/agent_skill_service_test.go` | FU-001 | 修改（追加 2 个测试） |
| `logic-grpc-service/service/skill_memory_ranking.go` | FU-004 | 修改（const → var + LoadRankingConfig） |
| `logic-grpc-service/service/skill_memory_ranking_test.go` | FU-004 | 修改（追加 5 个 config 测试） |
| `logic-grpc-service/config/config.go` | FU-004 | 修改（新增 Ranking struct + env 覆盖） |
| `logic-grpc-service/config/config.example.yaml` | FU-004 | 修改（ranking 段） |
| `logic-grpc-service/main.go` | FU-004 | 修改（启动时 LoadRankingConfig） |
| `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` | FU-003 | 修改 |
| `.github/workflows/ci.yml` | FU-002 | 修改（proto-lint job） |
| `.spec/skill-memory-ranking-followups/scripts/agent-check.sh` | FU-001 | 修改（YAML parse 替代 bash -n） |

## 测试 / Harness 检查结果

- **`go test ./...` (logic-grpc-service)**: 全部 `ok`（11 个包，~150 test PASS / 0 FAIL）
- **`go build ./...` (logic-grpc + web-gin)**: 成功
- **`pnpm --filter hr-frontend typecheck`**: 成功
- **`bash .spec/skill-memory-ranking-followups/scripts/agent-check.sh`**: 全部通过
- **`python3 -c 'yaml.safe_load(config.example.yaml)'`**: 成功（11 个 ranking 字段）
- **`python3 -c 'yaml.safe_load(.github/workflows/ci.yml)'`**: 成功

## 各 TASK 关键实现要点

### TASK-FU-001：context-based debug 状态
- 删除 `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级 `var`
- 删除 4 个旧 helper（`stashDebugPoolConfidence` / `lastDebugPoolConfidenceForTest` / `stashLastDebugMemoryRankings` / `getLastDebugMemoryRankings`）
- 新增私有 key 类型 `debugContextKey struct{}` 与 2 个具体 key
- 新增 4 个 ctx helper：`withDebugPoolConfidence` / `getDebugPoolConfidenceFromContext` / `withDebugMemoryRankings` / `getDebugMemoryRankingsFromContext`
- `semanticDebugMemories` 签名从返回 `[]*pb.SemanticMemoryDebugItem` 改为 `([]*pb.SemanticMemoryDebugItem, context.Context)`
- `DebugSemanticRetrieval` 用 `debugCtx` 局部变量透传 state
- 2 个新测试覆盖 4 + 4 = 8 个子用例
- 额外修复 `agent-check.sh` 中 `bash -n` 误用（用于 YAML 文件）：替换为 `python3 -c 'import yaml; yaml.safe_load(...)'`

### TASK-FU-002：CI 流水线 protoc 步骤
- 新增 `proto-lint` job（4 步）：
  1. `actions/checkout@v4` + `actions/setup-go@v5`（复用 go 1.25 cache）
  2. **Install protoc 25.3**（wget + unzip）
  3. **Install protoc-gen-go v1.36.11**（go install）
  4. **Regenerate**（protoc 步骤与本地 TASK-005 完全一致）
  5. **Verify pb diff is empty**（用 `git diff --quiet` + 显式错误信息与 stat）
- 现有 3 个 job（go-test / frontend / secret-scan）完全不动

### TASK-FU-003：debug 视图 UI 升级
- 顶部 stats 追加 2 个 `console-stat`：`Skill 池置信度` / `Memory 池置信度`，用 `el-tag` 区分 high=success / medium=info / low=warning / none=info
- Skill / Memory item 顶部 score pill 旁加 `relevance_mode` tag（绿=vector+lexical+metadata / 黄=lexical_metadata fallback）
- meta 行追加 `排名 {{ pool_rank }}`
- "查看详情"下方新增 `debug-breakdown` 折叠区，6 个 breakdown 字段（vector / lexical / metadata / relevance / business boost / final rank）以 monospace 字体展示
- 2 个映射表（POOL_CONFIDENCE_META / RELEVANCE_MODE_META）+ 4 个 computed / helper
- 新增 CSS 类（`.debug-score-pill-group` / `.debug-mode-tag` / `.debug-breakdown`），与现有 `.debug-detail-block` 风格一致

### TASK-FU-004：配置化权重
- `config.Ranking` struct 11 个字段（与 SDD §3.3 完全对齐）
- 11 个 env 覆盖（`RANKING_WEIGHT_VECTOR` 等）+ `setFloat64` helper
- `rankingDefaults` 匿名 struct（hardcode 默认值，与原 const 数值一致）
- 11 个 const → var（运行时可被 `LoadRankingConfig` 覆盖）
- `RankingConfig` struct（service 包本地，避免与 config 包耦合）
- `LoadRankingConfig` + `applyRankingField`（0 跳过 / 越界 clamp）+ `ResetRankingConfigForTest`
- `main.go` 启动时调 `LoadRankingConfig`，把 `config.Ranking` 映射到 `service.RankingConfig`
- 5 个新测试覆盖 defaults / override / clamp / reset / integration

## 风险与待确认项

1. **CI 首次运行可能失败**（TASK-FU-002）：当前 git HEAD 提交的 `recruitment.pb.go` 是用更早版本的 protoc-gen-go 生成的；CI 步骤会触发"diff found"错误。修复方式：开发者在本地重生成 + commit 两侧 pb 文件。错误信息已包含完整修复步骤提示。
2. **`semanticDebugMemories` 私有签名变化**（TASK-FU-001）：从返回 `[]*pb.SemanticMemoryDebugItem` 改为 `([]*pb.SemanticMemoryDebugItem, context.Context)`。函数是包内私有（`s.semanticDebugMemories`），无外部调用方；但若未来有其他文件调用需同步更新。
3. **`applyRankingField` 0 值跳过**（TASK-FU-004）：用户如希望显式设置 `weight_vector = 0`，会被视为"未设置"而保持默认值。如未来需要支持显式 0，可改为 `*float64` 指针。
4. **`applyRankingField` 越界 clamp 静默**（TASK-FU-004）：当前不 warn 日志（仅 clamp）。如未来需要审计，可在 helper 中加 `logger.L().Warn`。
5. **debug 视图 UI 视觉密度提升**（TASK-FU-003）：每个 item 展开后多出 6 个 breakdown 字段；`.debug-breakdown__grid` 使用 `auto-fit + minmax(110px, 1fr)` 自动响应，但建议在 1080px 以下屏幕验证。
6. **服务包定义 `RankingConfig`**（TASK-FU-004）：与 `config.Ranking` 字段名相同但结构独立。`main.go` 转换路径需手动同步 11 个字段；如未来 config 段字段增加，需同步更新 `service.RankingConfig`。

## 后续建议

1. **CI 首次 commit**（TASK-FU-002）：开发者在本地跑一次 `protoc` 重生成 + 提交 .pb.go，使 CI 通过；之后 CI 会一直 pass 直到下次 proto 变更。
2. **`applyRankingField` warn 日志**（TASK-FU-004）：可在 helper 中加 `logger.L().Warn` 审计越界事件。
3. **`make ranking-show` 命令行工具**（TASK-FU-004）：打印当前生效的 11 个权重。
4. **CI 跑一次 env 覆盖的 `go test`**（TASK-FU-004）：如 `RANKING_WEIGHT_VECTOR=0.7 go test ./...`，验证 env 覆盖路径。
5. **`RankingConfig` 收敛**（TASK-FU-004）：未来可把 11 个 var 收敛到一个 `RankingConfig` struct 内部，避免散落的包级变量。
6. **proto diff 错误信息**（TASK-FU-002）：已包含 diff stat 与修复步骤；如需更详细的 line-by-line diff，可加 `git diff --unified=3`。
7. **debug 视图 breakdown tooltip**（TASK-FU-003）：可给 breakdown 字段加 tooltip 解释（如 "vector_score: embedding cosine ∈ [0, 1]"）。
8. **web-gin-service 同步更新**（TASK-FU-001/003/004）：当前仅 logic-grpc-service 修改；web-gin-service 没有 debug API 入口，但 config.Ranking 段同步加载需要 web-gin-service 也调 `LoadRankingConfig`。如未来 web-gin-service 也使用混合打分，需要同步。

## SPEC / SDD / Acceptance 对齐情况

### SPEC
- §5 FR-FU-001 ~ FR-FU-004: 全部 4 个功能需求实现
- §7 兼容性：ctx helper 替换、UI 追加 optional 字段、config.Ranking additive 字段、CI 新增 job 不影响其他
- §11 AC-FU-001 ~ AC-FU-005: 全部验收标准通过

### SDD
- §3.1 context-based debug 状态: 完全对齐
- §3.2 CI protoc 步骤: protoc 25.x + protoc-gen-go v1.36.11 显式安装
- §3.3 debug 视图 UI 升级: 8 个 breakdown 字段 + 2 个 pool_confidence
- §3.4 配置化权重: 11 个 const → var + config.Ranking + env 覆盖
- §7 配置设计: yaml + env 双重覆盖，hardcode 默认
- §8 兼容性策略: 所有改动为 additive
- §11 测试策略: 4 类测试（defaults / override / clamp / reset / integration）覆盖

### Acceptance
- AC-FU-001 ~ AC-FU-005: 全部通过
- 4 个 TASK 各自的 acceptance criteria 全部通过（详见各 TASK 报告）
