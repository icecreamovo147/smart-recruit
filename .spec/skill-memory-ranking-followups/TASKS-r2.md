# TASKS - skill-memory-ranking-followups-r2

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-FU-005 | applyRankingField warn 日志 | pending | skill_memory_ranking.go + test | acceptance/TASK-FU-005.md |
| TASK-FU-006 | debug 视图 breakdown tooltip | pending | SemanticRetrievalDebugView.vue | acceptance/TASK-FU-006.md |
| TASK-FU-007 | RankingConfig 收敛为单个 struct | pending | skill_memory_ranking.go + test | acceptance/TASK-FU-007.md |
| TASK-FU-008 | ranking-show CLI | pending | cmd/ranking-show/main.go | acceptance/TASK-FU-008.md |
| TASK-FU-009 | web-gin-service 同步 LoadRankingConfig | pending | web-gin-service/main.go | acceptance/TASK-FU-009.md |

## TASK-FU-005 - applyRankingField warn 日志

### Goal

`applyRankingField` 在 env 值越界时输出结构化 warn 日志。

### Scope

- `logic-grpc-service/service/skill_memory_ranking.go`：11 个 `applyRankingField` 调用点都传 key 名
- `logic-grpc-service/service/skill_memory_ranking_test.go`：新增 warn 日志测试

### Allowed Files

- `logic-grpc-service/service/skill_memory_ranking.go`
- `logic-grpc-service/service/skill_memory_ranking_test.go`

### Forbidden Files

- 任何 `proto/**` / `pb/**` / `config/**` / `cmd/**` / `model/**` / `repository/**` / `frontend/**` / `web-gin-service/**`

### Dependencies

- TASK-FU-004 完成（`applyRankingField` 已存在）

### Acceptance Criteria

- AC-001：`applyRankingField` 增加 `key string` 参数
- AC-002：value 越界时输出 `logger.L().Warn` 包含 key / value / clamped 字段
- AC-003：value 在 [min, max] 区间内不 warn
- AC-004：value == 0 不 warn（视为"未设置"）
- AC-005：单测覆盖 3 种 case（normal / below_min / above_max）
- AC-006：`go test ./...` 通过

### Required Tests

- `TestApplyRankingFieldWarnBelowMin`
- `TestApplyRankingFieldWarnAboveMax`
- `TestApplyRankingFieldNoWarnOnValidValue`
- `TestApplyRankingFieldNoWarnOnZero`

### Risks

- warn 日志测试需要捕获 zap 输出；可使用 `observer` 或 `testutil`。

## TASK-FU-006 - debug 视图 breakdown tooltip

### Goal

debug 视图的 6 个 breakdown 字段标签加 `el-tooltip` 解释。

### Scope

- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`：6 个字段加 tooltip

### Allowed Files

- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`

### Forbidden Files

- `hr-frontend/src/types/**` / `api/**` / `components/**` / `router/**` / `package.json` / `pnpm-lock.yaml`
- `web-gin-service/**` / `logic-grpc-service/**`

### Dependencies

- TASK-FU-003 完成

### Acceptance Criteria

- AC-001：6 个 breakdown 字段都有 tooltip
- AC-002：tooltip 文案简洁（≤ 30 字符）
- AC-003：`pnpm --filter hr-frontend typecheck` 通过
- AC-004：旧展示行为不变

### Required Tests

- typecheck 即可；无需新增测试

### Risks

- Element Plus tooltip 性能：6 个 tooltip 不会显著影响首屏。

## TASK-FU-007 - RankingConfig 收敛为单个 struct

### Goal

11 个 `var rankXxx` 替换为 `var rankingConfig` 单一 struct。

### Scope

- `logic-grpc-service/service/skill_memory_ranking.go`：重构
- `logic-grpc-service/service/skill_memory_ranking_test.go`：所有 `rankXxx` 引用改为 `rankingConfig.Xxx`

### Allowed Files

- `logic-grpc-service/service/skill_memory_ranking.go`
- `logic-grpc-service/service/skill_memory_ranking_test.go`

### Forbidden Files

- 其他文件

### Dependencies

- TASK-FU-005 完成

### Acceptance Criteria

- AC-001：11 个 `var rankXxx` 替换为 `var rankingConfig` struct
- AC-002：所有 `rankXxx` 引用改为 `rankingConfig.Xxx`
- AC-003：`LoadRankingConfig` / `ResetRankingConfigForTest` 签名不变
- AC-004：所有现有测试通过（无弱化）
- AC-005：`go test ./...` 通过

### Required Tests

- 现有测试无修改即可
- `grep -c "rank[A-Z]" skill_memory_ranking.go` 应该只剩 1 个 struct 定义 + 引用 `rankingConfig.Xxx`（无散落 var）

### Risks

- 大批量替换：约 30+ 处引用。
- 测试引用同步：约 50+ 处。

## TASK-FU-008 - ranking-show CLI

### Goal

新增 `cmd/ranking-show/main.go` 打印当前生效的 11 个权重。

### Scope

- `logic-grpc-service/cmd/ranking-show/main.go`：新增

### Allowed Files

- `logic-grpc-service/cmd/ranking-show/main.go`

### Forbidden Files

- 其他文件

### Dependencies

- TASK-FU-007 完成（`rankingConfig` 收敛后可读）

### Acceptance Criteria

- AC-001：`go run ./cmd/ranking-show/` 输出 11 行
- AC-002：每行格式 `key=value`（4 位小数）
- AC-003：env 覆盖时输出覆盖值
- AC-004：未设 env 时输出 hardcode 默认
- AC-005：`go build ./...` 通过

### Required Tests

- 手动执行验证（无需单测）
- 输出格式可被 `awk` / `grep` 解析

### Risks

- cmd 入口与 logic-grpc 共享 import 路径，需要正确 build 上下文。
- `config.Load()` 在 dev 环境依赖 yaml；CLI 需要 graceful 处理。

## TASK-FU-009 - web-gin-service 同步 LoadRankingConfig

### Goal

web-gin-service 启动时也加载 `config.Ranking` 段。

### Scope

- `web-gin-service/main.go`：追加启动加载
- 如 web-gin 依赖 logic-grpc-service：直接调 `service.LoadRankingConfig`
- 如 web-gin 不依赖：简化方案（仅读取 + 日志）

### Allowed Files

- `web-gin-service/main.go`
- `web-gin-service/cmd/**/main.go`（如有）

### Forbidden Files

- `web-gin-service/handler/**` / `router/**` / `rpc/**`
- `logic-grpc-service/**`

### Dependencies

- TASK-FU-004 完成

### Acceptance Criteria

- AC-001：web-gin 启动时调 `LoadRankingConfig`（如可依赖）或读取 `config.Ranking` 段（fallback）
- AC-002：`go build ./...` 通过（web-gin）
- AC-003：启动日志显示 `ranking` 段已加载

### Required Tests

- `go build ./...` 即可
- 启动日志存在 `[ranking] config loaded` 类似输出

### Risks

- web-gin 与 logic-grpc 的依赖关系需要先确认；如不依赖需要 fallback 方案。

## 串行执行顺序

1. TASK-FU-005（warn 日志）— 小、基础
2. TASK-FU-006（tooltip）— 前端独立
3. TASK-FU-007（收敛）— 依赖 TASK-FU-005（先 warn 后收敛）
4. TASK-FU-008（CLI）— 依赖 TASK-FU-007（收敛后才能用单一 var）
5. TASK-FU-009（web-gin）— 独立，最后做
