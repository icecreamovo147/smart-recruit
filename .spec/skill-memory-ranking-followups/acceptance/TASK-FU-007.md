# Acceptance - TASK-FU-007

## TASK Summary

11 个 `var rankXxx` 替换为 `var rankingConfig` 单一 struct。

## SPEC References

- SPEC §5 FR-FU-007
- SPEC §11 AC-FU-007

## SDD References

- SDD §3.3 RankingConfig 收敛

## Acceptance Criteria

- [ ] AC-001：11 个 `var rankXxx` 替换为 `var rankingConfig` struct
- [ ] AC-002：所有 `rankXxx` 引用改为 `rankingConfig.Xxx`
- [ ] AC-003：`LoadRankingConfig` / `ResetRankingConfigForTest` 签名不变
- [ ] AC-004：所有现有测试通过
- [ ] AC-005：`go test ./...` 通过

## Required Checks

- [ ] `cd logic-grpc-service && go test ./...` 通过
- [ ] `grep -E "^\s*rankWeightVector\s*=" logic-grpc-service/service/skill_memory_ranking.go` 应该为 0（var 定义已删除）
- [ ] `grep -E "rankingConfig\." logic-grpc-service/service/skill_memory_ranking.go` >= 11

## Manual Verification, if needed

- 阅读 `skill_memory_ranking.go`，确认 11 个 var 已替换为单一 struct。
- 跑全套测试，确保无回归。

## Out-of-Scope

- 不得修改 `LoadRankingConfig` / `ResetRankingConfigForTest` 公开签名。
- 不得弱化或删除任何已有测试。
- 不得修改算法公式。
