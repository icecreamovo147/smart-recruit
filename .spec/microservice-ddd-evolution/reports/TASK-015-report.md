# TASK Report - TASK-015

## 1. TASK ID

TASK-015 - Recruitment DDD 骨架与核心域盘点。

## 2. Modified File List

实际变更文件：

- `smart-recruit-recruitment-service/internal/application/doc.go`
- `smart-recruit-recruitment-service/internal/application/command/doc.go`
- `smart-recruit-recruitment-service/internal/application/dto/doc.go`
- `smart-recruit-recruitment-service/internal/application/port/doc.go`
- `smart-recruit-recruitment-service/internal/application/query/doc.go`
- `smart-recruit-recruitment-service/internal/application/service/doc.go`
- `smart-recruit-recruitment-service/internal/docs/core_domain_inventory.md`
- `smart-recruit-recruitment-service/internal/domain/doc.go`
- `smart-recruit-recruitment-service/internal/domain/event/doc.go`
- `smart-recruit-recruitment-service/internal/domain/model/doc.go`
- `smart-recruit-recruitment-service/internal/domain/policy/doc.go`
- `smart-recruit-recruitment-service/internal/domain/repository/doc.go`
- `smart-recruit-recruitment-service/internal/domain/service/doc.go`
- `smart-recruit-recruitment-service/internal/domain/valueobject/doc.go`
- `smart-recruit-recruitment-service/internal/infrastructure/doc.go`
- `smart-recruit-recruitment-service/internal/infrastructure/client/doc.go`
- `smart-recruit-recruitment-service/internal/infrastructure/mq/doc.go`
- `smart-recruit-recruitment-service/internal/infrastructure/oss/doc.go`
- `smart-recruit-recruitment-service/internal/infrastructure/persistence/doc.go`
- `smart-recruit-recruitment-service/internal/interfaces/doc.go`
- `smart-recruit-recruitment-service/internal/interfaces/grpc/doc.go`
- `smart-recruit-recruitment-service/internal/interfaces/mapper/doc.go`
- `.spec/microservice-ddd-evolution/reports/TASK-015-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-015-evidence.json`

## 3. Change Summary by File

- `internal/domain/**`: 新增 Recruitment domain package skeleton，划分 model、valueobject、repository、service、event、policy 边界，覆盖 jobs、candidate profiles、resumes、applications、collaboration、taxonomy、usage stats 的后续迁移承载位置。
- `internal/application/**`: 新增 application package skeleton，划分 command、query、dto、port、service，用于后续承载 Recruitment use-case orchestration 与外部依赖端口。
- `internal/infrastructure/**`: 新增 infrastructure package skeleton，划分 persistence、client、oss、mq adapters，用于后续隔离 GORM、Authz、OSS、outbox/MQ 等技术实现。
- `internal/interfaces/**`: 新增 inbound adapter skeleton，划分 grpc 与 mapper，用于后续把 protobuf-facing service 映射到本地 application layer。
- `internal/docs/core_domain_inventory.md`: 新增 Recruitment 核心域盘点，记录当前 runtime 边界、服务表面、jobs/candidate/resume/application/collaboration/taxonomy/usage stats 领域、事件和外部依赖、表归属、现有测试锚点和后续迁移注意事项。
- `.spec/.../TASK-015-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-015-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-015` 通过。

变更均在 TASK-015 allowed files 内：`smart-recruit-recruitment-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-001 至 FR-007：建立 Recruitment service 的目标 DDD 分层目录，后续可按 domain/application/infrastructure/interfaces/runtime 迁移。
- FR-012：盘点 Recruitment job、candidate、resume、application、collaboration、taxonomy、usage stats 核心域、表归属和外部依赖。
- CR-007 / 兼容性要求：未修改 protobuf、schema、runtime wiring、gateway、HTTP/gRPC API 行为或 active shared implementation。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：本 TASK 只建立骨架与核心域盘点，不切换 active runtime。
- 12. Migration Risks：盘点记录 Recruitment 最大上下文中的 OSS、outbox、Identity authz、Interview/Offer/AI/Analytics transitional reads 与 usage stats 归属风险。
- 13. Implementation Boundaries：变更限制在 Recruitment service 与 TASK report 目录内。

## 7. Acceptance Comparison Result

- Recruitment DDD 骨架存在：已完成，新增 domain/application/infrastructure/interfaces 分层及子 package。
- job/candidate/resume/application/collaboration/taxonomy/usage stats 盘点完成：已完成，`internal/docs/core_domain_inventory.md` 覆盖对应领域、规则、表归属、依赖和后续迁移注意事项。
- 不改变招聘 API 行为：已完成，本 TASK 未修改 `cmd/recruitment-service/main.go`、runtime registration、protobuf、schema、gateway 或现有服务实现。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-recruitment-service` | 0 | passed | Recruitment 全 package 测试通过，新增 skeleton packages 可编译。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | 当前 TASK 新增文件为 untracked；通过 `git status --short`、`git ls-files --others --exclude-standard`、scope check 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-015` | 0 | passed | `Scope check passed for TASK-015. Changed files: 24`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Recruitment module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree fc8fb85682d605a8bfcfc7aa1d6b21263f58c99b` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-recruitment-service/internal/docs/core_domain_inventory.md
    - smart-recruit-recruitment-service/internal/domain/**
    - smart-recruit-recruitment-service/internal/application/**
    - smart-recruit-recruitment-service/internal/infrastructure/**
    - smart-recruit-recruitment-service/internal/interfaces/**
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/runbooks/debug-recruitment-lifecycle.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-recruitment-core.md
  coverage_gap: false
  reason: TASK-015 records the new Recruitment DDD target boundary while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 新增文件均在 `smart-recruit-recruitment-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- Active Recruitment runtime wiring 仍保持现状，未改变 API 行为。
- `internal/docs/core_domain_inventory.md` 覆盖 acceptance 要求的全部核心域，并记录跨服务/平台依赖风险。
- `go test ./...`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 11. Risks

- Recruitment 当前 active implementation 仍在 shared `smart-recruit-domain-go`；本 TASK 只建立骨架和盘点，业务迁移由后续 TASK 执行。
- Usage stats 当前依赖 platform-owned `third_party_usage_logs`，后续迁移需要在 Recruitment read model 与 Analytics/platform ownership 之间保持清晰边界。
- Collaboration workspace 仍有 Interview/Offer transitional reads；后续迁移必须保持跨上下文只读兼容，不引入新写入。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-016 可开始 Recruitment job/candidate/resume domain/application 迁移。
- 后续 `.knowledge/**` scope 可用时，应更新 Recruitment 领域和 lifecycle 知识文档的 source_refs 与 DDD 目标边界。

## 13. Whether the Next TASK Can Start

TASK-015 通过；TASK-016 可以开始。
