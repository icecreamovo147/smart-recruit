# Acceptance - TASK-FU-005

## TASK Summary

`applyRankingField` 在 env 值越界时输出结构化 warn 日志。

## SPEC References

- SPEC §5 FR-FU-005
- SPEC §11 AC-FU-005

## SDD References

- SDD §3.1 warn 日志

## Acceptance Criteria

- [ ] AC-001：`applyRankingField` 增加 `key string` 参数
- [ ] AC-002：value 越界时输出 `logger.L().Warn` 包含 key / value / clamped 字段
- [ ] AC-003：value 在 [min, max] 区间内不 warn
- [ ] AC-004：value == 0 不 warn
- [ ] AC-005：单测覆盖 3 种 case（normal / below_min / above_max / zero）
- [ ] AC-006：`go test ./...` 通过

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过
- [ ] `gofmt -l logic-grpc-service/service/skill_memory_ranking.go` 无输出
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/service/skill_memory_ranking.go` / `skill_memory_ranking_test.go`

## Manual Verification, if needed

- 设置 `RANKING_BUSINESS_BOOST_MAX=2.0` 启动 logic-grpc-service，确认 warn 日志输出 `[ranking] env value out of range, clamped` + key=RANKING_BUSINESS_BOOST_MAX value=2.0 clamped=1.5。

## Out-of-Scope

- 不得修改 proto / pb / config / 前端 / web-gin。
- 不得修改 TASK-001..008 / TASK-FU-001..004 已落地逻辑。
