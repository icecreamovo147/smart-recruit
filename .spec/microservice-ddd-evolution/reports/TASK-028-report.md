# TASK Report - TASK-028

## 1. TASK ID

TASK-028 - Worker infrastructure/tests 与 owner contract 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-worker-service/internal/docs/worker_workload_inventory.md`
- `smart-recruit-worker-service/internal/runtime/workload_profile.go`
- `smart-recruit-worker-service/internal/runtime/workload_profile_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-028-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-028-evidence.json`

## 3. Change Summary by File

- `internal/runtime/workload_profile.go`: OwnerContract 新增 `Retry` 字段，并对所有 workload profile 增加 retry contract 与校验。
- `internal/runtime/workload_profile_test.go`: 新增 readiness/toggle/idempotency/retry/DLQ contract 覆盖测试。
- `internal/docs/worker_workload_inventory.md`: 增加 retry contract 列，记录 Worker 无 shared business service orchestration import，剩余 `smart-recruit-domain-go/mq` 是 infrastructure bridge debt。
- `.spec/.../TASK-028-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-028-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-028` 通过。

变更均在 TASK-028 allowed files 内：`smart-recruit-worker-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-015：Worker workload owner contract 现在显式覆盖 idempotency、retry、DLQ。
- FR-016：确认 Worker 未依赖 shared business service 编排；剩余 shared `mq` 作为 infra bridge debt 记录。
- NFR-007：默认 workload 行为兼容，不新增 workload。

## 6. SDD Comparison Result

符合 SDD：

- 8. Compatibility Strategy：worker runtime check 和默认 workload toggles 保持通过。
- 11. Testing Strategy：覆盖 workload readiness/toggle/idempotency/retry/DLQ contract。
- Implementation boundaries：不修改 shared module、schema、proto、deployment 或 package manifest。

## 7. Acceptance Comparison Result

- Worker 不再依赖共享业务 service 编排，或剩余依赖记录为 shared cleanup 债务：已完成，扫描确认无 `smart-recruit-domain-go/service|repository|model` imports；`smart-recruit-domain-go/mq` infra bridge 已记录为 debt。
- Workload readiness、idempotency、retry/DLQ 测试覆盖：已完成，新增 profile contract test。
- 默认 workload 行为兼容：已完成，profile/default tests 与 runtime check 通过。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `bash -lc '! rg -n "smart-recruit-domain-go/(service|repository|model|ai|oss|resumeparser)|service\\.NewServices|NotificationRuntime|AIAgentRuntime" smart-recruit-worker-service'` | 0 | passed | 无 shared business service orchestration import。 |
| `rg -n "smart-recruit-domain-go/" smart-recruit-worker-service` | 0 | passed | 仅剩 `smart-recruit-domain-go/mq` infrastructure bridge。 |
| `go test ./internal/runtime` in `smart-recruit-worker-service` | 0 | passed | Worker targeted runtime/profile tests 通过。 |
| `go test ./...` in `smart-recruit-worker-service` | 0 | passed | Worker 全 package 测试通过。 |
| `go run ./cmd/worker-service --check` in `smart-recruit-worker-service` | 0 | passed | `worker-service runtime check passed`。 |
| `git diff --name-only` | 0 | passed | 当前 TASK 变更文件已记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-028` | 0 | passed | Scope check passed，变更文件均在 TASK-028 allowed files 内。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Worker module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-028-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 00e48ea65b159a5550ed32fb57c95e2640da982a` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-worker-service/internal/runtime/**
    - smart-recruit-worker-service/internal/docs/worker_workload_inventory.md
  reviewed_documents:
    - .knowledge/runbooks/service-binary-convention.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/event-replay-dead-letter.md
    - .knowledge/pitfalls/migration-model-drift.md
  coverage_gap: false
  reason: TASK-028 completes Worker owner contract convergence, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- Worker service has no shared business service orchestration imports.
- Remaining shared dependency is `smart-recruit-domain-go/mq`, recorded as infrastructure bridge debt for shared cleanup.
- Readiness/toggle/idempotency/retry/DLQ owner contracts are test-covered.
- No default workload, queue, schema, proto, or package manifest changes.

## 11. Risks

- Concrete consumer adapters still need to preserve these owner contracts when introduced or cut over.
- Whether `mq` belongs in future commons is left to TASK-029 shared cleanup.
- Active knowledge source_refs have existing path debt, so knowledge impact detector cannot complete.

## 12. Follow-up Items

- TASK-029 可开始收缩 `smart-recruit-domain-go` 至 commons-ready shared kernel。

## 13. Whether the Next TASK Can Start

TASK-028 通过；TASK-029 可以开始。
