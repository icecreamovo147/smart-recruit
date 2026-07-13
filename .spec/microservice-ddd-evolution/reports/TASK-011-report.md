# TASK Report - TASK-011

## 1. TASK ID

TASK-011 - Notification infrastructure/interfaces/runtime/tests 收敛。

## 2. Modified File List

实际变更文件：

- `smart-recruit-notification-service/cmd/notification-service/main.go`
- `smart-recruit-notification-service/internal/infrastructure/cache/notification_cache.go`
- `smart-recruit-notification-service/internal/infrastructure/client/authorizer.go`
- `smart-recruit-notification-service/internal/infrastructure/email/email.go`
- `smart-recruit-notification-service/internal/infrastructure/mq/consumer.go`
- `smart-recruit-notification-service/internal/infrastructure/persistence/notification_repository.go`
- `smart-recruit-notification-service/internal/interfaces/grpc/notification_server.go`
- `smart-recruit-notification-service/internal/interfaces/mapper/notification.go`
- `smart-recruit-notification-service/internal/runtime/runtime.go`
- `smart-recruit-notification-service/internal/runtime/runtime_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-011-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-011-evidence.json`

## 3. Change Summary by File

- `cmd/notification-service/main.go`: 将 active Notification runtime 装配切换到本地 application service、GORM persistence、cache、email、MQ consumers 和 gRPC server；不再构造 shared `NotificationRuntime` / `NotificationService`。
- `internal/infrastructure/cache/notification_cache.go`: 新增 Redis unread cache 与 realtime publisher adapter，复用现有 cache client 语义。
- `internal/infrastructure/client/authorizer.go`: 新增本地 actor verifier，保留 gRPC metadata fail-closed actor match 行为。
- `internal/infrastructure/email/email.go`: 新增 shared email renderer/sender 到本地 application port 的 adapter。
- `internal/infrastructure/mq/consumer.go`: 新增 notification/email RabbitMQ consumers，复用本地 Inbox claim/processed/failed 幂等流程并调用本地 application service。
- `internal/infrastructure/persistence/notification_repository.go`: 新增本地 GORM notification、email log、inbox、user directory adapters，使用本地 row struct 映射到本地 domain model。
- `internal/interfaces/grpc/notification_server.go`: 新增本地 protobuf `NotificationService` server，调用本地 application service。
- `internal/interfaces/mapper/notification.go`: 新增 proto request/response 与 application command/query/domain model mapper。
- `internal/runtime/runtime.go`: 将 runtime facade 改为本地 components + Start lifecycle，注册本地 gRPC server 并启动 outbox/consumers。
- `internal/runtime/runtime_test.go`: 更新 runtime tests，覆盖本地组件检查和缺失 MQ 启动错误。
- `.spec/.../TASK-011-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-011-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-011` 通过。

变更均在 TASK-011 allowed files 内：`smart-recruit-notification-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-domain-go/**`、`smart-recruit-proto/**`、`db.sql`、migration、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-004/FR-005/FR-006：Notification 新增本地 infrastructure adapters、interfaces/grpc mapper/server 和 runtime lifecycle。
- FR-010：Notification persistence、unread/realtime、email coordination、Outbox/Inbox consumer 运行路径已接入本地 service。
- FR-016：Notification active runtime 不再直接依赖 shared `service.NotificationService` 或 `service.NotificationRuntime`。
- CR-001/CR-002/CR-006：未修改 HTTP API、protobuf、message schema、RabbitMQ queue、Redis channel、数据库 schema 或 Gateway 行为。

## 6. SDD Comparison Result

符合 SDD：

- 3.1 Target Service Shape：本 TASK 填充 `infrastructure`、`interfaces` 和 `runtime` 职责边界。
- 6. Algorithm or Workflow Changes：gRPC -> mapper -> application service -> persistence/cache/email/MQ adapter 的路径已形成；consumer 继续通过 Inbox 幂等保护。
- 8. Compatibility Strategy：服务启动、health/metrics/Nacos/internal auth/TLS 路径保持原有 main.go 结构。
- 11. Testing Strategy：运行 Notification 服务 `go test ./...`、scope、agent-check、table ownership、backend boundary。

## 7. Acceptance Comparison Result

- Notification runtime 使用本地 implementation：已完成，`buildNotificationRuntime` 构造本地 application service、persistence/cache/email/MQ adapters、local gRPC server 和 local runtime facade。
- Outbox/Inbox retry/DLQ/idempotency 兼容：RabbitMQ retry/DLQ 继续由 existing `mq.Conn.Consume` 管理；notification/email consumer 使用本地 `RunWithInbox`；Outbox publisher 暂时复用 shared generic publisher，记录为后续 shared-kernel 收敛债务。
- 对共享 notification implementation 的依赖清除或记录债务：已清除 active shared `NotificationService` / `NotificationRuntime` 依赖；仍复用 shared generic `OutboxPublisher`、shared email/cache packages 和 pagination helper，作为 transitional shared-kernel bridge。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `go test ./...` in `smart-recruit-notification-service` | 0 | passed | Notification service 全 package 测试通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |
| `git diff --name-only` | 0 | passed | Tracked diff 显示 3 个已跟踪文件；新增文件通过 `git status --short` 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-011` | 0 | passed | `Scope check passed for TASK-011. Changed files: 12`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-notification-service` 内运行 `go test ./...` 并通过。 |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree a6db3411c2ca9ee244b52785150002695d350ae1` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-notification-service/cmd/notification-service/main.go
    - smart-recruit-notification-service/internal/infrastructure/**
    - smart-recruit-notification-service/internal/interfaces/**
    - smart-recruit-notification-service/internal/runtime/**
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/notification-outbox.md
    - .knowledge/runbooks/event-replay-dead-letter.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: TASK-011 changes active Notification runtime/consumer wiring, while active knowledge still references legacy logic-grpc-service notification/outbox topology. Current TASK scope does not allow .knowledge/** edits.
```

## 10. Self-review and Repair

self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 未发现 Critical/High/Medium/Low 问题。
- 变更均在 `smart-recruit-notification-service/**` 与报告目录内。
- 未修改 shared/proto/schema/package/lockfile/global config。
- Active runtime 不再构造 shared `NotificationRuntime` 或 shared `NotificationService`。
- gRPC mapper/server 保留现有 response code/message/list/unread/summary/mark read semantics。
- Inbox idempotency、RabbitMQ retry/DLQ、cache realtime 发布和 email send failure 行为均通过本地 adapters 接入。
- `go test ./...`、scope check、agent-check、backend boundary、table ownership 检查均通过。

## 11. Risks

- `service.NewOutboxPublisher` 仍来自 `smart-recruit-domain-go/service`，本 TASK 记录为 shared generic outbox bridge；后续 TASK-029 shared-kernel 收敛时应迁出或确认其归属。
- Email renderer/sender 与 notification Redis cache 继续复用 shared infrastructure packages；这符合当前 transitional shared-kernel 策略，但最终 commons-ready 阶段需要分类。
- 本 TASK 未执行真实 RabbitMQ/SMTP integration；`go test ./...` 覆盖编译与本地接口，真实外部依赖由现有 runtime config 和 MQ retry/DLQ 语义保持。
- Active knowledge source_refs 有既有路径债务，knowledge impact detector 无法完成。

## 12. Follow-up Items

- TASK-012 是 Identity 安全契约盘点，`requiresHumanConfirmation=true`，pipeline 必须等待用户确认后继续。
- 后续 shared cleanup 需处理 shared outbox/email/cache/pagination 的归属。

## 13. Whether the Next TASK Can Start

TASK-011 通过；但 TASK-012 需要人工确认后才能开始。
