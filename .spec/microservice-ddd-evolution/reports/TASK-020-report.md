# TASK Report - TASK-020

## 1. TASK ID

TASK-020 - Analytics reporting/projection application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-analytics-service/internal/application/dto/reporting.go`
- `smart-recruit-analytics-service/internal/application/port/reporting.go`
- `smart-recruit-analytics-service/internal/application/query/reporting.go`
- `smart-recruit-analytics-service/internal/application/service/reporting_service.go`
- `smart-recruit-analytics-service/internal/application/service/reporting_service_test.go`
- `smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md`
- `smart-recruit-analytics-service/internal/domain/model/reporting.go`
- `smart-recruit-analytics-service/internal/domain/policy/reporting.go`
- `smart-recruit-analytics-service/internal/domain/policy/reporting_test.go`
- `smart-recruit-analytics-service/internal/domain/repository/reporting.go`
- `.spec/microservice-ddd-evolution/reports/TASK-020-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-020-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/reporting.go`: 新增 Analytics 本地 reporting/projection domain model，包括 scope、report filter、dashboard KPI、funnel stage、time-in-stage、interview/offer metrics、projection event/checkpoint 和 transitional read debt 描述。
- `internal/domain/policy/reporting.go`: 新增本地报表口径策略，保持旧漏斗顺序、stage label、转化率、平均小时、面试通过率、Offer 接受率和 actor mismatch 规则，并验证 projection strategy 禁止事务业务状态写入。
- `internal/domain/repository/reporting.go`: 新增 reporting read port 与 projection store port，后续 infrastructure adapter 只需实现本地接口。
- `internal/application/query/reporting.go`: 新增 dashboard、funnel、time-in-stage、interview-offer metrics query 输入。
- `internal/application/dto/reporting.go`: 新增 application 输出 DTO，保留 dashboard secondary read fail-soft warning。
- `internal/application/port/reporting.go`: 新增 permission authorizer 与 scope provider port，避免 application 直接依赖共享 authz repository。
- `internal/application/service/reporting_service.go`: 新增 ReportingService 和 ProjectionService，本地化报表编排、权限/范围检查、口径转换、projection event/checkpoint 写入。
- `internal/application/service/reporting_service_test.go`: 覆盖 dashboard、funnel、time-in-stage、interview-offer metrics 和 projection-only writes。
- `internal/domain/policy/reporting_test.go`: 覆盖漏斗顺序/转化率、阶段耗时、面试/Offer rate、actor mismatch 与事务写禁止策略。
- `internal/docs/projection_strategy_inventory.md`: 记录 TASK-020 本地 application boundary、只读过渡债务和 projection write 禁止边界。
- `.spec/.../TASK-020-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-020-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-020` 通过。

变更均在 TASK-020 allowed files 内：`smart-recruit-analytics-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-003：Analytics application 层现在承载报表 query 编排、权限/范围 port 调用、projection 写入时机。
- FR-013：Analytics 本地化为 reporting query、projection/read-model 能力，并通过 policy 明确禁止事务业务状态写回。
- FR-020：跨上下文读保持为受控 transitional read debt，没有新增直接跨 owner 表 join 或写入。
- FR-025：新增 domain policy 与 application service 单元测试覆盖 dashboard、funnel、time-in-stage、interview-offer metrics。

## 6. SDD Comparison Result

符合 SDD：

- 3.1 Target Service Shape：补齐 Analytics 本地 domain/application 内的 model、repository port、policy、query、dto、port、service。
- 4. Data Structure Changes：未修改 schema、migration 或 `db.sql`；projection 写入只描述本地 port，不创建新表。
- 8. Compatibility Strategy：runtime wiring 未切换，protobuf/API 行为不变；本地口径与旧 shared AnalyticsService/AnalyticsRepo 保持兼容。

## 7. Acceptance Comparison Result

- Reporting/projection application 本地化：已完成，新增本地 query/dto/port/service、domain model/policy/repository port。
- 不写回事务业务状态：已完成，`ProjectionService` 仅依赖 `ProjectionStore` 写 projection event/checkpoint，policy 拒绝 `TransactionalWrites=true`。
- 报表口径兼容，过渡只读债务被记录：已完成，漏斗顺序、stage label、转化率、time-in-stage 小时转换、面试/Offer rate 与旧实现一致，文档记录 transitional read debt。
- 覆盖 dashboard/funnel/time-in-stage/interview-offer metrics 测试：已完成。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-analytics-service` | 0 | passed | Analytics 全 package 测试通过，新增 reporting/projection application 和 policy tests 通过。 |
| `git diff --name-only` | 0 | passed | 跟踪文件 diff 显示 `smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md`；新增文件由 scope check 的 untracked 扫描覆盖。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-020` | 0 | passed | report/evidence 创建前 `Changed files: 10`，创建后复跑 `Changed files: 12`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Analytics module 变更并运行 `go test ./...` 通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-ddd-evolution/reports/TASK-020-evidence.json` | 0 | passed | `evidence_result: PASS` |
| `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 5d4d34f952b01f453f8df22fe515c06293eb0439` | 1 | non-blocking | 既有 active knowledge 仍引用已迁移/缺失的旧 `logic-grpc-service` / `web-gin-service` source_refs；本 TASK scope 不允许改 `.knowledge/**`，记录为 candidate_required。 |

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-analytics-service/internal/application/**
    - smart-recruit-analytics-service/internal/domain/**
    - smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/local-development.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-analytics-reporting-application.md
  coverage_gap: false
  reason: TASK-020 localizes Analytics reporting/projection application boundaries, but active knowledge validation is blocked by pre-existing legacy source_ref debt outside this TASK scope.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增文件均在 TASK-020 allowed scope 内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- Runtime 尚未切换到本地 application，符合 TASK-020 边界；TASK-021 负责 infrastructure/interfaces/runtime/tests 收敛。
- ReportingService 保持旧权限 key、scope actor mismatch、dashboard secondary read fail-soft、漏斗顺序/转化率、time-in-stage 和面试/Offer rate 口径。
- ProjectionService 只有 Analytics projection store port，没有事务业务状态写 port。

## 11. Risks

- Active runtime 仍使用 shared `AnalyticsService` 和 `AnalyticsRepo`，切换本地 application/infrastructure adapter 的风险留给 TASK-021。
- Transitional read debt 仍存在，后续需要 projection-backed read model、replay/backfill 和 owner service query adapter。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-021 可开始 Analytics infrastructure/interfaces/runtime/tests 收敛。
- 后续 `.knowledge/**` scope 可用时，应更新 Analytics projection/read-model 知识文档与旧路径 source_refs。

## 13. Whether the Next TASK Can Start

TASK-020 通过；TASK-021 可以开始。
