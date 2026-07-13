# TASK Report - TASK-008

## 1. TASK ID

TASK-008 - Interview infrastructure/interfaces/runtime/tests 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-interview-service/cmd/interview-service/main.go`
- `smart-recruit-interview-service/internal/application/port/interview_ports.go`
- `smart-recruit-interview-service/internal/application/query/queries.go`
- `smart-recruit-interview-service/internal/application/service/interview_service.go`
- `smart-recruit-interview-service/internal/application/service/interview_service_test.go`
- `smart-recruit-interview-service/internal/docs/contract_inventory.md`
- `smart-recruit-interview-service/internal/domain/model/feedback.go`
- `smart-recruit-interview-service/internal/domain/model/feedback_test.go`
- `smart-recruit-interview-service/internal/infrastructure/client/application_adapter.go`
- `smart-recruit-interview-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-interview-service/internal/infrastructure/client/staff_directory.go`
- `smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher.go`
- `smart-recruit-interview-service/internal/infrastructure/mq/outbox_publisher_test.go`
- `smart-recruit-interview-service/internal/infrastructure/persistence/interview_repository.go`
- `smart-recruit-interview-service/internal/interfaces/grpc/interview_server.go`
- `smart-recruit-interview-service/internal/interfaces/grpc/interview_server_test.go`
- `smart-recruit-interview-service/internal/interfaces/mapper/interview_mapper.go`
- `.spec/microservice-ddd-evolution/reports/TASK-008-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-008-evidence.json`

## 3. Change Summary by File

- `cmd/interview-service/main.go`: runtime wiring 改为构造本地 application service、infrastructure adapters 和 `internal/interfaces/grpc.Server`，不再通过 shared `service.NewServices` / `service.InterviewService` 注册 InterviewService。
- `internal/application/port/interview_ports.go`: 新增 staff directory port，补齐 staff user/page DTO；保持 application/lifecycle/outbox/authorizer ports 在 application 边界内。
- `internal/application/query/queries.go`: 新增 get/list interviewers/list application/list my/list candidate query DTO。
- `internal/application/service/interview_service.go`: 新增 get/list 查询编排，保留候选人 internal_note 过滤、面试官 has_feedback 聚合、权限和 scope 检查。
- `internal/application/service/interview_service_test.go`: 新增 ListMy has_feedback 聚合、ListCandidate internal_note 过滤测试。
- `internal/domain/model/feedback.go`: 将 feedback 校验错误拆分为 recommendation required、invalid recommendation、score out of range，便于 gRPC 兼容映射。
- `internal/domain/model/feedback_test.go`: 更新 typed feedback error 断言。
- `internal/infrastructure/persistence/interview_repository.go`: 新增本地 repository adapter，封装 legacy shared `InterviewRepo`，并提供事务上下文桥接。
- `internal/infrastructure/client/application_adapter.go`: 新增 application snapshot 与 lifecycle adapter，复用 shared `ApplicationRepo` 作为 transitional infrastructure debt。
- `internal/infrastructure/client/authorizer.go`: 新增 metadata actor 校验、permission、schedule/read scope adapter，复用 shared authz/job/application/interview repos。
- `internal/infrastructure/client/staff_directory.go`: 新增 staff interviewer directory adapter。
- `internal/infrastructure/mq/outbox_publisher.go`: 新增 Interview outbox publisher，保持 legacy envelope/top-level payload 兼容；修复 event id 随机数错误吞掉的问题。
- `internal/infrastructure/mq/outbox_publisher_test.go`: 覆盖 legacy-compatible outbox envelope、routing key、producer、nested payload 和 interview 字段。
- `internal/interfaces/mapper/interview_mapper.go`: 新增 RFC3339 解析、Interview/Feedback/Staff protobuf mapper。
- `internal/interfaces/grpc/interview_server.go`: 新增本地 protobuf `InterviewServiceServer` 实现，映射 schedule/update/cancel/batch/get/list/feedback 的 legacy response code/message 语义。
- `internal/interfaces/grpc/interview_server_test.go`: 覆盖时间解析、schedule 成功、batch zero、feedback typed errors、get not found、list mapping/forbidden/has_feedback/candidate-safe rows。
- `internal/docs/contract_inventory.md`: 更新 TASK-008 后 Interview runtime、本地 adapters、remaining transitional debt 和测试证据。
- `.spec/microservice-ddd-evolution/reports/TASK-008-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-008-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-008` 通过。

变更均在 TASK-008 allowed files 内：`smart-recruit-interview-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-004 至 FR-009：Interview 服务完成 infrastructure、interfaces、runtime wiring 和本地测试收敛；domain/application 不引入 GORM/proto/gRPC 等外层依赖。
- FR-016：active runtime 不再注册 shared `service.InterviewService`，直接注册本地 implementation；shared `model/repository` 仅作为 infrastructure transitional adapter debt 记录。
- CR-001/CR-002/CR-005/CR-006：未修改 HTTP API、protobuf、Gateway route mode、schema、table ownership、Outbox/Inbox schema。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：补齐 Interview persistence/client/outbox adapters、gRPC interface、runtime composition。
- 8. Compatibility Strategy：保持 protobuf response 形状和旧错误消息；runtime 切换仅发生在 Interview service 本地。
- 9. Error Handling and Fallback Design：typed errors 在 gRPC 层映射到 legacy code/message；`crypto/rand` 错误不再吞掉。
- 11. Testing Strategy：新增 infrastructure mq、interfaces grpc、application query/list 兼容测试，并继续运行 service-level `go test ./...`。

## 7. Acceptance Comparison Result

- Interview runtime 使用本地 implementation 注册 `InterviewService`：已完成，`cmd/interview-service/main.go` 构造 `interviewgrpc.NewServer(interviewService)` 并传入 `interviewruntime.New`。
- 对共享 `service.InterviewService` 的直接依赖消除或记录债务：Go 代码搜索 `smart-recruit-commons/service|service.InterviewService|NewServices|services.Interview|buildDomainServices` 无命中；shared `model/repository` adapter debt 已记录在 `internal/docs/contract_inventory.md`。
- gRPC 语义兼容：保持 schedule/update/cancel/batch/get/list/feedback response code/message 和候选人字段过滤；新增 tests 覆盖关键路径。
- Out-of-Scope 遵守：未删除 shared 旧实现，未修改 Recruitment/Offer 行为。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 输出 8 个已跟踪 service 文件；未跟踪新增 service/report/evidence 文件由 `git status --short` 和 evidence `changed_files` 完整记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-008` | 0 | passed | `Scope check passed for TASK-008. Changed files: 19`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-interview-service` 内运行 `go test ./...` 并通过。 |
| `go test ./...` in `smart-recruit-interview-service` | 0 | passed | Interview service 全 package 通过，含新增 application/interface/infrastructure tests。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |

额外核验：

- `rg -n "smart-recruit-commons/service|service\\.InterviewService|NewServices|services\\.Interview|buildDomainServices" smart-recruit-interview-service -g'*.go'` exit 1，无 Go 代码命中，active runtime 直接 shared service 依赖已消除。
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree d38e121d6e3656a55e70f789a287699b6881c574` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-interview-service/cmd/interview-service/main.go
    - smart-recruit-interview-service/internal/infrastructure/**
    - smart-recruit-interview-service/internal/interfaces/**
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
  reason: TASK-008 将 Interview runtime 切到 smart-recruit-interview-service 本地实现，但 active knowledge 仍描述旧 logic-grpc-service 拓扑；当前 TASK scope 不允许修改 .knowledge/**。
```

## 10. Self-review and Repair

独立只读 self-review 第 1 轮 verdict: 不通过。

发现：

- M1：TASK-008 report/evidence 缺失。
- M2：ListInterviewers/ListApplication/ListMy/ListCandidate 的 gRPC/list/read 兼容测试不足。
- L1：`newEventID` 忽略 `crypto/rand.Read` 错误。

修复：

- 新增本 report/evidence，记录 scope、checks、knowledge impact 和 review/repair。
- 增加 gRPC list mapping/forbidden/has_feedback/candidate-safe rows 测试。
- 增加 application list 测试，覆盖候选人 internal_note 过滤与面试官 has_feedback 聚合。
- `newEventID` 改为返回 `(string, error)`，`buildEvent` 向上返回随机数生成错误。

独立只读 self-review 第 2 轮 verdict: 通过。

二轮复核确认：

- 第一轮 M1/M2/L1 均已修复。
- Runtime 已通过本地 `interviewgrpc.NewServer(interviewService)` 注册 InterviewService。
- Go 代码中无 shared `service.InterviewService` / `NewServices` / `buildDomainServices` / `services.Interview` active dependency。
- 当前 19 个变更文件均在 TASK-008 scope 内，未修改 shared/proto/schema/其他服务。

## 11. Risks

- `smart-recruit-commons/model` 与 `repository` 仍作为 infrastructure transitional bridge 使用，后续 TASK-029 才能收缩 shared kernel。
- Interview lifecycle 仍同步写 ApplicationRepo；这是迁移期 adapter debt，未新增跨服务 API 或事件 schema。
- Knowledge active 文档仍有旧路径 source_refs 债务；本 TASK 已记录 candidate_required，但不越界修改知识库。

## 12. Follow-up Items

- TASK-009 开始 Notification DDD 骨架与 Outbox/Inbox 契约盘点。
- 后续 cleanup TASK 继续收缩 shared model/repository 依赖，最终在 TASK-029/TASK-030 处理 commons rename。

## 13. Whether the Next TASK Can Start

可以开始 TASK-009。
