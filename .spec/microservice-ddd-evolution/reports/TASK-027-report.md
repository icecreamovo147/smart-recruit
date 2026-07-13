# TASK Report - TASK-027

## 1. TASK ID

TASK-027 - Worker workload application/runtime 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-worker-service/internal/runtime/runtime.go`
- `smart-recruit-worker-service/internal/runtime/runtime_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-027-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-027-evidence.json`

## 3. Change Summary by File

- `internal/runtime/runtime.go`: Runtime 启动时校验 workload profile、toggle 和 owner contract；新增 `EnabledProfiles`、`Stop` 和 optional `Stopper` lifecycle；starter 使用 runtime-owned cancellable context。
- `internal/runtime/runtime_test.go`: 覆盖 profile-backed 启动、enabled profile 可见性，以及 graceful shutdown 取消 workload context 并按反向顺序 stop。
- `.spec/.../TASK-027-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-027-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-027` 通过。

变更均在 TASK-027 allowed files 内：`smart-recruit-worker-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-015：Worker runtime 通过 profile/toggle/owner contract 驱动 workload 编排。
- NFR-007：未新增 workload，默认启停语义保持不变。
- SSR-006：后台写入由 owner contract 校验约束，不新增绕过 owner service/domain 的写路径。

## 6. SDD Comparison Result

符合 SDD：

- 6. Algorithm or Workflow Changes：runtime 启动前校验 profile/owner contract，启动后用 runtime-owned context 管理 workload lifecycle。
- 8. Compatibility Strategy：`WORKER_WORKLOADS`/`WORKER_DISABLED_WORKLOADS` parse 行为不变，health/runtime check 继续通过。
- Implementation boundaries：不修改 shared module、schema、proto、queue binding 或真实 consumer 实现。

## 7. Acceptance Comparison Result

- Workload 通过 profile/toggle 分类运行：已完成，`Runtime` 持有 profile set 并在 Start 前校验 enabled profile/toggle/owner contract。
- 后台写入遵守 owner contract：已完成，enabled workload 缺失 owner context/writes contract 会拒绝启动；本 TASK 未新增写路径。
- 覆盖 workload parse/toggle/graceful shutdown 测试：已完成，新增/扩展 runtime tests。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./internal/runtime` in `smart-recruit-worker-service` | 0 | passed | Worker runtime targeted tests 通过，覆盖 profile/toggle/graceful stop。 |
| `go test ./...` in `smart-recruit-worker-service` | 0 | passed | Worker 全 package 测试通过。 |
| `go run ./cmd/worker-service --check` in `smart-recruit-worker-service` | 0 | passed | `worker-service runtime check passed`。 |
| `git diff --name-only` | 0 | passed | 当前 TASK 变更文件已记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-027` | 0 | passed | Scope check passed，变更文件均在 TASK-027 allowed files 内。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Worker module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-027-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree c703a9602168549971ed339e9c8119b0416a58b8` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-worker-service/internal/runtime/**
  reviewed_documents:
    - .knowledge/runbooks/service-binary-convention.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/event-replay-dead-letter.md
  coverage_gap: false
  reason: TASK-027 changes Worker runtime lifecycle, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 不新增 workload、不修改 queue/schema/proto/shared module。
- Runtime 启动仍按 `DefaultWorkloads` descriptor order，parse/toggle 默认行为不变。
- Graceful stop 只取消 runtime-owned context 并调用 optional `Stopper`，不会改变现有 starter contract。
- Owner contract 校验为启动前 guard；没有新增实际写表路径。

## 11. Risks

- Stop lifecycle 依赖 future concrete starters 实现 optional `Stopper` 才能执行自定义清理；当前 controlled starters 仍依赖 context cancellation。
- Owner contract guard 不能替代后续真实 consumer adapter 的表访问审查。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-028 可开始 Worker infrastructure/tests 与 owner contract 收敛。

## 13. Whether the Next TASK Can Start

TASK-027 通过；TASK-028 可以开始。
