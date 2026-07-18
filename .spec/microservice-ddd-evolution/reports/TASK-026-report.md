# TASK Report - TASK-026

## 1. TASK ID

TASK-026 - Worker DDD/workload 骨架与边界盘点。

## 2. Modified File List

实际变更文件：

- `smart-recruit-worker-service/README.md`
- `smart-recruit-worker-service/internal/docs/worker_workload_inventory.md`
- `smart-recruit-worker-service/internal/runtime/workload_profile.go`
- `smart-recruit-worker-service/internal/runtime/workload_profile_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-026-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-026-evidence.json`

## 3. Change Summary by File

- `README.md`: 增加 workload profile/owner-contract inventory 文档入口。
- `internal/docs/worker_workload_inventory.md`: 新增 Worker workload toggle、Outbox/Inbox/DLQ/notification/email/resume/embedding/agent-run/analytics owner contract 盘点。
- `internal/runtime/workload_profile.go`: 新增 workload profile、toggle、owner contract model，并为所有默认 workloads 建立 profile。
- `internal/runtime/workload_profile_test.go`: 新增 profile 与默认 runtime descriptor 一致性、owner contract 完整性测试。
- `.spec/.../TASK-026-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-026-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-026` 通过。

变更均在 TASK-026 allowed files 内：`smart-recruit-worker-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-001 至 FR-007：Worker 服务内新增 workload profile/owner contract 骨架。
- FR-015：记录 Outbox、Inbox、DLQ、notification、email、resume parsing、embedding、agent run、analytics projection workload owner boundaries。
- NFR-007：不改变默认 workload 启停行为，不拆分 worker binary。

## 6. SDD Comparison Result

符合 SDD：

- 3.3 Required Migration Order：AI Agent 阶段完成后进入 Worker 阶段。
- 3.4 Per-Service Migration Pattern：本 TASK 建立 Worker runtime/workload boundary，并记录 owner context contract。
- Implementation boundaries：不迁移 shared worker implementation，不新增 workload，不修改 queue/schema/proto。

## 7. Acceptance Comparison Result

- Worker workload profile/toggle 骨架存在：已完成，新增 `WorkloadProfile`、`WorkloadToggle`、`OwnerContract`。
- Outbox/Inbox/DLQ/notification/email/resume/embedding/agent-run/analytics owner contract 被记录：已完成，见 `internal/docs/worker_workload_inventory.md` 和 `DefaultWorkloadProfiles`。
- 默认 workload 行为不变：已完成，profile tests 验证 profile names 与 `ParseWorkloadConfig("", "")` 默认 enabled workloads 一致。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-worker-service` | 0 | passed | Worker 全 package 测试通过。 |
| `go run ./cmd/worker-service --check` in `smart-recruit-worker-service` | 0 | passed | `worker-service runtime check passed`。 |
| `git diff --name-only` | 0 | passed | tracked diff 记录 README；新增 Worker files 由 `git status --short` 和 evidence 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-026` | 0 | passed | Scope check passed，变更文件均在 TASK-026 allowed files 内。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Worker module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-026-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 44f662e52e4ac2014e2de2a3c41daf64983fcbe6` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-worker-service/**
  reviewed_documents:
    - .knowledge/runbooks/service-binary-convention.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/event-replay-dead-letter.md
    - .knowledge/domains/resume-intelligence.md
    - .knowledge/pitfalls/resume-sensitive-data.md
    - .knowledge/pitfalls/migration-model-drift.md
  coverage_gap: false
  reason: TASK-026 records Worker workload ownership, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 未拆分 worker binary，未新增 workload，未改 queue binding 或 startup behavior。
- Default workload profile 与 runtime descriptor 一一对应，测试覆盖默认启用集合。
- Owner contract 覆盖 Outbox/Inbox/DLQ/notification/email/resume/embedding/agent-run/analytics projection。
- 未修改 shared module、schema、proto、deployment、package manifest 或 lockfile。

## 11. Risks

- 本 TASK 仅建立 profile/contract 骨架；实际 consumer starter 迁移留给后续 Worker runtime TASK。
- Worker owner contract 仍依赖后续 adapter 实现遵守，不是运行时强制表访问隔离。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-027 可开始 Worker workload application/runtime 迁移。

## 13. Whether the Next TASK Can Start

TASK-026 通过；TASK-027 可以开始。
