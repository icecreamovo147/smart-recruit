# TASK Report - TASK-017

## 1. TASK ID

TASK-017 - Recruitment application/collaboration/taxonomy 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-recruitment-service/internal/application/command/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/dto/recruitment.go`
- `smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service.go`
- `smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service_test.go`
- `smart-recruit-recruitment-service/internal/application/service/job_candidate_resume_service_test.go`
- `smart-recruit-recruitment-service/internal/domain/model/recruitment.go`
- `smart-recruit-recruitment-service/internal/domain/policy/recruitment.go`
- `smart-recruit-recruitment-service/internal/domain/repository/recruitment.go`
- `.spec/microservice-ddd-evolution/reports/TASK-017-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-017-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/recruitment.go`: 新增 application status keys、legacy status mapping、HR labels、terminal status、application/transition、notification payload、collaboration、taxonomy、usage stats 本地模型。
- `internal/domain/policy/recruitment.go`: 新增 application 状态机、状态变更校验、apply 前置条件、通知内容、department create policy、tag 默认颜色、usage stats 时间范围策略。
- `internal/domain/repository/recruitment.go`: 新增 application、collaboration、taxonomy、usage stats repository/application ports。
- `internal/application/command/recruitment.go`: 新增 apply/status/collaboration/taxonomy/usage stats command DTO。
- `internal/application/dto/recruitment.go`: 新增 application/collaboration/taxonomy/usage stats result DTO。
- `internal/application/service/application_collaboration_taxonomy_service.go`: 新增 ApplicationLifecycleService、CollaborationService、TaxonomyService、UsageStatsService，本地化 apply、status update、notification/email outbox、collaboration auth gates、department create、usage stats permission/time-range/dimension。
- `internal/application/service/application_collaboration_taxonomy_service_test.go`: 覆盖 apply/update status/collaboration/taxonomy/usage stats targeted tests。
- `internal/application/service/job_candidate_resume_service_test.go`: 扩展 TASK-016 fakes，支持 TASK-017 application service 读取 profile/resume/job。
- `.spec/.../TASK-017-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-017-evidence.json`: 新增机器可读 evidence。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-017` 通过。

变更均在 TASK-017 allowed files 内：`smart-recruit-recruitment-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002 / FR-003：Application lifecycle、Collaboration、Taxonomy、UsageStats 本地 domain/application 层已建立，外部依赖通过端口表达。
- FR-012：保留 Recruitment application/collaboration/taxonomy/usage stats 核心域边界和跨服务依赖。
- FR-019：未引入新的 snapshot API、事件 schema、proto/schema 或 public API 行为变更。

## 6. SDD Comparison Result

符合 SDD：

- 6. Algorithm or Workflow Changes：本地状态机复刻 legacy status mapping、terminal/re-pass、reason requirement、notification/email outbox 语义。
- 8. Compatibility Strategy：active runtime 未切换，protobuf/gateway/schema 不变；后续 TASK-018 负责 adapter/runtime 收敛。
- 11. Testing Strategy：新增 targeted tests 覆盖 apply/update status/collaboration/taxonomy/usage stats。

## 7. Acceptance Comparison Result

- Application 状态机、collaboration、taxonomy、usage stats 本地化：已完成，本地模型、policy、ports 和 application services 已新增。
- 保留 transitions、outbox、权限和分页语义：状态机、legacy mapping、terminal/re-pass、通知/email outbox、collaboration permission/candidate access、usage stats permission/time range 已端口化保留；分页 repository 语义保留在端口边界，未改变 active runtime。
- 覆盖 apply/update status/collaboration/taxonomy 测试：已完成，并额外覆盖 usage stats 默认维度/时间范围。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./internal/application/service ./internal/domain/policy` in `smart-recruit-recruitment-service` | 0 | passed | Targeted apply/update status、collaboration、taxonomy、usage stats、policy 测试通过。 |
| `go test ./...` in `smart-recruit-recruitment-service` | 0 | passed | Recruitment 全 package 测试通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | Tracked diff listed 6 modified files; two new application service files are listed by `git status --short` and evidence `changed_files`。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-017` | 0 | passed | `Scope check passed for TASK-017. Changed files: 10`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；检测到 Recruitment module 变更并运行 `go test ./...` 通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree f228364a9635f89991fff945cb2983e5af94b9ad` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-recruitment-service/internal/application/service/application_collaboration_taxonomy_service.go
    - smart-recruit-recruitment-service/internal/domain/model/recruitment.go
    - smart-recruit-recruitment-service/internal/domain/policy/recruitment.go
    - smart-recruit-recruitment-service/internal/domain/repository/recruitment.go
  reviewed_documents:
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/debug-recruitment-lifecycle.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-recruitment-application-collaboration.md
  coverage_gap: false
  reason: TASK-017 adds local application/collaboration/taxonomy/usage stats domain/application anchors while active knowledge still references legacy logic-grpc-service/web-gin-service source paths. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 变更均在 `smart-recruit-recruitment-service/**` 与报告目录内。
- 未修改 shared domain、proto、schema、workspace、package manifest、lockfile、gateway、deployment 或配置。
- 未新增 snapshot API 或事件 schema。
- Application outbox topic/event type 仍使用 `application.notification_requested`、`notification.create`、`application.email_requested`、`email.send`。
- Collaboration 操作保留 candidate access gate 和具体 permission gate。
- `go test ./...`、targeted tests、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 11. Risks

- Active runtime 仍使用 shared implementation；TASK-018 需要接入本地 adapters/interfaces/runtime 并验证 protobuf-facing 兼容。
- 本地 ApplicationRepository 端口把分页/list 语义留给 adapter 层承接，当前 TASK 重点覆盖状态机和关键写路径。
- UsageStats 当前仍是 platform-owned log read model；本 TASK 仅本地化 Recruitment 暴露的 query orchestration。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-018 可开始 Recruitment infrastructure/interfaces/runtime/tests 收敛。
- TASK-018 需要保证 active runtime 使用本地 Job/Candidate/Application/Collaboration/Admin 子集，并记录剩余 shared dependency debt。

## 13. Whether the Next TASK Can Start

TASK-017 通过；TASK-018 可以开始。
