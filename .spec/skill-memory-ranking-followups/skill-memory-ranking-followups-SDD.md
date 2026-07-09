# Skill / Memory 召回排序 — 后续硬化 SDD

## 1. 现有架构摘要

### 1.1 已落地的实现（TASK-001..008）

- `logic-grpc-service/service/skill_memory_ranking.go`：权重常量、纯函数 helper、Skill / Memory 打分函数
- `logic-grpc-service/service/agent_skill_selector.go` / `agent_context.go`：rankMemories / selectAgentSkillsWithSemantic 走混合打分
- `logic-grpc-service/service/agent_skill_service.go`：`DebugSemanticRetrieval` 填充 proto 新字段 + `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级变量
- `logic-grpc-service/recruitment/pb/recruitment.pb.go`（含 web-gin 同步）：含 18 个新 breakdown + pool_confidence 字段
- `hr-frontend/src/types/agentSkill.ts`：追加 18 个 optional 字段
- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`：尚未展示新字段

### 1.2 本 SDD 覆盖的 4 个改动

1. **并发安全**：context-based debug 状态
2. **CI 流水线**：protoc 25.x + pb diff 检查
3. **debug 视图**：UI 升级展示 breakdown + pool_confidence
4. **配置化权重**：从 hardcode 改为 config.Ranking 段

## 2. 问题分析

| 编号 | 问题 | 影响 |
| --- | --- | --- |
| FU-P-1 | `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 是包级变量；`DebugSemanticRetrieval` 并发请求下会被互相覆盖 | debug 数据污染；生产误用可能误导排查 |
| FU-P-2 | `.github/workflows/ci.yml` 无 protoc 步骤；TASK-005 用 `/tmp/protoc/bin/protoc 25.3` 重生成 11000+ 行 diff | proto 字段被无意修改无 CI 拦截 |
| FU-P-3 | `SemanticRetrievalDebugView.vue` 只展示 `score` 一个数字；新 breakdown 字段不展示 | 用户无法理解新排序；运营 / 排查仍依赖日志 |
| FU-P-4 | 权重 / 阈值 hardcode 在 `skill_memory_ranking.go` 顶部 11 个 `const` | 调整需改代码 + 重新发布；无法热调做 AB |

## 3. 提议设计

### 3.1 context-based debug 状态

- 私有 key 类型 `debugContextKey struct{}`；
- helper 函数：
  - `withDebugPoolConfidence(ctx, v) context.Context` — 写入
  - `getDebugPoolConfidenceFromContext(ctx) debugPoolConfidenceView` — 读取
  - `withDebugMemoryRankings(ctx, items) context.Context` — 写入
  - `getDebugMemoryRankingsFromContext(ctx) []RankedMemoryItem` — 读取
- `DebugSemanticRetrieval` 入口把 `ctx` 透传给 `semanticDebugMemories` / `semanticDebugSkillsToPB`；helper 内部读写 ctx。
- 删除 `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级变量 + `stashDebugPoolConfidence` / `lastDebugPoolConfidenceForTest` / `stashLastDebugMemoryRankings` / `getLastDebugMemoryRankings` 旧函数。

### 3.2 CI protoc 步骤

- 新增 `proto-lint` job：
  - runner: `ubuntu-latest`
  - 步骤：
    1. `actions/checkout@v4`
    2. `actions/setup-go@v5`（与 go-test 复用 cache）
    3. 安装 protoc 25.x：
       ```bash
       sudo apt-get update
       sudo apt-get install -y wget unzip
       wget https://github.com/protocolbuffers/protobuf/releases/download/v25.3/protoc-25.3-linux-x86_64.zip
       unzip protoc-25.3-linux-x86_64.zip -d /usr/local
       ```
    4. 安装 protoc-gen-go：`go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11`
    5. 重生成两侧 pb：
       ```bash
       cd logic-grpc-service
       protoc --proto_path=proto --go_out=recruitment/pb --go_opt=paths=source_relative --go_opt=Mrecruitment.proto=recruitment/pb recruitment.proto
       cp recruitment/pb/recruitment.pb.go ../web-gin-service/recruitment/pb/recruitment.pb.go
       ```
    6. diff 检查：
       ```bash
       git diff --exit-code logic-grpc-service/recruitment/pb/recruitment.pb.go web-gin-service/recruitment/pb/recruitment.pb.go
       ```
- 触发条件：`on.push` 与 `on.pull_request` 沿用现有配置；不影响其他 job。

### 3.3 debug 视图 UI 升级

- **顶部 stats 区域追加 2 个指标**：
  - `Skill 池置信度`：`{{ result.skill_pool_confidence || '-' }}`，用 `el-tag` 显示，色彩对应 high/medium/low/none。
  - `Memory 池置信度`：同上。
- **每个 item 展开区追加 breakdown 表格**：
  - vector_score / lexical_score / metadata_score
  - relevance_score / business_boost / final_rank_score
  - relevance_mode（显示为 tag：`vector_lexical_metadata` / `lexical_metadata`）
  - pool_rank（badge）
- **卡片角标**：把 `formatScore(item.score)` 改为同时显示 `final_rank_score` 与 `relevance_mode` 的小 tag。
- **样式**：复用现有 `.debug-detail-block` 网格布局；breakdown 用 monospace 字体显示数字。

### 3.4 配置化权重

- `config.Ranking` 结构：
  ```go
  type Ranking struct {
      WeightVector     float64 `yaml:"weight_vector" env:"RANKING_WEIGHT_VECTOR"`
      WeightLexical    float64 `yaml:"weight_lexical" env:"RANKING_WEIGHT_LEXICAL"`
      WeightMetadata   float64 `yaml:"weight_metadata" env:"RANKING_WEIGHT_METADATA"`
      BusinessBoostMax float64 `yaml:"business_boost_max" env:"RANKING_BUSINESS_BOOST_MAX"`
      PriorityNorm     float64 `yaml:"priority_norm" env:"RANKING_PRIORITY_NORM"`
      BoostAlpha       float64 `yaml:"boost_alpha" env:"RANKING_BOOST_ALPHA"`
      BoostBeta        float64 `yaml:"boost_beta" env:"RANKING_BOOST_BETA"`
      BoostGamma       float64 `yaml:"boost_gamma" env:"RANKING_BOOST_GAMMA"`
      RelevanceGate    float64 `yaml:"relevance_gate" env:"RANKING_RELEVANCE_GATE"`
      GapHigh          float64 `yaml:"gap_high" env:"RANKING_GAP_HIGH"`
      GapMedium        float64 `yaml:"gap_medium" env:"RANKING_GAP_MEDIUM"`
  }
  ```
- 默认值常量（`skill_memory_ranking.go` 顶部）：
  ```go
  var (
      rankWeightVector     = 0.6
      rankWeightLexical    = 0.3
      ...
  )
  ```
- `LoadRankingConfig(cfg config.Config) Ranking`：启动时调用；用 `cfg.GetFloat64("ranking.weight_vector")` 等读取；空值用默认；越界 clamp。
- `loadRankingConfig(cfg config.Config)`：把 `Ranking` 写入 package-level `var`。
- helper `ResetRankingConfigForTest()`：测试时把 var 恢复默认。

## 4. 数据结构变化

- `config.Config` 新增 `Ranking Ranking` 字段（additive）
- `agent_skill_service.go` 删除 2 个 `var`，增加 4 个 `ctx` helper
- `SemanticRetrievalDebugView.vue` 模板增加 ~30 行 breakdown UI
- `.github/workflows/ci.yml` 增加 `proto-lint` job

## 5. API 与接口变化

| 接口 | 变化 |
| --- | --- |
| `DebugSemanticRetrieval` | 签名不变；内部走 ctx |
| `semanticDebugMemories` / `semanticDebugSkillsToPB` | 签名不变；内部从 ctx 读 |
| `config.Config` | 字段增加（additive） |
| `config.example.yaml` | 增加 `ranking` 段 |
| CI yaml | 增加新 job（不影响其他 job） |
| 前端类型 | 不变（TASK-007 已完成） |
| 前端视图 | 模板增加 breakdown UI |

## 6. 算法与工作流变化

| 阶段 | 旧实现 | 新实现 |
| --- | --- | --- |
| Debug 状态存储 | 包级 var | ctx value |
| 权重加载 | hardcode const | config.Ranking 启动时载入 |
| protoc CI | 无 | 自动重生成 + diff 检查 |
| UI breakdown | 仅 score | score + 8 个 breakdown + pool_confidence |

## 7. 配置设计

- `config.example.yaml` 增加：
  ```yaml
  ranking:
    weight_vector: 0.6
    weight_lexical: 0.3
    ...
  ```
- env 变量：`RANKING_WEIGHT_VECTOR` / `RANKING_BUSINESS_BOOST_MAX` 等 11 个 key。
- 启动时 `config.LoadConfig` 解析；空值用 hardcode 默认。

## 8. 兼容性策略

1. `config.Ranking` 字段为 additive；空值时退回到 hardcode 默认。
2. ctx helper：未携带时返回 zero value，与旧"全局最后值"行为不兼容；TASK-006 调用方同步替换。
3. proto 字段不变（TASK-005 已完成）。
4. 前端类型不变（TASK-007 已完成）。
5. CI 旧 job 全部保留；新 job 失败不阻塞其他 job（除非显式 `needs`）。

## 9. 错误处理与降级

| 触发 | 行为 |
| --- | --- |
| env 解析失败（非数字） | warn 日志 + 用 hardcode 默认 |
| 越界 | clamp 到合法范围 + warn 日志 |
| ctx 缺失 | zero value（与"无 debug 状态"等价） |
| protoc 不可用 | CI 步骤 fail + 提示安装方式 |
| proto diff | 步骤 fail + 输出 diff 行号 |

## 10. 可观测性设计

- 启动日志：`logger.L().Info("[ranking] config loaded", zap.Float64("weight_vector", rankWeightVector), ...)`
- 越界 warn：`logger.L().Warn("[ranking] env value out of range, clamped", zap.String("key", "..."), ...)`
- debug 视图：在 pool_confidence 旁边显示 icon（high → check / medium → info / low / none → warning）。
- CI：失败时输出 protoc 版本 + diff 行号。

## 11. 测试策略

### 单元测试
- `TestContextDebugState`：验证 ctx helper 读写与隔离。
- `TestLoadRankingConfig`：验证默认值 + env 覆盖 + 越界 clamp。
- `TestResetRankingConfigForTest`：测试 helper。

### CI 集成测试
- `proto-lint` job 在 PR 中自动跑，protoc 25.x 重生成两侧 pb，与 committed diff 必须为空。

### 兼容性测试
- 现有 `TestSelectAgentSkills*` / `TestAgentContextBuilderMemoryRecall*` / `TestAgentSkillServiceDebugSemanticRetrieval*` 全部不回归。

### 前端
- `pnpm --filter hr-frontend typecheck` 通过。

## 12. 迁移风险

| 风险 | 等级 | 缓解 |
| --- | --- | --- |
| ctx 替换：旧测试可能依赖包级 var | 中 | TASK-006 的 2 个测试需要小幅调整读取方式 |
| config 加载：env 越界 clamp 隐藏错误 | 低 | warn 日志 + 启动时一次性提示 |
| protoc CI：与团队实际版本不一致 | 低 | 步骤内显式安装 25.x |
| UI 改动：与现有 `formatScore` / `debug-detail-block` 风格冲突 | 低 | 复用现有 class |

## 13. 实现边界

| 模块 | 是否本期修改 |
| --- | --- |
| `logic-grpc-service/service/skill_memory_ranking.go` | 改：const → var + LoadRankingConfig + ResetRankingConfig |
| `logic-grpc-service/service/agent_skill_service.go` | 改：包级 var → ctx helper |
| `logic-grpc-service/service/agent_skill_service_test.go` | 改：context-based 读取（最小调整） |
| `logic-grpc-service/config/config.go` | 改：增加 Ranking 段 |
| `logic-grpc-service/config/config.example.yaml` | 改：增加 ranking 注释 |
| `logic-grpc-service/cmd/**/main.go` | 改：启动时调 LoadRankingConfig |
| `.github/workflows/ci.yml` | 改：增加 proto-lint job |
| `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` | 改：增加 breakdown UI |
| `hr-frontend/src/types/agentSkill.ts` | 不变（TASK-007 已完成） |
| `proto/**` | 不变 |
| `pb/**` | 不变（CI 自检） |
| `web-gin-service/**` | 不变 |
| `deploy/**` / `docker/**` | 不变 |
| `docs/**` | 不变 |

## 14. 备选方案

### 14.1 sync.Mutex 保护包级 var

- 优点：改动小。
- 缺点：与 `ctx` 模式相比不易追踪；debug 状态是 request-scoped，ctx 模式更合适。

### 14.2 config 重载机制

- 优点：AB 实验友好。
- 缺点：本期不需要；增加复杂度。

### 14.3 buf lint 替代 protoc 重生成

- 优点：行业标准。
- 缺点：引入新依赖；团队 CI 未使用。

## 15. 待确认问题

- `RANKING_*` env 命名是否与团队其他 env 规范一致。
- `config.example.yaml` 的 `ranking` 段是否需要放在 `agent` 段下。
- CI protoc 步骤：是否需要 cache `apt-get` 或 protoc binary 加速。

## 16. 开放问题

- 是否提供 `make ranking-show` 命令行工具。
- 是否在 `pkg/logger` 增加 `[ranking]` tag。
- proto-lint 步骤是否扩展为 lint + breaking-change 检查（需要 buf）。
