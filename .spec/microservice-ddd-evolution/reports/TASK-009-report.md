# TASK Report - TASK-009

## 1. TASK ID

TASK-009 - Notification DDD 骨架与契约盘点。

## 2. Modified File List

实际变更文件：

- `smart-recruit-notification-service/internal/domain/doc.go`
- `smart-recruit-notification-service/internal/domain/model/doc.go`
- `smart-recruit-notification-service/internal/domain/event/doc.go`
- `smart-recruit-notification-service/internal/domain/repository/doc.go`
- `smart-recruit-notification-service/internal/domain/service/doc.go`
- `smart-recruit-notification-service/internal/application/doc.go`
- `smart-recruit-notification-service/internal/application/command/doc.go`
- `smart-recruit-notification-service/internal/application/query/doc.go`
- `smart-recruit-notification-service/internal/application/port/doc.go`
- `smart-recruit-notification-service/internal/application/service/doc.go`
- `smart-recruit-notification-service/internal/infrastructure/doc.go`
- `smart-recruit-notification-service/internal/infrastructure/persistence/doc.go`
- `smart-recruit-notification-service/internal/infrastructure/mq/doc.go`
- `smart-recruit-notification-service/internal/infrastructure/cache/doc.go`
- `smart-recruit-notification-service/internal/infrastructure/email/doc.go`
- `smart-recruit-notification-service/internal/infrastructure/client/doc.go`
- `smart-recruit-notification-service/internal/interfaces/doc.go`
- `smart-recruit-notification-service/internal/interfaces/grpc/doc.go`
- `smart-recruit-notification-service/internal/interfaces/mapper/doc.go`
- `smart-recruit-notification-service/internal/interfaces/event/doc.go`
- `smart-recruit-notification-service/internal/docs/contract_inventory.md`
- `.spec/microservice-ddd-evolution/reports/TASK-009-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-009-evidence.json`

## 3. Change Summary by File

- `internal/domain/**/doc.go`: 创建 Notification 本地 domain/package 边界，预留 model、event、repository、service 职责。
- `internal/application/**/doc.go`: 创建 Notification application/package 边界，预留 command、query、port、service 职责。
- `internal/infrastructure/**/doc.go`: 创建 Notification infrastructure/package 边界，预留 persistence、mq、cache、email、client 适配职责。
- `internal/interfaces/**/doc.go`: 创建 Notification inbound adapters 边界，预留 grpc、mapper、event 职责。
- `internal/docs/contract_inventory.md`: 盘点 Notification protobuf/Gateway/runtime、shared dependencies、notification/email/outbox/inbox/SSE 契约、队列、表访问和测试覆盖。
- `.spec/microservice-ddd-evolution/reports/TASK-009-report.md`: 新增 TASK 报告。
- `.spec/microservice-ddd-evolution/reports/TASK-009-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-009` 通过。

变更均在 TASK-009 allowed files 内：`smart-recruit-notification-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、deployment、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-001：Notification 服务已具备本地 `domain/application/infrastructure/interfaces/runtime` 分层包形状。
- FR-010：完成 notification persistence、unread count、summary、mark read、email coordination、SSE/realtime、Outbox/Inbox 相关契约盘点。
- FR-016/FR-017：记录当前对 shared `service/model/repository` 的依赖和后续迁移债务；本 TASK 未新增 shared 业务依赖。
- CR-001/CR-002/CR-006：未修改 HTTP API、protobuf、Gateway route mode、Outbox/Inbox schema、RabbitMQ queue、Redis channel 或邮件发送行为。

## 6. SDD Comparison Result

符合 SDD：

- 3.4 Per-Service Migration Pattern：完成 Notification 服务第一步盘点和本地 DDD 目录骨架。
- 5. API and Interface Changes：只记录 `NotificationService` protobuf 和 Gateway routes，不改变接口。
- 8. Compatibility Strategy：保留当前同步查询、Outbox/Inbox、RabbitMQ retry/DLQ、Redis SSE 和 SMTP 行为。
- 10. Observability and Debug Output Design：记录 outbox retry/DLQ、consumer inbox、cache invalidation、SSE publish 和 email log 的诊断点。

## 7. Acceptance Comparison Result

- Notification DDD 骨架存在：已创建 domain/application/infrastructure/interfaces 包边界。
- notification/email/Outbox/Inbox/SSE 依赖盘点完成：已在 `internal/docs/contract_inventory.md` 记录 runtime deps、queues、tables、idempotency、retry/DLQ、Redis channel、email skip/send failure 语义。
- 不改变通知 API 行为：未修改 runtime、protobuf、Gateway、schema、MQ、Redis 或 SMTP 代码。
- 邮件测试不会触发真实发送：本 TASK 只新增 doc packages 和 markdown 盘点；`go test ./...` 未启动 `--serve`，未初始化 SMTP send path。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `git diff --name-only` | 0 | passed | 输出为空，因为当前 TASK 文件均为未跟踪新增文件；完整列表由 `git status --short` 和 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-009` | 0 | passed | `Scope check passed for TASK-009. Changed files: 23`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-notification-service` 内运行 `go test ./...` 并通过。 |
| `go test ./...` in `smart-recruit-notification-service` | 0 | passed | Notification service 全 package 编译/测试通过，新增 doc packages 均可编译。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree d2823d905767e4720dc54603492f0c99a706d28f` exit 1。
- 失败原因仍是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-notification-service/internal/domain/**
    - smart-recruit-notification-service/internal/application/**
    - smart-recruit-notification-service/internal/infrastructure/**
    - smart-recruit-notification-service/internal/interfaces/**
    - smart-recruit-notification-service/internal/docs/contract_inventory.md
  reviewed_documents:
    - .knowledge/architecture/system-overview.md
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/recruitment.md
    - .knowledge/domains/recruitment-lifecycle.md
    - .knowledge/domains/notification-outbox.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: TASK-009 开始 smart-recruit-notification-service 本地 DDD 边界，但 active knowledge 仍描述旧 logic-grpc-service 拓扑；当前 TASK scope 不允许修改 .knowledge/**。
```

## 10. Self-review and Repair

独立只读 self-review 第 1 轮 verdict: 通过。

低风险记录：

- `users` email receiver lookup 是 transitional direct read；当前 table ownership manifest 未声明 Notification 读取 `users`，后续 TASK-010/TASK-011 需通过 owner-service query/client port 或债务记录处理。
- `AuthzRepo` 已作为 runtime dependency wiring 存在，但当前 Notification read path 主要使用 gRPC metadata actor verification，未单独查询权限表；已在 contract inventory 中澄清。

## 11. Risks

- Active runtime 仍使用 shared `service.NotificationRuntime`，这是 TASK-009 明确允许的迁移前基线；TASK-010/TASK-011 将逐步迁移。
- Notification 是异步一致性关键路径；后续迁移必须保留 `event_outbox` retry/DLQ、`event_inbox` 幂等、Redis unread/SSE 和 `email_logs` 邮件幂等语义。
- `users` email lookup 仍是未在 ownership manifest 中声明的 transitional read debt；后续迁移应收敛为 client/query port 或明确债务。
- Active knowledge 仍有旧路径 source_refs 债务；本 TASK 已记录 candidate_required，但不越界修改知识库。

## 12. Follow-up Items

- TASK-010 迁移 Notification domain/application，用本地 tests 覆盖 unread、summary、mark read/all read、consumer idempotency 和 email coordination，并处理 `users` email lookup 的 owner 边界。
- TASK-011 迁移 infrastructure/interfaces/runtime adapters，并证明 no-real-SMTP 测试路径。

## 13. Whether the Next TASK Can Start

独立 self-review 通过且 evidence validator 通过后，可以开始 TASK-010。
