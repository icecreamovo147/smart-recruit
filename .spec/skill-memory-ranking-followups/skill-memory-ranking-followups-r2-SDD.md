# Skill / Memory 召回排序 — 第二轮硬化 SDD

## 1. 现有架构摘要

### 1.1 已落地的实现（TASK-001..008 + TASK-FU-001..004）

- `logic-grpc-service/service/skill_memory_ranking.go`：11 个 `var` + 纯函数 + 4 个核心打分函数
- `logic-grpc-service/service/agent_skill_service.go`：ctx-based debug state
- `logic-grpc-service/config/config.go`：`Ranking` struct + 11 个 env 覆盖
- `logic-grpc-service/main.go`：启动时 `LoadRankingConfig`
- `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue`：breakdown + pool_confidence 展示
- `.github/workflows/ci.yml`：proto-lint job

### 1.2 本 SDD 覆盖的 5 个改动

1. warn 日志（`applyRankingField`）
2. debug 视图 breakdown tooltip
3. RankingConfig 收敛
4. ranking-show CLI
5. web-gin-service 同步

## 2. 问题分析

| 编号 | 问题 | 影响 |
| --- | --- | --- |
| R2-P-1 | `applyRankingField` 静默 clamp，越界无告警 | 运维调权重时不知道 env 写错 |
| R2-P-2 | debug 视图 breakdown 字段无解释 | 业务用户看到 6 个数字不知道含义 |
| R2-P-3 | 11 个 `var rankXxx` 散落命名空间 | 代码 review 不友好；扩展困难 |
| R2-P-4 | 没有 CLI 工具打印当前生效权重 | 运维需要 grep 代码或写脚本 |
| R2-P-5 | web-gin 启动不加载 `config.Ranking` | 后续如果 web-gin 引入排序逻辑会读到旧值 |

## 3. 提议设计

### 3.1 warn 日志

- 修改 `applyRankingField`：
  ```go
  func applyRankingField(target *float64, value, min, max float64, key string) {
      if value == 0 {
          return
      }
      if value < min {
          logger.L().Warn("[ranking] env value out of range, clamped",
              zap.String("key", key),
              zap.Float64("value", value),
              zap.Float64("clamped", min),
          )
          *target = min
          return
      }
      if value > max {
          logger.L().Warn("[ranking] env value out of range, clamped",
              zap.String("key", key),
              zap.Float64("value", value),
              zap.Float64("clamped", max),
          )
          *target = max
          return
      }
      *target = value
  }
  ```
- 11 个调用点都传 key 名（"weight_vector" / "business_boost_max" 等）。
- 引入 `service` 包对 `pkg/logger` 的依赖（已存在）。

### 3.2 debug 视图 tooltip

- 在 `.debug-breakdown__grid` 中，每个 `<div><span>label</span>...</div>` 的 `<span>` 加 `el-tooltip`。
- tooltip 文案：
  - `vector`：`embedding cosine ∈ [0, 1]`
  - `lexical`：`keyword hit ratio ∈ [0, 1]`
  - `metadata`：`scope / category hit ∈ [0, 1]`
  - `relevance`：`weighted sum ∈ [0, 1]`
  - `business boost`：`relevance × boost ∈ [0, 1.5]`
  - `final rank`：`final_rank_score ∈ [0, 1.5]`
- 边界：使用 Element Plus 内置 `el-tooltip` 组件，不引入新依赖。

### 3.3 RankingConfig 收敛

- 当前：
  ```go
  var (
      rankWeightVector   = rankingDefaults.WeightVector
      rankWeightLexical  = rankingDefaults.WeightLexical
      // ... 11 个 var
  )
  ```
- 新：
  ```go
  var rankingConfig = struct {
      WeightVector     float64
      WeightLexical    float64
      // ... 11 字段
  }{
      WeightVector:     rankingDefaults.WeightVector,
      WeightLexical:    rankingDefaults.WeightLexical,
      // ... 11 字段初值
  }
  ```
- 所有 `rankXxx` 引用替换为 `rankingConfig.Xxx`。
- `LoadRankingConfig` 写入 `rankingConfig` struct（不需变 key / 11 个 applyRankingField 调用）。
- `ResetRankingConfigForTest` 恢复 `rankingConfig` 整体到 `rankingDefaults` 副本。
- 测试文件中的 `rankXxx` 引用同步更新。

### 3.4 ranking-show CLI

- 新文件 `logic-grpc-service/cmd/ranking-show/main.go`：
  ```go
  package main

  import (
      "fmt"
      "logic-grpc-service/config"
      "logic-grpc-service/service"
  )

  func main() {
      cfg, _ := config.Load() // ignore error for dev CLI
      service.LoadRankingConfig(service.RankingConfig{
          WeightVector:     cfg.Ranking.WeightVector,
          // ... 11 fields
      })
      // print 11 lines
      fmt.Printf("weight_vector=%.4f\n", rankCurrent(rankWeightVector))
      // ... 11 lines
  }
  ```
- 边界：
  - 不读真实 `rankingConfig` struct（避免 import cycle）；读 11 个 `rankXxx` 变量。
  - 输出格式：`key=value` 单行；脚本友好。
- 验证：`go run ./cmd/ranking-show/` 输出 11 行。

### 3.5 web-gin-service 同步

- 检查 `web-gin-service/go.mod`：是否已依赖 `logic-grpc-service`？
- 如已依赖：在 `web-gin-service/main.go` 启动时调 `service.LoadRankingConfig(...)`。
- 如未依赖：通过 yaml 段读取（不调 LoadRankingConfig，但保证配置项可见）。
- 简化方案：在 web-gin 端只读取 `config.Ranking` 段（不调 LoadRankingConfig），并打印日志确认配置已加载。

## 4. 数据结构变化

- `skill_memory_ranking.go`：11 个 `var` → 1 个 `var rankingConfig` struct
- `cmd/ranking-show/main.go`：新增
- `web-gin-service/main.go`：追加 `LoadRankingConfig` 调用

## 5. API 与接口变化

| 接口 | 变化 |
| --- | --- |
| `applyRankingField` | 增加 `key string` 参数 |
| `LoadRankingConfig` | 签名不变；写入 `rankingConfig` struct |
| `ResetRankingConfigForTest` | 签名不变；恢复整个 struct |
| `cmd/ranking-show` | 新增 CLI |
| `web-gin-service/main.go` | 启动时追加 `LoadRankingConfig` |
| debug 视图 | 6 个 breakdown 字段加 tooltip |

## 6. 算法与工作流变化

| 阶段 | 旧实现 | 新实现 |
| --- | --- | --- |
| 权重加载 | 11 个 `var` | 1 个 `rankingConfig` struct |
| 越界处理 | 静默 clamp | warn + clamp |
| 权重查看 | grep 代码 | `cmd/ranking-show` |
| breakdown 字段 | 数字 | 数字 + tooltip |

## 7. 配置设计

- 11 个 `RANKING_*` env 变量不变。
- `config.Ranking` 段 yaml 字段不变。
- 越界值：clamp + warn。

## 8. 兼容性策略

1. 11 个 var → 1 个 struct：`skill_memory_ranking.go` 是包内文件，所有引用都在包内。包外不受影响。
2. `applyRankingField` 增加 `key` 参数是包内 helper，调用点同步更新。
3. `LoadRankingConfig` / `ResetRankingConfigForTest` 公开签名不变。
4. debug 视图 tooltip 是 additive。
5. `cmd/ranking-show` 是新子命令。
6. web-gin 启动同步是 additive。

## 9. 错误处理与降级

| 触发 | 行为 |
| --- | --- |
| env 越界 | warn + clamp |
| env 解析失败 | 保留 target 原值（已有行为） |
| web-gin 不依赖 logic-grpc-service | 仅读取 config.Ranking 段；不调 LoadRankingConfig |
| cmd/ranking-show config 加载失败 | exit 0 + 打印 hardcode 默认 |

## 10. 可观测性设计

- warn 日志：`[ranking] env value out of range, clamped` + key / value / clamped
- cmd/ranking-show：`key=value` 11 行
- debug 视图 tooltip：hover 显示字段含义

## 11. 测试策略

- warn 日志：捕获 zap 输出，断言 warn 出现
- tooltip：typecheck 通过即可
- 收敛：所有现有测试 + 新增 `TestRankingConfigStructField` 验证 struct 字段
- CLI：新增 `TestRankingShowMain` 验证输出格式（不实际调 main）
- web-gin：`go build ./...` 成功

## 12. 迁移风险

| 风险 | 等级 | 缓解 |
| --- | --- | --- |
| 11 var → 1 struct 改动面大 | 中 | grep `rankXxx` 全量替换；CI 跑全套测试 |
| web-gin 不依赖 logic-grpc | 中 | 检查 go.mod；如不依赖走简化方案 |
| `applyRankingField` 加 key 参数破坏 helper | 低 | 调用点都在同文件 |
| tooltip 文案多语言 | 低 | 中文 + 英文缩写已 OK；不需 i18n |

## 13. 实现边界

| 模块 | 是否本期修改 |
| --- | --- |
| `logic-grpc-service/service/skill_memory_ranking.go` | 改：var 收敛 + applyRankingField 加 warn |
| `logic-grpc-service/service/skill_memory_ranking_test.go` | 改：引用同步 + 新增 warn 测试 |
| `logic-grpc-service/cmd/ranking-show/main.go` | 新增 |
| `web-gin-service/main.go` | 改：启动加载 config.Ranking |
| `hr-frontend/src/views/hr/admin/SemanticRetrievalDebugView.vue` | 改：6 个 tooltip |
| 其他 | 不变 |

## 14. 备选方案

### 14.1 cmd/ranking-show 通过独立 helper

- 不读 package-level var；新加 `ResolveRankingConfig() RankingConfig` 返回有效值。
- 优点：cmd 可独立测试；不依赖 main 启动顺序。
- 缺点：增加 helper 数量。

### 14.2 warn 日志用 slog

- 项目现有用 zap；切换到 slog 需要改 logger 包。
- 不推荐。

### 14.3 web-gin 走 proto

- 通过共享 `recruitment/pb` 共享 config 段。
- 缺点：proto 用于 RPC；配置共享语义不清晰。

## 15. 待确认问题

- warn 日志格式：zap field vs key=value。
- cmd/ranking-show 输出顺序：按字母序 vs 按 SDD §3.3 顺序。
- web-gin 同步方式：import logic-grpc vs 简化 helper。

## 16. 开放问题

- 是否在 cmd/ranking-show 加 `--json` flag。
- 是否在 debug 视图 breakdown 加 `final_rank` 的条形可视化。
