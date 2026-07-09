# Acceptance - TASK-FU-004

## TASK Summary

把 `skill_memory_ranking.go` 顶部 11 个 `const` 权重 / 阈值改为 `var`；启动时从 `config.Ranking` 段读取；提供 `RANKING_*` env 覆盖；hardcode 行为保持为默认。

## SPEC References

- SPEC §5 FR-FU-004：配置化权重
- SPEC §7 兼容性
- SPEC §11 AC-FU-004

## SDD References

- SDD §3.4 配置化权重
- SDD §7 配置设计
- SDD §11 测试策略

## Acceptance Criteria

- [ ] AC-001：`config.Ranking` 段定义完整；11 个字段都有 yaml + env tag
- [ ] AC-002：`LoadRankingConfig` 在 nil / 空 / 部分填充 / 越界 4 种输入下行为正确
- [ ] AC-003：`ResetRankingConfigForTest` 恢复 11 个 var 到 hardcode 默认
- [ ] AC-004：`go test ./...` 全部通过
- [ ] AC-005：现有 5 个 TASK-001..004 既有测试（如 `TestComputeRelevanceScore` / `TestComputePoolConfidence`）通过
- [ ] AC-006：`config.example.yaml` 增加 `ranking` 段注释

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过
- [ ] `gofmt -l logic-grpc-service/service/skill_memory_ranking.go` 无输出
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/config/**` / `logic-grpc-service/service/skill_memory_ranking.go` / `skill_memory_ranking_test.go` / `logic-grpc-service/cmd/**/main.go`
- [ ] `bash .spec/skill-memory-ranking-followups/scripts/check-task-scope.sh TASK-FU-004` 提示无越界

## Manual Verification, if needed

- 阅读 `LoadRankingConfig(cfg config.Config)` 函数，确认：
  - nil cfg 走 hardcode
  - 空值用默认
  - 部分填充 + 其他默认
  - 越界 clamp 到合法范围
- 设置 `RANKING_WEIGHT_VECTOR=0.7 RANKING_WEIGHT_LEXICAL=0.2 RANKING_WEIGHT_METADATA=0.1` 启动 logic-grpc-service，确认权重生效。

## Out-of-Scope

- 不得修改 proto / pb / 前端 / web-gin-service。
- 不得修改 TASK-001..008 已落地的算法公式。
- 不得引入新第三方依赖。
- 不得修改 `config.Config` 既有字段（只追加 `Ranking` 段）。
- 不得引入权重热加载（启动时一次性 init）。
