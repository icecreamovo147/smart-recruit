# TASKS - skill-memory-ranking-followups

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-FU-001 | context-based debug 状态 | pending | agent_skill_service.go | acceptance/TASK-FU-001.md |
| TASK-FU-002 | CI 流水线 protoc 步骤 | pending | .github/workflows/ci.yml | acceptance/TASK-FU-002.md |
| TASK-FU-003 | debug 视图 UI 升级 | pending | SemanticRetrievalDebugView.vue | acceptance/TASK-FU-003.md |
| TASK-FU-004 | 配置化权重（config.Ranking） | pending | config/config.go + skill_memory_ranking.go | acceptance/TASK-FU-004.md |

## TASK-FU-001 - context-based debug 状态

### Goal

把 `agent_skill_service.go` 内 `lastDebugPoolConfidence` / `lastDebugMemoryRankings` 包级变量改为 `context.Context` 私有 key 传递；消除 `DebugSemanticRetrieval` 并发请求下的状态污染。

### Scope

- 修改 `agent_skill_service.go`：删除 2 个 `var` + 4 个 helper；增加 4 个 ctx helper。
- 修改 `agent_skill_service_test.go`：调整 2 个 TASK-006 既有测试的读取方式（最小改动）。
- 不修改其他 service 文件、不修改 proto / pb / config / 前端。

### Allowed Files

- `logic-grpc-service/service/agent_skill_service.go`
- `logic-grpc-service/service/agent_skill_service_test.go`

### Forbidden Files

- 任何 `proto/**` / `pb/**` / `config/**` / `model/**` / `repository/**` / `frontend/**` / `web-gin-service/**`

### Dependencies

- TASK-004 / TASK-006 写入端需同步替换（`computeDebugPoolConfidence` / `stashLastDebugMemoryRankings` 调用点）。

### Acceptance Criteria

- AC-001：包级 `var lastDebugPoolConfidence` / `var lastDebugMemoryRankings` 删除。
- AC-002：新增 `withDebugPoolConfidence` / `getDebugPoolConfidenceFromContext` / `withDebugMemoryRankings` / `getDebugMemoryRankingsFromContext` 4 个 helper。
- AC-003：`DebugSemanticRetrieval` 内部用 ctx helper 传递状态。
- AC-004：`computeDebugPoolConfidence` 接受 ctx 参数；`semanticDebugMemories` 内部用 ctx helper 写入。
- AC-005：现有 2 个 TASK-006 测试 `TestAgentSkillServiceDebugSemanticRetrievalPopulatesScoreBreakdown` / `...FallbackPopulatesBreakdown` 通过。
- AC-006：`go test ./...` 通过。

### Required Tests

- `TestDebugContextStateIsolation`：验证两个并发 ctx 互相不污染。
- `TestDebugContextStateEmptyDefaults`：验证 nil ctx / 缺失 key 时返回 zero value。

### Risks

- 旧 `lastDebugPoolConfidenceForTest` 等导出 helper 会被删除；如果未来有外部测试依赖，需要单独处理。

## TASK-FU-002 - CI 流水线 protoc 步骤

### Goal

在 `.github/workflows/ci.yml` 新增 `proto-lint` job，protoc 25.x 重生成两侧 `recruitment.pb.go` 并与 committed diff 必须为空；protoc 版本与本地一致。

### Scope

- 修改 `.github/workflows/ci.yml`：增加 `proto-lint` job。
- 不修改其他 job；不影响 go-test / frontend / secret-scan。

### Allowed Files

- `.github/workflows/ci.yml`

### Forbidden Files

- 其他 workflows 文件 / `.github/**` 配置
- `proto/**` / `pb/**`（CI 自检对象，不允许直接修改）

### Dependencies

- 无。

### Acceptance Criteria

- AC-001：`.github/workflows/ci.yml` 包含 `proto-lint` job。
- AC-002：protoc 25.x 显式安装。
- AC-003：protoc-gen-go 安装并缓存。
- AC-004：重生成两侧 pb 的步骤与本地 TASK-005 完全一致。
- AC-005：`git diff --exit-code` 对两侧 pb 必须为空；失败时输出 diff 行号。
- AC-006：不修改现有 3 个 job。

### Required Tests

- 本地验证：在干净 working tree 下 `bash .github/workflows/proto-lint.sh`（如新增）应 exit 0。

### Risks

- protoc 安装步骤在 ubuntu-latest 镜像上的稳定性；`wget` 镜像地址变更需要更新。
- 旧 `apt-get install -y protobuf-compiler` 提供 3.x 版本，步骤必须显式 wget 25.x。

## TASK-FU-003 - debug 视图 UI 升级

### Goal

在 `SemanticRetrievalDebugView.vue` 展示 TASK-005/006 新增的 8 个 breakdown 字段（vector / lexical / metadata / relevance / business_boost / final_rank_score / relevance_mode / pool_rank）以及顶部 2 个 pool_confidence（skill_pool_confidence / memory_pool_confidence）。

### Scope

- 修改 `SemanticRetrievalDebugView.vue`：
  - 顶部 stats 区域追加 2 个 `console-stat`（pool_confidence）
  - Skill / Memory item 展开区追加 breakdown 表格
  - 卡片角标添加 `relevance_mode` tag
- 不修改 TypeScript 类型、不修改 API 层、不引入新组件。

### Allowed Files

- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`

### Forbidden Files

- `hr-frontend/src/types/**`（TASK-007 已完成）
- `hr-frontend/src/api/**`
- `hr-frontend/src/components/**`
- 其他 frontend 目录
- 后端代码 / proto / pb

### Dependencies

- TASK-005 / TASK-006 / TASK-007 已完成。

### Acceptance Criteria

- AC-001：顶部 stats 展示 `Skill 池置信度` / `Memory 池置信度`，使用 `el-tag` 区分 high/medium/low/none。
- AC-002：Skill item 展开区显示 8 个 breakdown 字段。
- AC-003：Memory item 展开区显示 8 个 breakdown 字段。
- AC-004：卡片角标显示 `relevance_mode` tag。
- AC-005：`pnpm --filter hr-frontend typecheck` 通过。
- AC-006：旧 `score` 字段展示保留。

### Required Tests

- 本地 typecheck 通过；可选 `pnpm --filter hr-frontend test`（如已有 Vue Test）。

### Risks

- 视觉风格与现有 `debug-detail-block` 冲突；需保持 monospace 字体与紧凑布局。
- 表格行数较多，移动端布局需保持可读。

## TASK-FU-004 - 配置化权重（config.Ranking）

### Goal

把 `skill_memory_ranking.go` 顶部 11 个 `const` 权重 / 阈值改为 `var`，启动时从 `config.Ranking` 段读取；提供 `RANKING_*` env 覆盖；hardcode 行为保持为默认。

### Scope

- 修改 `config/config.go`：增加 `Ranking` struct + 11 个 `yaml` / `env` tag。
- 修改 `config/config.example.yaml`：增加 `ranking` 段。
- 修改 `skill_memory_ranking.go`：`const` → `var` + `LoadRankingConfig(cfg config.Config)` + `ResetRankingConfigForTest()`。
- 修改 `cmd/.../main.go`（如有）：启动时调 `LoadRankingConfig`。
- 修改 `skill_memory_ranking_test.go`：增加 config 相关测试。
- 不修改算法公式；不修改 proto / pb / 前端。

### Allowed Files

- `logic-grpc-service/config/config.go`
- `logic-grpc-service/config/config.example.yaml`
- `logic-grpc-service/service/skill_memory_ranking.go`
- `logic-grpc-service/service/skill_memory_ranking_test.go`
- `logic-grpc-service/cmd/**/main.go`（如有 main 启动入口）

### Forbidden Files

- `proto/**` / `pb/**` / `web-gin-service/**` / `hr-frontend/**` / `model/**` / `repository/**`

### Dependencies

- 无。

### Acceptance Criteria

- AC-001：`config.Ranking` 段定义完整；11 个字段都有 yaml + env tag。
- AC-002：`LoadRankingConfig` 在 nil / 空 / 部分填充 / 越界 4 种输入下行为正确。
- AC-003：`ResetRankingConfigForTest` 恢复 11 个 var 到 hardcode 默认。
- AC-004：`go test ./...` 全部通过。
- AC-005：现有 5 个 TASK-001..004 既有测试（如 `TestComputeRelevanceScore` / `TestComputePoolConfidence`）通过。
- AC-006：`config.example.yaml` 增加 `ranking` 段注释。

### Required Tests

- `TestLoadRankingConfigDefaults`：nil cfg 走 hardcode。
- `TestLoadRankingConfigOverride`：env 覆盖生效。
- `TestLoadRankingConfigClamp`：越界 clamp 到合法范围。
- `TestResetRankingConfigForTest`：reset 后 11 个 var 等于 hardcode。

### Risks

- 启动时一次性 init；如果 `config.Config` 不可用（早期启动）需 nil-safe。
- 旧 const 引用方（如 `rankBusinessBoostMax`）改为 `var` 后可被运行时修改，需要测试用 `ResetRankingConfigForTest()` 隔离。

## 总实现边界

- 不修改 `proto/**` / `pb/**`（TASK-005 已完成；CI 自检）
- 不修改前端类型 / API（TASK-007 已完成）
- 不修改 TASK-001..008 的算法公式
- 不引入新第三方依赖
- 不修改 deploy / docker / 启动脚本（除 cmd/main.go）

## 串行执行顺序

1. TASK-FU-001（context-based debug 状态）— 小、基础
2. TASK-FU-002（CI protoc 步骤）— 与 1 独立，可并行；建议按 spec-harness 串行
3. TASK-FU-003（UI 升级）— 与 1 / 2 独立，前端
4. TASK-FU-004（配置化权重）— 与 1 / 2 / 3 独立；最后做以避免 config 调整影响其他 TASK 的测试
