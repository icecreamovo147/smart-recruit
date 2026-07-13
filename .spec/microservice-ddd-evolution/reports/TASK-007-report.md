# TASK Report - TASK-007

## 1. TASK ID

TASK-007 - Interview domain/application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-interview-service/internal/docs/contract_inventory.md`
- `smart-recruit-interview-service/internal/domain/model/status.go`
- `smart-recruit-interview-service/internal/domain/model/interview.go`
- `smart-recruit-interview-service/internal/domain/model/feedback.go`
- `smart-recruit-interview-service/internal/domain/model/interview_test.go`
- `smart-recruit-interview-service/internal/domain/model/feedback_test.go`
- `smart-recruit-interview-service/internal/domain/service/interview_policy.go`
- `smart-recruit-interview-service/internal/domain/service/interview_policy_test.go`
- `smart-recruit-interview-service/internal/domain/repository/interview_repository.go`
- `smart-recruit-interview-service/internal/application/command/commands.go`
- `smart-recruit-interview-service/internal/application/query/queries.go`
- `smart-recruit-interview-service/internal/application/port/interview_ports.go`
- `smart-recruit-interview-service/internal/application/service/interview_service.go`
- `smart-recruit-interview-service/internal/application/service/interview_service_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-007-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-007-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/status.go`: 新增 Interview 本地 application/interview status、terminal status 判断和状态标签。
- `internal/domain/model/interview.go`: 新增 Interview 聚合、schedule 默认值、patch、cancel 和 complete 状态规则。
- `internal/domain/model/feedback.go`: 新增 feedback model、推荐结论/分数校验、重复/终态/assignment 领域错误。
- `internal/domain/model/*_test.go`: 覆盖 schedule 默认值、重复取消保护、feedback 推荐结论和分数校验。
- `internal/domain/service/interview_policy.go`: 新增 schedule/cancel/feedback lifecycle transition policy，保持 legacy schedule no-op 兼容分支。
- `internal/domain/service/interview_policy_test.go`: 覆盖 schedule transition/no-op 状态矩阵、cancel no-op、feedback no-op。
- `internal/domain/repository/interview_repository.go`: 新增本地 Interview repository port 和 transactional writer port。
- `internal/application/command/commands.go`: 新增 schedule/update/cancel/batch cancel/feedback command DTO。
- `internal/application/query/queries.go`: 新增 Interview query DTO，为 TASK-008 interface adapter 使用。
- `internal/application/port/interview_ports.go`: 新增 authorizer、application snapshot、application lifecycle、outbox publisher、clock ports。
- `internal/application/service/interview_service.go`: 新增本地 application service，编排 schedule/update/cancel/batch cancel/feedback/get feedback 的权限、状态、事务、lifecycle 和 outbox 语义。
- `internal/application/service/interview_service_test.go`: 覆盖 schedule/update/cancel/batch cancel/feedback 的状态与 outbox 语义，包含 schedule legacy no-op 兼容分支。
- `internal/docs/contract_inventory.md`: 更新 TASK-007 后 Interview domain/application 本地化状态和剩余 TASK-008 runtime 切换边界。
- `.spec/microservice-ddd-evolution/reports/TASK-007-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-007-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-007` 通过。

变更均在 TASK-007 allowed files 内：`smart-recruit-interview-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、schema/migration、deployment、package/lockfile 或全局配置。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002：Interview domain model/status/feedback/policy 不依赖 GORM、proto、gRPC、Redis、RabbitMQ、HTTP、Nacos、OSS、SMTP 或外部 SDK。
- FR-003/FR-009：Application 层通过 ports 编排 schedule、update、cancel、batch cancel、feedback 的事务、权限、lifecycle 与 outbox 时机。
- FR-016/FR-017：新增本地 repository port 和 application ports，为后续 TASK-008 移除 shared `service.InterviewService` active wiring 做准备。
- CR-001/CR-002/CR-005/CR-006：未修改 HTTP API、protobuf、Gateway route mode、schema、table ownership 或 Outbox/Inbox schema。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：完成 Interview domain model/value object/state rules、repository interface、application command/query/ports/service。
- 6. Algorithm or Workflow Changes：保持旧 schedule no-op lifecycle 兼容分支、cancel 仅在 interview_pending/interviewing 推进、feedback 从 interview_pending 推进到 interviewing。
- 9. Error Handling and Fallback Design：domain/application 返回 typed errors，不吞异常；transport response mapping 留给 TASK-008。

## 7. Acceptance Comparison Result

- Interview domain/application 已本地化。
- schedule/update/cancel/batch cancel/feedback 规则被本地 service 和 tests 覆盖。
- Domain 不依赖外层技术；application 仅依赖本地 domain 和 ports。
- 未做 runtime 完整切换，未改 proto/schema，符合 Out-of-Scope。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 输出已跟踪修改 `smart-recruit-interview-service/internal/docs/contract_inventory.md`；新增 domain/application/report/evidence 文件由 `changed_files` 完整记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-007` | 0 | passed | `Scope check passed for TASK-007. Changed files: 16`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-interview-service` 内运行 `go test ./...` 并通过。 |
| `go test ./...` in `smart-recruit-interview-service` | 0 | passed | Interview service 所有 package 编译/测试通过，含新增 domain/application tests。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `rg -n "gorm|grpc|smart-recruit-proto|redis|rabbit|nacos|http|oss|mysql" smart-recruit-interview-service/internal/domain smart-recruit-interview-service/internal/application -g'*.go'` | 0 | passed | 仅命中测试数据 URL 字符串；未发现 domain/application 外层技术 import。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 1d96a96fb636dcc143086b266bee4a52e946b29f` exit 1。
- 失败原因仍是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 已不存在；TASK-001 已创建 `.knowledge/inbox/microservice-ddd-evolution-boundaries.md` 作为 candidate，本 TASK scope 不允许修改 `.knowledge/**`。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-interview-service/internal/domain/**
    - smart-recruit-interview-service/internal/application/**
    - smart-recruit-interview-service/internal/docs/contract_inventory.md
  reviewed_documents:
    - .knowledge/architecture/system-overview.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/domains/notification-outbox.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: 当前 active knowledge 尚未正式更新为 smart-recruit-* 微服务根；本 TASK 只能记录 candidate_required，不能越界修改知识库。
```

## 10. Self-review and Repair

独立只读 self-review 第 1 轮 verdict: 不通过。

发现：

- H1：`ScheduleInterview` 初版对不可推进 application status 返回错误，改变 legacy “创建面试但不推进 lifecycle”语义。
- H2：缺少 TASK-007 report/evidence 和 knowledge impact 记录。
- M1：状态矩阵测试覆盖不足。

修复：

- `domain/service.ScheduleTransition` 改为仅对 viewed/screen_passed/interview_cancelled/interview_passed 返回 transition；其他状态 no-op 且不报错。
- 新增 `domain/service/interview_policy_test.go` 状态矩阵测试。
- 新增 application service 测试覆盖 schedule 在 `screening` 状态下仍创建面试和发送 outbox 但不调用 lifecycle。
- 新增本 report/evidence 并记录 knowledge impact。

独立只读 self-review 第 2 轮 verdict: 通过。

## 11. Risks

- Active runtime 仍使用 shared `service.InterviewService`，本地 application service 将在 TASK-008 通过 infrastructure/interface/runtime adapter 接入。
- Listing/query DTO 已建立，但本 TASK 重点覆盖 schedule/update/cancel/batch cancel/feedback；完整 gRPC listing compatibility 测试留给 TASK-008。
- Outbox payload 仍为 port-level 语义模型，具体 legacy envelope/schema 兼容将在 TASK-008 infrastructure/mq adapter 中验证。
- Active knowledge 仍有旧路径 source_refs 债务；本 TASK scope 不允许修复 `.knowledge/**`。

## 12. Follow-up Items

- TASK-008 完成 Interview persistence、client/outbox adapters、gRPC interface、runtime wiring 和兼容测试，消除 active runtime 对 shared `service.InterviewService` 的直接依赖。

## 13. Whether the Next TASK Can Start

可以开始 TASK-008。
