# Acceptance - TASK-FU-008

## TASK Summary

新增 `logic-grpc-service/cmd/ranking-show/main.go` 打印当前生效的 11 个权重。

## SPEC References

- SPEC §5 FR-FU-008
- SPEC §11 AC-FU-008

## SDD References

- SDD §3.4 ranking-show CLI

## Acceptance Criteria

- [ ] AC-001：`go run ./cmd/ranking-show/` 输出 11 行
- [ ] AC-002：每行格式 `key=value`（4 位小数）
- [ ] AC-003：env 覆盖时输出覆盖值
- [ ] AC-004：未设 env 时输出 hardcode 默认
- [ ] AC-005：`go build ./...` 通过

## Required Checks

- [ ] `go build ./...` (logic-grpc-service) 通过
- [ ] `go run ./cmd/ranking-show/` 输出 11 行 `key=value`
- [ ] `git diff --name-only` 仅包含 `logic-grpc-service/cmd/ranking-show/main.go`

## Manual Verification, if needed

- 执行 `cd logic-grpc-service && go run ./cmd/ranking-show/`，输出 11 行 key=value。
- 设置 `RANKING_WEIGHT_VECTOR=0.7` 再跑一次，输出第一行应为 `weight_vector=0.7000`。

## Out-of-Scope

- 不得修改 `LoadRankingConfig` / `ResetRankingConfigForTest`。
- 不得引入新第三方依赖。
- 不得修改 logic-grpc 主体业务代码。
