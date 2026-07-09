# Skill / Memory 召回排序 — 后续硬化 SPEC

## 1. 背景

`.spec/skill-memory-ranking/` 下的 8 个 TASK（TASK-001..TASK-008）已完成混合打分数据模型、Skill / Memory 池排序、top1/top2 gap 置信度、proto 字段扩展、DebugSemanticRetrieval 透传、前端类型同步、补充单测。本 SPEC 覆盖后续建议中的 4 项硬化与可观测性提升：

1. **并发安全加固**：`lastDebugPoolConfidence` / `lastDebugMemoryRankings` 是包级变量，在 `DebugSemanticRetrieval` 并发请求下会互相覆盖。
2. **CI 流水线固化**：`.github/workflows/ci.yml` 当前没有 protoc 步骤；TASK-005 用 `/tmp/protoc/bin/protoc 25.3` 重生成两份 pb，CI 必须固化 protoc 版本并对两侧 pb 做 baseline diff。
3. **debug 视图 UI 升级**：`SemanticRetrievalDebugView.vue` 只展示 `score` 一个数字，未展示 TASK-005/006 新增的 8 个 breakdown 字段与 `skill_pool_confidence` / `memory_pool_confidence`。
4. **配置化权重（SPEC out-of-scope 扩展）**：原 SPEC §13 把权重 / 阈值 hardcode；本 SPEC 扩展为 `config.Ranking` 段，可由环境变量 / yaml 覆盖，hardcode 行为保持为默认。

## 2. 目标

1. 将 `DebugSemanticRetrieval` 内部的 debug 状态从包级变量改为 `context.Context`，消除并发覆盖。
2. CI 流水线补一次 protoc 重生成 + 两侧 `recruitment.pb.go` diff 检查；protoc 版本固化。
3. `SemanticRetrievalDebugView.vue` 展示 8 个 breakdown 字段 + 顶部 2 个 pool_confidence，含 fallback 模式下的特殊展示。
4. 权重 / 阈值由 `config.Ranking` 段驱动，hardcode 行为保持为默认；提供重置 / 覆盖 helper。

## 3. 非目标

1. 不修改 TASK-001..TASK-008 已落地的算法公式与 proto 字段。
2. 不引入新依赖；config.Ranking 仅依赖 `config.Config` 现有 env 解析机制。
3. 不修改 `selectAgentSkills` / `selectAgentSkillsWithSemantic` / `rankMemories` / `DebugSemanticRetrieval` 等公开函数签名。
4. 不在 debug 视图引入 A/B 实验、阈值调参 UI（仅展示新指标）。
5. 不在生产侧引入热加载；本期只支持启动时从 env / yaml 解析。
6. 不修改 CI 的其他 job（go-test / frontend / secret-scan）；只新增 protoc 相关步骤。

## 4. 用户角色与使用场景

- **HR 业务用户**：在 debug 视图直接看到当前 Skill / Memory 排序的 breakdown 与置信度，无需查日志。
- **运维 / 研发**：通过 CI pipeline 的 protoc diff 检查发现 proto 字段被无意修改；通过 config 调整权重做 AB 实验。
- **算法 / 产品**：通过 `config.example.yaml` 注释理解权重含义；通过 `RANKING_*` 环境变量快速调整阈值。

## 5. 功能需求

### FR-FU-001：context-based debug 状态

- 描述：把 `agent_skill_service.go` 内的 `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级变量改为 `context.Context` 值；key 类型私有。
- 输入：`ctx context.Context` + `debugPoolConfidenceView` / `[]RankedMemoryItem`。
- 输出：`ctx context.Context` 携带状态；helper `getDebugPoolConfidenceFromContext(ctx) debugPoolConfidenceView` 读取。
- 边界：
  - context 中无 key 时返回 zero value（`SkillPoolConfidence: none, MemoryPoolConfidence: none`）。
  - 不影响生产行为；仅 debug 路径使用。

### FR-FU-002：CI 流水线 protoc 步骤

- 描述：在 `.github/workflows/ci.yml` 增加一个 `proto-lint` job，使用与本地一致的 protoc 25.x 重生成两份 `recruitment.pb.go` 并与 committed 版本做 diff。
- 输入：committed `recruitment.pb.go`、当前 `.proto` 文件。
- 输出：通过 / 失败；失败时输出 diff 路径。
- 边界：
  - 不修改现有 `go-test` / `frontend` / `secret-scan` job。
  - 步骤中显式安装 protoc 25.x + protoc-gen-go；缓存 `$GOPATH/bin` 加速。

### FR-FU-003：debug 视图 UI 升级

- 描述：在 `SemanticRetrievalDebugView.vue` 中：
  - 顶部 stats 区域追加 2 个新指标：`Skill 池置信度` / `Memory 池置信度`（高亮色与 fallback 警告）。
  - Skill / Memory 每个 item 展开 "查看详情" 时展示 breakdown 表格（vector / lexical / metadata / relevance / business_boost / final_rank_score / relevance_mode / pool_rank）。
  - 卡片角标用 relevance_mode / pool_rank 表达 `vector_lexical_metadata` / `lexical_metadata` 模式与排名。
- 边界：
  - 旧 `score` 字段展示保留（兼容旧用户）。
  - 不引入新组件库；继续用 Element Plus。
  - 不修改 TypeScript 类型（`agentSkill.ts` 已在 TASK-007 完成）。

### FR-FU-004：配置化权重

- 描述：在 `logic-grpc-service/config/config.go` 引入 `Ranking` struct；通过环境变量 `RANKING_WEIGHT_VECTOR` / `RANKING_BUSINESS_BOOST_MAX` / `RANKING_RELEVANCE_GATE` 等 11 个 key 覆盖权重。
- 默认值与 SDD §3.3 完全一致；空值时使用默认。
- `skill_memory_ranking.go` 的 `const` 改为 `var`（默认值），新增 `loadRankingConfig(cfg config.Config)` 函数在启动时调用。
- 边界：
  - 启动后权重不可热修改（仅 env / yaml 启动时）。
  - `config.example.yaml` 增加 `ranking` 段注释。
  - 单测 `TestResetRankingConfig` 验证 helper。

## 6. 非功能需求

- **性能**：context 读写 O(1)；config 加载 O(1) 启动时一次。
- **稳定性**：现有 `go test ./...` / `pnpm --filter hr-frontend typecheck` 全部通过。
- **可观测性**：CI protoc 步骤失败时清晰指出 diff 行号；debug 视图的 pool_confidence 与 breakdown 在 fallback 模式下正确显示。
- **可维护性**：权重配置集中在 `config.Ranking` 与 `skill_memory_ranking.go` 顶部；变更不需翻多处。

## 7. 兼容性需求

- `config.Config` 既有字段保持；新增 `Ranking` 段是 additive。
- `DebugSemanticRetrieval` / `RankSkillCandidates` / `RankMemoryCandidates` 公开签名不变。
- 现有 4 个包级变量 `lastDebugPoolConfidence` / `lastDebugMemoryRankings` / `stashDebugPoolConfidence` / `getLastDebugMemoryRankings` 在本 SPEC 内替换为 context 实现；调用方（TASK-004 / TASK-006 写入端、TASK-006 读取端）同步替换。
- CI 新增 job 不影响其他 job 的 concurrency / timeout / secret。
- debug 视图已有结构保留；breakdown 表格以"展开详情"方式展示，不强制展开。

## 8. 可观测性需求

- CI protoc 步骤：失败时输出 protoc 版本与 diff 行号。
- context 状态：`stashDebugPoolConfidence(ctx, v)` 接受 ctx，debug 日志可读 ctx 携带的状态。
- config 加载：`logger.L().Info("[ranking] config loaded", zap.Float64s("weights", ...))`。

## 9. 异常处理与降级

| 场景 | 行为 |
| --- | --- |
| env 解析失败（如 RANKING_WEIGHT_VECTOR 非数字） | 回退到 hardcode 默认；warn 日志 |
| env 解析越界（如 RANKING_BUSINESS_BOOST_MAX=2.0） | clamp 到 [1.0, 1.5]；warn 日志 |
| context 携带 nil | zero value |
| CI 缺少 protoc | 步骤显式 `apt-get install -y protobuf-compiler=25.x` 或 `go install`；失败时返回非 0 |

## 10. 安全需求

- env 解析不输出原始字符串（避免 log 注入）。
- debug 视图不展示完整 Memory content / Skill markdown（已遵守原 SPEC §10）。
- CI 步骤不引入第三方 action；仅用 `actions/checkout` / `actions/setup-go` / 官方 `apt`。

## 11. 验收标准

- AC-FU-001：context 替换完成；并发测试不覆盖。
- AC-FU-002：`.github/workflows/ci.yml` 包含 `proto-lint` job，protoc 版本 25.x。
- AC-FU-003：`SemanticRetrievalDebugView.vue` 展示新 breakdown + pool_confidence；`pnpm --filter hr-frontend typecheck` 通过。
- AC-FU-004：`config.Ranking` 段在 env 覆盖与 hardcode 默认下行为一致；`go test ./...` 通过。
- AC-FU-005：4 个 TASK 都通过自审，无回归。

## 12. 不在本期范围

- 权重热加载（运行时改 config）。
- 召回策略的 A/B 实验框架。
- proto 文件 lint / breaking-change 检查（buf 工具链）。
- 完整前端 UI 设计系统升级。

## 13. 待确认问题

- 11 个 env 变量名是否合理（建议 `RANKING_<NAME>`，如 `RANKING_WEIGHT_VECTOR`）。
- `config.example.yaml` 的 ranking 段是否需要与 `Agent` 段同级，还是独立顶层段。

## 14. 开放问题

- 是否提供 `make ranking-show` 命令行工具打印当前生效的权重。
- 是否在 `pkg/logger` 增加 `[ranking]` tag，便于运维筛选。
