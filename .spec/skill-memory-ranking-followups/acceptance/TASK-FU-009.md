# Acceptance - TASK-FU-009

## TASK Summary

web-gin-service 启动时也加载 `config.Ranking` 段。

## SPEC References

- SPEC §5 FR-FU-009
- SPEC §11 AC-FU-009

## SDD References

- SDD §3.5 web-gin-service 同步

## Acceptance Criteria

- [ ] AC-001：web-gin 启动时调 `LoadRankingConfig`（如可依赖）或读取 `config.Ranking` 段（fallback）
- [ ] AC-002：`go build ./...` 通过（web-gin）
- [ ] AC-003：启动日志显示 `ranking` 段已加载

## Required Checks

- [ ] `cd web-gin-service && go build ./...` 通过
- [ ] `grep -c "Ranking" web-gin-service/main.go` >= 1
- [ ] `git diff --name-only` 仅包含 `web-gin-service/main.go` / `web-gin-service/cmd/**/main.go`

## Manual Verification, if needed

- 启动 web-gin-service，确认日志包含 ranking 段加载信息。
- 设置 `RANKING_WEIGHT_VECTOR=0.7` 启动，确认日志输出 weight_vector=0.7。

## Out-of-Scope

- 不得修改 web-gin 的 handler / router / rpc。
- 不得引入 web-gin 与 logic-grpc 的新依赖关系（如之前不存在）。
- 不得修改 proto / pb。
