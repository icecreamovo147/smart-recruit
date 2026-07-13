# TASK Report - TASK-019

## 1. TASK ID

TASK-019 - Analytics DDD 骨架与投影策略盘点。

## 2. Modified File List

实际变更文件：

- `smart-recruit-analytics-service/internal/application/doc.go`
- `smart-recruit-analytics-service/internal/application/dto/doc.go`
- `smart-recruit-analytics-service/internal/application/port/doc.go`
- `smart-recruit-analytics-service/internal/application/query/doc.go`
- `smart-recruit-analytics-service/internal/application/service/doc.go`
- `smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md`
- `smart-recruit-analytics-service/internal/domain/doc.go`
- `smart-recruit-analytics-service/internal/domain/event/doc.go`
- `smart-recruit-analytics-service/internal/domain/model/doc.go`
- `smart-recruit-analytics-service/internal/domain/policy/doc.go`
- `smart-recruit-analytics-service/internal/domain/repository/doc.go`
- `smart-recruit-analytics-service/internal/domain/service/doc.go`
- `smart-recruit-analytics-service/internal/infrastructure/doc.go`
- `smart-recruit-analytics-service/internal/infrastructure/client/doc.go`
- `smart-recruit-analytics-service/internal/infrastructure/persistence/doc.go`
- `smart-recruit-analytics-service/internal/infrastructure/projection/doc.go`
- `smart-recruit-analytics-service/internal/interfaces/doc.go`
- `smart-recruit-analytics-service/internal/interfaces/grpc/doc.go`
- `smart-recruit-analytics-service/internal/interfaces/mapper/doc.go`
- `.spec/microservice-ddd-evolution/reports/TASK-019-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-019-evidence.json`

## 3. Change Summary by File

- `internal/domain/**`: 新增 Analytics domain skeleton，划分 model、repository、service、event、policy。
- `internal/application/**`: 新增 Analytics application skeleton，划分 query、dto、port、service。
- `internal/infrastructure/**`: 新增 Analytics infrastructure skeleton，划分 persistence、projection、client。
- `internal/interfaces/**`: 新增 Analytics inbound adapter skeleton，划分 grpc 与 mapper。
- `internal/docs/projection_strategy_inventory.md`: 盘点 active runtime、reporting API、projection ownership、transitional read debt、future projection event inputs、existing tests 和迁移注意事项。
- `.spec/.../TASK-019-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-019-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-019` 通过。

变更均在 TASK-019 allowed files 内：`smart-recruit-analytics-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-001 至 FR-007：建立 Analytics service 的目标 DDD 分层目录。
- FR-013：记录 reporting API、projection table、只读跨表依赖和事件投影目标。
- Compatibility：未修改 runtime wiring、protobuf、schema、gateway 或 active shared reporting implementation，因此不改变报表 API 行为。

## 6. SDD Comparison Result

符合 SDD：

- 3.3 Required Migration Order：Analytics 在 Recruitment 收敛后开始骨架/盘点。
- 8. Compatibility Strategy：保持 AdminService reporting subset、projection descriptor、Nacos、health、metrics、trace、MySQL/Redis runtime 行为不变。
- Implementation boundaries：未创建新 projection schema 或事件 schema。

## 7. Acceptance Comparison Result

- Analytics DDD 骨架存在：已完成，新增 domain/application/infrastructure/interfaces 分层及子 package。
- reporting API、projection、只读跨表债务被记录：已完成，`internal/docs/projection_strategy_inventory.md` 覆盖 dashboard、funnel、time-in-stage、interview-offer metrics、projection ownership 和 transitional read debt。
- 不改变报表 API 行为：已完成，本 TASK 未修改 `cmd/analytics-service/main.go`、runtime registration、protobuf、schema、gateway 或 shared reporting service。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-analytics-service` | 0 | passed | Analytics 全 package 测试通过，新增 skeleton packages 可编译。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | 当前 TASK 新增文件为 untracked；通过 `git status --short`、scope check 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-019` | 0 | passed | `Scope check passed for TASK-019. Changed files: 21`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Analytics module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 681e0c2339b9d3a57ed413f61ff2e35d4373f8d2` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-analytics-service/internal/docs/projection_strategy_inventory.md
    - smart-recruit-analytics-service/internal/domain/**
    - smart-recruit-analytics-service/internal/application/**
    - smart-recruit-analytics-service/internal/infrastructure/**
    - smart-recruit-analytics-service/internal/interfaces/**
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/architecture/persistence-and-migrations.md
    - .knowledge/inbox/analytics-projection-knowledge.md
    - .knowledge/runbooks/service-binary-convention.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-analytics-projection.md
  coverage_gap: false
  reason: TASK-019 establishes the Analytics DDD/projection target boundary while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增文件均在 `smart-recruit-analytics-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- Active Analytics runtime/reporting behavior 未改变。
- Projection inventory 明确 Analytics-owned projection tables 和 transitional read-only cross-table debt。
- `go test ./...`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 11. Risks

- Analytics active reporting 仍使用 shared `AnalyticsService` 和 `AnalyticsRepo`；本 TASK 只建立骨架与盘点，业务迁移由 TASK-020/TASK-021 执行。
- Projection-backed reads 需要后续 adapter/replay/backfill 策略，当前未创建 schema 或事件 schema。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-020 可开始 Analytics reporting/projection application 迁移。
- 后续 `.knowledge/**` scope 可用时，应更新 Analytics projection/read-model 知识文档的 source_refs 与 service root。

## 13. Whether the Next TASK Can Start

TASK-019 通过；TASK-020 可以开始。
