# TASK Report - TASK-010

## 1. TASK ID

TASK-010 - Notification domain/application 迁移。

## 2. Modified File List

实际变更文件：

- `smart-recruit-notification-service/internal/domain/model/notification.go`
- `smart-recruit-notification-service/internal/domain/model/email.go`
- `smart-recruit-notification-service/internal/domain/model/notification_test.go`
- `smart-recruit-notification-service/internal/domain/repository/notification.go`
- `smart-recruit-notification-service/internal/application/command/notification.go`
- `smart-recruit-notification-service/internal/application/query/notification.go`
- `smart-recruit-notification-service/internal/application/port/notification.go`
- `smart-recruit-notification-service/internal/application/service/notification_service.go`
- `smart-recruit-notification-service/internal/application/service/email_service.go`
- `smart-recruit-notification-service/internal/application/service/inbox.go`
- `smart-recruit-notification-service/internal/application/service/time.go`
- `smart-recruit-notification-service/internal/application/service/notification_service_test.go`
- `.spec/microservice-ddd-evolution/reports/TASK-010-report.md`
- `.spec/microservice-ddd-evolution/reports/TASK-010-evidence.json`

## 3. Change Summary by File

- `internal/domain/model/notification.go`: 新增 Notification 本地领域模型、默认 account type、created/read 状态规则和 realtime event DTO。
- `internal/domain/model/email.go`: 新增 EmailLog、EmailRecipient 与 sent/skipped/failed 状态常量。
- `internal/domain/model/notification_test.go`: 覆盖 account type 默认值、MarkRead 幂等和 CreatedAt 初始化规则。
- `internal/domain/repository/notification.go`: 定义 NotificationRepository、EmailLogRepository 和 InboxRepository 本地 port。
- `internal/application/command/notification.go`: 定义 create notification、notification consumer、mark read/all read 和 email consumer 命令。
- `internal/application/query/notification.go`: 定义 list、unread count、summary 查询 DTO 及兼容 result code。
- `internal/application/port/notification.go`: 定义 actor verifier、unread cache、realtime publisher、user directory、email renderer/sender、clock 外部端口。
- `internal/application/service/notification_service.go`: 迁移 list、unread、summary、mark read/all read、create 和 notification consumer 编排，保留旧实现软失败与幂等语义。
- `internal/application/service/email_service.go`: 迁移 email coordination，用 port 替代直接 `users`/SMTP/template 依赖，保留 missing template、duplicate、missing email、send failure 的 skip/retry 语义。
- `internal/application/service/inbox.go`: 本地化 Inbox consumer 幂等 identity 推导、claim、processed/failed 标记流程。
- `internal/application/service/time.go`: 统一 protobuf-compatible 时间格式化辅助。
- `internal/application/service/notification_service_test.go`: 覆盖 list/unread/summary/mark read/email coordination/notification idempotency/inbox idempotency。
- `.spec/.../TASK-010-report.md`: 新增 TASK 报告。
- `.spec/.../TASK-010-evidence.json`: 新增机器可读证据。

## 4. Scope Check Result

`bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-010` 通过。

变更均在 TASK-010 allowed files 内：`smart-recruit-notification-service/**` 与 `.spec/microservice-ddd-evolution/reports/**`。

未修改 forbidden files：`smart-recruit-commons/**`、`smart-recruit-proto/**`、`db.sql`、migration、deployment、go workspace、package manifest 或 lockfile。

## 5. SPEC Comparison Result

符合 SPEC：

- FR-002：domain 层只使用 Go 标准库，不依赖 GORM、RabbitMQ、Redis、gRPC、HTTP、Nacos、SMTP、proto 或 shared module。
- FR-003：application 层承载 notification list/unread/summary/mark read/email/consumer/inbox 用例编排，并通过 port 表达外部依赖。
- FR-010：迁出 unread count、summary、mark read、email coordination、SSE/realtime payload 和 Inbox 消费幂等相关 application 语义。
- FR-017：仓储接口定义在本服务 `domain/repository`，具体 GORM adapter 留给 TASK-011。
- CR-001/CR-002/CR-006：未修改 HTTP API、protobuf、Gateway route mode、RabbitMQ queue、Redis channel、Outbox/Inbox schema 或邮件模板。

## 6. SDD Comparison Result

符合 SDD：

- 6. Algorithm or Workflow Changes：本地 application service 保留 page/cursor list、unread cache fallback、summary fallback、mark read/all read 批处理、notification created realtime、email skip/send failure、Inbox claim/process/fail 算法。
- 9. Error Handling and Fallback Design：保留旧语义中 DB list/unread/summary/mark read 的软降级；email send failure 返回错误以进入 retry/DLQ，duplicate/missing template/missing email 跳过不重试。
- 13. Implementation Boundaries：只新增 Notification 服务本地 domain/application 代码与测试，不切换 runtime，不改 shared/proto/schema。

## 7. Acceptance Comparison Result

- Notification domain/application 本地化：已新增本地 domain model、repository port、application command/query/port/service。
- unread/summary/mark read/email coordination/idempotency 语义兼容：已在 service 与 tests 覆盖。
- Domain 不依赖 GORM/RabbitMQ/gRPC：`rg` 导入检查未发现 domain/application 使用这些外层技术；唯一匹配是既有 doc 注释说明独立于外层技术。
- Runtime 切换、真实 SMTP、消息 schema 修改均未执行，符合 Out-of-Scope。

## 8. Test Commands and Results

| Command | Exit | Result | Summary |
|---|---:|---|---|
| `gofmt -w ... && go test ./...` in `smart-recruit-notification-service` | 0 | passed | Notification service 全 package 测试通过；新增 domain/application tests 通过。 |
| `git diff --name-only` | 0 | passed | 输出为空，因为当前 TASK 文件均为未跟踪新增文件；实际变更由 `git status --short` 与 evidence `changed_files` 记录。 |
| `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-010` | 0 | passed | `Scope check passed for TASK-010. Changed files: 14`。 |
| `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh` | 0 | passed | Harness JSON validation passed；在 `smart-recruit-notification-service` 内运行 `go test ./...` 并通过。 |
| `node scripts/check-mysql-table-ownership.mjs` | 0 | passed | `mysql_table_ownership: PASS (67 tables, single MySQL instance)` |
| `node scripts/check-backend-boundaries.mjs` | 0 | passed | `backend_boundary_result: PASS` |

额外知识影响检测：

- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 344d874243c5130e448c9c58356679bd3205a883` exit 1。
- 失败原因是既有 active knowledge 中大量旧 `logic-grpc-service` / `web-gin-service` `source_refs` 缺失；本 TASK scope 不允许修改 `.knowledge/**`，因此记录为非阻塞 candidate debt。

## 9. Knowledge Impact

```yaml
knowledge_impact:
  result: candidate_required
  triggered_by:
    - smart-recruit-notification-service/internal/domain/**
    - smart-recruit-notification-service/internal/application/**
  reviewed_documents:
    - .knowledge/architecture/service-boundaries.md
    - .knowledge/domains/notification-outbox.md
  update_paths:
    - .knowledge/inbox/microservice-ddd-evolution-boundaries.md
  coverage_gap: false
  reason: TASK-010 本地化 Notification domain/application，但 active knowledge 仍描述旧 logic-grpc-service notification/outbox 拓扑；当前 TASK scope 不允许修改 .knowledge/**。
```

## 10. Self-review and Repair

独立只读 self-review 第 1 轮 verdict: 通过。

Reviewer 核对结果：

- 未发现 Critical/High/Medium/Low 问题。
- 变更均在 `smart-recruit-notification-service/**` 与报告目录内。
- 未修改 shared/proto/schema/package/lockfile/global config。
- domain/application 未引入 GORM、RabbitMQ、gRPC、proto 依赖。
- unread、summary、mark read/all read、email coordination、notification idempotency、Inbox identity/claim/processed/failed 语义与旧实现基本兼容。
- `gofmt -l`、scope check、`go test ./...`、agent-check、backend boundary/table ownership 检查均通过。

## 11. Risks

- Active runtime 仍未切换到本地 application service；这是 TASK-010 Out-of-Scope，TASK-011 负责 adapter/interface/runtime 收敛。
- Email recipient lookup 已被抽为 `UserDirectory` port，但实际 `users` 读取 adapter 尚未迁移；TASK-011 需要实现 adapter 或继续记录 transitional read debt。
- 本 TASK 没有修改 Outbox publisher；只本地化 notification/email consumer 用例语义。
- Active knowledge 仍有旧路径 source_refs 债务；本 TASK 已记录 candidate_required，但不越界修改知识库。

## 12. Follow-up Items

- TASK-011 实现 persistence/cache/mq/email/client adapters、gRPC mapper/server 和 runtime wiring。
- TASK-011 需要证明 no-real-SMTP 测试路径，保留 retry/DLQ/Inbox/realtime 兼容。

## 13. Whether the Next TASK Can Start

独立 self-review 通过且 evidence validator 通过后，可以开始 TASK-011。
