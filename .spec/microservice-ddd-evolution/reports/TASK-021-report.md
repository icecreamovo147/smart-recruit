# TASK Report - TASK-021

## 1. TASK ID

TASK-021 - Analytics infrastructure/interfaces/runtime/tests 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-analytics-service/cmd/analytics-service/main.go`
- `smart-recruit-analytics-service/internal/application/port/reporting.go`
- `smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md`
- `smart-recruit-analytics-service/internal/infrastructure/client/authz_adapter.go`
- `smart-recruit-analytics-service/internal/infrastructure/persistence/gorm_reporting_repository.go`
- `smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store.go`
- `smart-recruit-analytics-service/internal/infrastructure/projection/gorm_store_test.go`
- `smart-recruit-analytics-service/internal/interfaces/grpc/reporting_api.go`
- `smart-recruit-analytics-service/internal/interfaces/grpc/reporting_api_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-021-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-021-evidence.json`

## 3. Change Summary by File

- `cmd/analytics-service/main.go`: runtime reporting 装配从 shared `service.NewAnalyticsService`/`repository.NewAnalyticsRepo` 切换为本地 `application/service.ReportingService`、本地 GORM reporting repository 和本地 gRPC adapter。
- `internal/application/port/reporting.go`: 增加 `ErrPermissionDenied` typed error，供 interfaces 层保持 forbidden/internal 响应分类。
- `internal/infrastructure/client/authz_adapter.go`: 新增 Identity permission/data-scope 兼容 adapter，暂时通过 shared `AuthzRepo` 读取权限与 scope。
- `internal/infrastructure/persistence/gorm_reporting_repository.go`: 新增本地 GORM reporting repository，覆盖 dashboard、unread、trend、stage distribution、funnel、time-in-stage、interview metrics 和 offer metrics 查询。
- `internal/infrastructure/projection/gorm_store.go`: 新增 Analytics-owned projection event/checkpoint GORM store，仅写 projection 表。
- `internal/infrastructure/projection/gorm_store_test.go`: 覆盖 projection event 幂等保存和 checkpoint upsert。
- `internal/interfaces/grpc/reporting_api.go`: 新增 AdminService reporting subset proto adapter，保持 response code/msg、字段映射、date parse 和旧错误语义。
- `internal/interfaces/grpc/reporting_api_test.go`: 覆盖 dashboard、funnel、time-in-stage、interview-offer metrics proto 响应兼容和 forbidden 映射。
- `internal/docs/projection_strategy_inventory.md`: 更新 TASK-021 后当前事实，记录 shared analytics implementation 已移除，剩余 AuthzRepo 兼容债务。
- `.spec/.../TASK-021-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-021-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-021` 通过。

变更均在 TASK-021 allowed files 内：`smart-recruit-analytics-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-004：新增本地 infrastructure persistence/client/projection adapter。
- FR-005：新增本地 interfaces/grpc adapter，负责 proto request/response mapping。
- FR-006：runtime 装配改用本地 implementation，保留健康检查、metrics、trace、Nacos 和 gRPC internal auth。
- FR-013：Analytics 收敛为 reporting query、projection/read-model 能力，不写回事务业务状态。
- FR-016：清除了 active runtime 对 shared `AnalyticsService`/`AnalyticsRepo` 的依赖；剩余 shared `AuthzRepo` 记录为 Identity scope 兼容债务。

## 6. SDD Comparison Result

符合 SDD：

- 8. Compatibility Strategy：未改 protobuf、Gateway、schema 或 public API；AdminService reporting subset 注册保持不变。
- 11. Testing Strategy：新增 interfaces/projection tests，并运行 Analytics 服务 `go test ./...`。
- Implementation boundaries：只修改 Analytics 服务本地代码和 TASK 报告，未改 shared modules。

## 7. Acceptance Comparison Result

- Analytics runtime 使用本地 implementation：已完成，`buildReportingService` 装配本地 application/infrastructure/interfaces。
- Reporting API 兼容：已完成，gRPC adapter 保持旧 response code/msg、字段映射、stage label、date parse 和错误消息策略。
- 对共享 analytics implementation 的依赖清除或记录债务：已完成，代码中不再引用 shared `AnalyticsService`/`AnalyticsRepo`；文档记录剩余 shared `AuthzRepo` 为 Identity 兼容债务。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-analytics-service` | 0 | passed | Analytics 全 package 测试通过，包含 local application、interfaces、projection 和 runtime tests。 |
| `go run ./cmd/analytics-service --check` | 0 | passed | `analytics-service runtime check passed`，AdminService reporting subset 可注册。 |
| `sh -c '! rg -n "NewAnalyticsService|NewAnalyticsRepo|smart-recruit-commons/service" smart-recruit-analytics-service'` | 0 | passed | 无匹配，active analytics service 不再引用 shared analytics service/repo 或 shared service package。 |
| `git diff --name-only` | 0 | passed | tracked diff 列出 `cmd/analytics-service/main.go`、`internal/application/port/reporting.go`、`internal/docs/projection_strategy_inventory.md`；untracked 新文件由 scope check 覆盖。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-021` | 0 | passed | report/evidence 创建前 `Changed files: 9`，创建后复跑 `Changed files: 11`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Analytics module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-021-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 23de4d37380dcb1a499e8d45a0b48447fc5720bf` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-analytics-service/cmd/analytics-service/main.go
    - smart-recruit-analytics-service/internal/infrastructure/**
    - smart-recruit-analytics-service/internal/interfaces/**
    - smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/local-development.md
    - .knowledge/runbooks/service-binary-convention.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-analytics-runtime.md
  coverage_gap: false
  reason: TASK-021 switches Analytics runtime to local implementation, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- Runtime no longer imports shared `smart-recruit-commons/service`.
- Code search confirms no `NewAnalyticsService` or `NewAnalyticsRepo` usage in Analytics service.
- Public protobuf contract and AdminService reporting registration are unchanged.
- Local persistence adapter only reads transitional source-domain tables and writes Analytics-owned projection tables.
- Projection store does not write transactional Recruitment, Interview, Offer, Notification, Identity, or AI Agent state.
- Remaining shared `AuthzRepo` is documented as Identity permission/data-scope compatibility debt.

## 11. Risks

- Local reporting SQL mirrors legacy shared query semantics; integration behavior depends on existing MySQL schema and transitional read access.
- Identity permission/data-scope still uses shared `AuthzRepo` until an owner service query/snapshot contract exists.
- Active knowledge source_refs have existing path debt, so knowledge impact detection cannot complete.

## 12. Follow-up Items

- TASK-022 可开始 AI Agent DDD 骨架与能力盘点。
- 后续 `.knowledge/**` scope 可用时，应更新 Analytics runtime/projection 知识 source_refs。

## 13. Whether the Next TASK Can Start

TASK-021 通过；TASK-022 可以开始。
