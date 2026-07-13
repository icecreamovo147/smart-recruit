# TASK Report - TASK-029

## 1. TASK ID

TASK-029 - 收缩 smart-recruit-domain-go 至 commons-ready shared kernel。

## 2. Modified File List

完整变更清单共 605 个路径，见 `.spec/microservice-ddd-evolution/reports/TASK-029-evidence.json` 的 `changed_files`。

主要变更分组：

- `smart-recruit-domain-go/model/**`、`repository/**`、`service/**`：删除 active 业务 model/repository/service 实现。
- `smart-recruit-domain-go/ai/**`：删除 HR/Candidate 业务 tool executor，仅保留 provider/fallback 与通用 tool metadata。
- `smart-recruit-offer-service/internal/legacydomain/**`、`smart-recruit-interview-service/internal/legacydomain/**`、`smart-recruit-recruitment-service/internal/legacydomain/**`、`smart-recruit-ai-agent-service/internal/legacydomain/**`：迁入仍被运行时使用的 owner-local compatibility 代码。
- `smart-recruit-analytics-service/internal/infrastructure/client/authz_adapter.go`：改为 analytics-local SQL authz adapter。
- `smart-recruit-notification-service/internal/infrastructure/mq/outbox_publisher.go`：新增 notification-local outbox publisher。
- `scripts/check-mysql-table-ownership.mjs`、`scripts/redis-prefix-exceptions.json`：同步 owner-local legacy 扫描与 Redis prefix 例外。
- `docs/architecture/**` 与各服务 `internal/docs/**`：记录 TASK-029 后 shared kernel 基线与剩余 service-local legacy debt。

## 3. Change Summary by Area

- Shared module cleanup: `smart-recruit-domain-go` 不再包含 active business `model`、`repository`、`service` 目录；剩余目录为 `ai` provider helpers、`config`、`email`、`internal/platform/events`、`migration`、`mq`、`oss`、`pkg/*`、`resumeparser`。
- Service dependency cleanup: old `smart-recruit-domain-go/model|repository|service` imports 在 Go 代码中清零；需要继续运行的历史实现迁到服务私有 `internal/legacydomain`。
- Runtime adapter cleanup: analytics authz、notification outbox 不再依赖 shared business repository/service。
- Guardrail cleanup: table ownership scanner 覆盖 service-local legacy repository/service roots；backend boundary checker 继续通过。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-029` 通过，变更文件均在 TASK-029 allowed files 内。

补充说明：执行 TASK-029 时先单独提交了 harness 修复 `72675d4`，用于修复 scope checker 对 `smart-recruit-*-service/**` wildcard 的误判；该提交不包含 TASK-029 业务变更。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-016：Worker/Notification/AI Agent 等服务不再依赖 shared business service 编排。
- FR-027：共享模块向 commons-ready shared kernel 收敛。
- FR-028：旧共享业务实现依赖被迁出或删除，module rename 留给 TASK-030。

## 6. SDD Comparison Result

符合 SDD：

- 3.2 Shared Module Target：`smart-recruit-domain-go` 剩余内容已归类为 shared kernel、platform-adjacent、migration helper 或 generic infra helper。
- 3.6 Final Commons Target：未改 module path，保留给 TASK-030。
- 13. Implementation Boundaries：未修改 proto、schema SQL、go.work、package.json 或 lockfile。

## 7. Acceptance Comparison Result

- `smart-recruit-domain-go` 不再包含 active 业务 `model/repository/service` 实现：已完成。
- 剩余内容分类为 shared kernel、platform-adjacent、testutil 或 migration helper：已完成并在 docs 中记录。
- 所有服务对旧业务共享实现依赖清零或转为允许保留的 shared kernel 依赖：Go import 扫描通过，旧业务实现已转为 service-local compatibility。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 记录 tracked unstaged TASK-029 diff；完整 staged/untracked 清单见 evidence。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-029` | 0 | passed | Scope check passed；Changed files: 605。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness validation passed；affected Go modules 的 `go test ./...` 全部通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `bash -lc '! rg -n "smart-recruit-domain-go/(model|repository|service)" smart-recruit-*-service smart-recruit-gateway smart-recruit-domain-go -g "*.go"'` | 0 | passed | Go 代码中旧 shared business imports 清零。 |
| `find smart-recruit-domain-go -maxdepth 2 -type d \| sort` | 0 | passed | 确认 `model`、`repository`、`service` 目录不存在。 |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-029-evidence.json` | 0 | passed | `evidence_result: PASS` |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-domain-go/**
    - smart-recruit-*-service/internal/legacydomain/**
    - scripts/check-mysql-table-ownership.mjs
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/architecture/persistence-and-migrations.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/service-binary-convention.md
  coverage_gap: false
  reason: Active knowledge validation is still blocked by pre-existing legacy logic-grpc-service/web-gin-service source_ref debt; TASK-029 updated direct service docs and records the remaining knowledge refresh as candidate work.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- No Go imports remain for `smart-recruit-domain-go/model`, `repository`, or `service`.
- `smart-recruit-domain-go` has no active business `model/repository/service` directories.
- Affected Go modules compile and test through `agent-check.sh`.
- No proto, schema SQL, module path, workspace, package manifest, or lockfile changes were included.

## 11. Risks

- `internal/legacydomain` keeps compatibility code local to several service owners; later cleanup should reduce copied code once DDD application layers fully replace it.
- `.knowledge` still has broad historical source_ref debt from earlier monolith paths; this TASK records candidate knowledge refresh instead of rewriting the whole knowledge base.
- TASK-030 will be a large import/module/path rename and should rerun full workspace checks.

## 12. Follow-up Items

- TASK-030 can perform the final `smart-recruit-domain-go` to `smart-recruit-commons` rename.

## 13. Whether the Next TASK Can Start

TASK-029 通过；TASK-030 可以开始。
