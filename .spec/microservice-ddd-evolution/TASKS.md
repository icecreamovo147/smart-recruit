# TASKS - microservice-ddd-evolution

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | 建立迁移总则与边界检查基线 | pending | docs, scripts, knowledge candidates | acceptance/TASK-001.md |
| TASK-002 | 盘点共享依赖、表归属与服务迁移基线 | pending | docs, reports, optional knowledge | acceptance/TASK-002.md |
| TASK-003 | Offer DDD 骨架与契约盘点 | pending | offer service only | acceptance/TASK-003.md |
| TASK-004 | Offer domain/application 迁移 | pending | offer service only | acceptance/TASK-004.md |
| TASK-005 | Offer infrastructure/interfaces/runtime/tests 收敛 | pending | offer service only | acceptance/TASK-005.md |
| TASK-006 | Interview DDD 骨架与契约盘点 | pending | interview service only | acceptance/TASK-006.md |
| TASK-007 | Interview domain/application 迁移 | pending | interview service only | acceptance/TASK-007.md |
| TASK-008 | Interview infrastructure/interfaces/runtime/tests 收敛 | pending | interview service only | acceptance/TASK-008.md |
| TASK-009 | Notification DDD 骨架与契约盘点 | pending | notification service only | acceptance/TASK-009.md |
| TASK-010 | Notification domain/application 迁移 | pending | notification service only | acceptance/TASK-010.md |
| TASK-011 | Notification infrastructure/interfaces/runtime/tests 收敛 | pending | notification service only | acceptance/TASK-011.md |
| TASK-012 | Identity DDD 骨架与安全契约盘点 | pending | identity service only | acceptance/TASK-012.md |
| TASK-013 | Identity domain/application 迁移 | pending | identity service only | acceptance/TASK-013.md |
| TASK-014 | Identity infrastructure/interfaces/runtime/tests 收敛 | pending | identity service only | acceptance/TASK-014.md |
| TASK-015 | Recruitment DDD 骨架与核心域盘点 | pending | recruitment service only | acceptance/TASK-015.md |
| TASK-016 | Recruitment job/candidate/resume domain/application 迁移 | pending | recruitment service only | acceptance/TASK-016.md |
| TASK-017 | Recruitment application/collaboration/taxonomy 迁移 | pending | recruitment service only | acceptance/TASK-017.md |
| TASK-018 | Recruitment infrastructure/interfaces/runtime/tests 收敛 | pending | recruitment service only | acceptance/TASK-018.md |
| TASK-019 | Analytics DDD 骨架与投影策略盘点 | pending | analytics service only | acceptance/TASK-019.md |
| TASK-020 | Analytics reporting/projection application 迁移 | pending | analytics service only | acceptance/TASK-020.md |
| TASK-021 | Analytics infrastructure/interfaces/runtime/tests 收敛 | pending | analytics service only | acceptance/TASK-021.md |
| TASK-022 | AI Agent DDD 骨架与复杂依赖盘点 | pending | ai-agent service only | acceptance/TASK-022.md |
| TASK-023 | AI Agent chat/agent-run/prompt domain/application 迁移 | pending | ai-agent service only | acceptance/TASK-023.md |
| TASK-024 | AI Agent MCP/skill/embedding/intelligence 迁移 | pending | ai-agent service only | acceptance/TASK-024.md |
| TASK-025 | AI Agent infrastructure/interfaces/runtime/tests 收敛 | pending | ai-agent service only | acceptance/TASK-025.md |
| TASK-026 | Worker DDD/workload 骨架与边界盘点 | pending | worker service only | acceptance/TASK-026.md |
| TASK-027 | Worker workload application/runtime 迁移 | pending | worker service only | acceptance/TASK-027.md |
| TASK-028 | Worker infrastructure/tests 与 owner contract 收敛 | pending | worker service only | acceptance/TASK-028.md |
| TASK-029 | 收缩 smart-recruit-commons 至 commons-ready shared kernel | pending | domain-go, docs, scripts | acceptance/TASK-029.md |
| TASK-030 | 最终重命名 smart-recruit-commons 为 smart-recruit-commons | pending | cross-repo rename | acceptance/TASK-030.md |

## TASK-001 - 建立迁移总则与边界检查基线

### Goal

建立本功能点可执行的架构迁移规则、DDD 分层规范、Hard Stop 清单和初始边界检查能力，作为后续所有服务迁移的先决条件。

### Scope

仅允许创建或修改架构文档、边界检查脚本和知识候选记录，不迁移业务代码。

### Allowed Files

- `docs/architecture/**`
- `scripts/check-backend-boundaries.mjs`
- `scripts/backend-final-readiness-audit.mjs`
- `.knowledge/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- `smart-recruit-*-service/**`
- `smart-recruit-commons/**`
- `smart-recruit-platform-go/**`
- `smart-recruit-proto/**`
- `go.mod`, `go.sum`, `go.work`, `go.work.sum`
- `package.json`, `pnpm-lock.yaml`
- `db.sql`, `smart-recruit-commons/migrations/**`

### Dependencies

无。

### Acceptance Criteria

- DDD 分层规范、服务迁移顺序、shared kernel 规则和 Hard Stop 条件被文档化。
- 边界检查至少定义 domain 不得依赖 GORM/gRPC/Redis/RabbitMQ/HTTP/Nacos 等外层技术的规则。
- 不修改业务代码。

### Required Tests

- `node scripts/check-backend-boundaries.mjs`，若脚本不可用则记录原因。
- `bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh TASK-001`
- `bash .spec/microservice-ddd-evolution/scripts/agent-check.sh`

### Risks

边界检查过严可能先暴露大量历史债务。TASK 只能建立规则与报告，不应强行修复所有债务。

### Notes

若需要修改 CI/CD 或 package/lockfile，停止并请求确认。

## TASK-002 - 盘点共享依赖、表归属与服务迁移基线

### Goal

建立当前 `smart-recruit-commons` 业务依赖、服务 import、表访问和 owner 边界基线，为后续每个服务迁移提供可比较证据。

### Scope

仅允许文档、报告、知识候选和只读盘点脚本输出。

### Allowed Files

- `docs/architecture/**`
- `.knowledge/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- 任何业务代码、protobuf、schema、migration、package manifest、lockfile。

### Dependencies

TASK-001。

### Acceptance Criteria

- 记录每个服务当前对 `smart-recruit-commons/model`、`repository`、`service`、`ai`、`oss`、`email`、`mq` 的依赖。
- 记录每个服务当前 owner 表、transitional read 表和潜在违规写风险。
- 明确各服务进入迁移前的剩余风险。

### Required Tests

- `node scripts/check-mysql-table-ownership.mjs`
- `go test ./...` 针对未修改业务代码可说明未运行，但 agent-check 必须运行。

### Risks

盘点可能发现现有 manifest 或脚本不完整；只记录，不在本 TASK 修复业务边界。

### Notes

本 TASK 的输出将作为后续 service-specific TASK 的 baseline。

## TASK-003 - Offer DDD 骨架与契约盘点

### Goal

在 `smart-recruit-offer-service` 内创建目标 DDD 目录骨架，盘点当前 Offer protobuf、runtime、repository、表访问和测试覆盖。

### Scope

仅限 `smart-recruit-offer-service` 和本功能报告。

### Allowed Files

- `smart-recruit-offer-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- `smart-recruit-commons/**`
- `smart-recruit-proto/**`
- schema/migration/package/lockfile/global config。

### Dependencies

TASK-002。

### Acceptance Criteria

- Offer 服务本地 DDD 骨架存在。
- Offer gRPC contract、当前 `service.OfferService` 依赖和表访问被记录。
- 未改变 protobuf 或用户可见行为。

### Required Tests

- `go test ./...` from `smart-recruit-offer-service`
- scope check and agent-check

### Risks

骨架创建不应引入未使用代码导致编译失败。

### Notes

若必须修改 shared module，停止并请求确认。

## TASK-004 - Offer domain/application 迁移

### Goal

迁移 Offer 生命周期规则、领域事件、repository port 和 application command/query 编排到 `smart-recruit-offer-service/internal`。

### Scope

仅限 Offer 服务本地 domain/application 及测试。

### Allowed Files

- `smart-recruit-offer-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-003。

### Acceptance Criteria

- Offer domain model 不依赖 GORM/proto/gRPC。
- Application 层定义事务和跨上下文 port，不直接依赖 GORM。
- 覆盖 create/update/send/withdraw/accept/reject/list event 关键规则。

### Required Tests

- Offer domain/application unit tests
- `go test ./...` from `smart-recruit-offer-service`

### Risks

Offer 当前可能依赖 Application/Profile/Job 读取；新增跨服务 API 需要 Hard Stop。

### Notes

可保留临时 adapter，但必须记录移除条件。

## TASK-005 - Offer infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 Offer GORM persistence、outbox/client adapter、gRPC interface、runtime 装配和测试迁移，使 Offer 不再依赖共享 `service.OfferService`。

### Scope

仅限 Offer 服务本地代码和报告。

### Allowed Files

- `smart-recruit-offer-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- `smart-recruit-commons/**` 删除或重构、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-004。

### Acceptance Criteria

- Offer runtime 使用本地 application/interfaces 实现注册 `OfferService`。
- 现有 gRPC 语义兼容。
- Offer 服务对共享 `service.OfferService` 的直接依赖消除或记录为临时债务。

### Required Tests

- `go test ./...` from `smart-recruit-offer-service`
- table ownership check

### Risks

Runtime 切换可能遗漏健康检查、metrics、trace 或 auth metadata 传递。

### Notes

不得删除 shared 旧实现，除非后续 cleanup TASK 授权。

## TASK-006 - Interview DDD 骨架与契约盘点

### Goal

在 `smart-recruit-interview-service` 内创建 DDD 骨架并盘点 Interview gRPC、runtime、表访问、事件和测试覆盖。

### Scope

仅限 Interview 服务。

### Allowed Files

- `smart-recruit-interview-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-005。

### Acceptance Criteria

- Interview DDD 骨架存在。
- schedule/update/cancel/batch cancel/feedback/listing 依赖盘点完成。
- 不改变 protobuf 或用户行为。

### Required Tests

- `go test ./...` from `smart-recruit-interview-service`

### Risks

Interview 与 Recruitment/Application/Identity 的只读依赖需要明确过渡策略。

### Notes

保持 Offer 迁移成果不回退。

## TASK-007 - Interview domain/application 迁移

### Goal

迁移 Interview 状态规则、feedback 规则、repository port 和 application 编排。

### Scope

仅限 Interview 服务本地 domain/application。

### Allowed Files

- `smart-recruit-interview-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-006。

### Acceptance Criteria

- Interview domain 不依赖外层技术。
- Application 层保持当前权限、状态、通知/outbox 语义。
- 覆盖 schedule/update/cancel/batch cancel/feedback 状态测试。

### Required Tests

- Interview domain/application tests
- `go test ./...` from `smart-recruit-interview-service`

### Risks

取消面试和通知事件的一致性不能退化。

### Notes

新增跨服务 API 或事件 schema 需要确认。

## TASK-008 - Interview infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 Interview persistence、adapter、gRPC interface、runtime 装配和测试迁移。

### Scope

仅限 Interview 服务。

### Allowed Files

- `smart-recruit-interview-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-007。

### Acceptance Criteria

- Interview runtime 使用本地 implementation 注册 `InterviewService`。
- 对共享 `service.InterviewService` 的直接依赖消除或记录为临时债务。
- gRPC 语义兼容。

### Required Tests

- `go test ./...` from `smart-recruit-interview-service`
- table ownership check

### Risks

列表查询和权限范围可能出现细微行为漂移。

### Notes

不得修改 Recruitment/Offer 行为。

## TASK-009 - Notification DDD 骨架与契约盘点

### Goal

在 `smart-recruit-notification-service` 内创建 DDD 骨架并盘点通知、邮件、Outbox/Inbox、SSE/realtime 相关契约。

### Scope

仅限 Notification 服务。

### Allowed Files

- `smart-recruit-notification-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-008。

### Acceptance Criteria

- Notification DDD 骨架存在。
- Notification runtime dependencies、queue、table、consumer 幂等点被记录。
- 不改变通知 API 行为。

### Required Tests

- `go test ./...` from `smart-recruit-notification-service`

### Risks

通知是异步一致性关键路径，需保留 retry/DLQ 语义。

### Notes

不得扩大邮件发送真实外部调用。

## TASK-010 - Notification domain/application 迁移

### Goal

迁移通知记录、未读数、summary、mark read、email coordination 和消费用例到本地 domain/application。

### Scope

仅限 Notification 服务本地 domain/application。

### Allowed Files

- `smart-recruit-notification-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-009。

### Acceptance Criteria

- Notification domain 不依赖 GORM/RabbitMQ/gRPC。
- Application 层保留幂等、权限和邮件协调语义。
- 覆盖 unread/summary/mark read/all read/idempotency 测试。

### Required Tests

- Notification domain/application tests
- `go test ./...` from `smart-recruit-notification-service`

### Risks

Unread count 和 summary 容易与旧实现产生边界差异。

### Notes

外部 SMTP 调用必须 fake 或 env-gated。

## TASK-011 - Notification infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 Notification GORM/MQ/email/cache adapter、gRPC interface、runtime 装配和测试。

### Scope

仅限 Notification 服务。

### Allowed Files

- `smart-recruit-notification-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-010。

### Acceptance Criteria

- Notification runtime 使用本地 application/interfaces 实现。
- Outbox/Inbox 幂等和 retry 语义兼容。
- 对共享 notification service/runtime 的直接依赖消除或记录为临时债务。

### Required Tests

- `go test ./...` from `smart-recruit-notification-service`
- table ownership check

### Risks

Consumer lifecycle 与 health/readiness 可能被破坏。

### Notes

不得修改消息 schema，除非停止确认。

## TASK-012 - Identity DDD 骨架与安全契约盘点

### Goal

在 `smart-recruit-identity-service` 内创建 DDD 骨架并盘点 auth、refresh token、principal、RBAC、data scope、invite code、audit 契约。

### Scope

仅限 Identity 服务。

### Allowed Files

- `smart-recruit-identity-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-011。

### Acceptance Criteria

- Identity DDD 骨架存在。
- 安全相关 API、权限语义、表访问和测试覆盖盘点完成。
- 未改变 auth/authz 行为。

### Required Tests

- `go test ./...` from `smart-recruit-identity-service`

### Risks

Identity 涉及安全语义，后续 TASK 默认高审慎。

### Notes

任何 auth/authz 行为变化必须停止确认。

## TASK-013 - Identity domain/application 迁移

### Goal

迁移用户、认证、Refresh Token、RBAC、data scope、invite code、audit 的 domain/application 能力。

### Scope

仅限 Identity 服务本地 domain/application。

### Allowed Files

- `smart-recruit-identity-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-012。

### Acceptance Criteria

- Identity domain 不依赖外层技术。
- Application 层保留 JWT/refresh/RBAC/data scope/audit 语义。
- 覆盖 register/login/refresh/revoke/principal/RBAC 测试。

### Required Tests

- Identity domain/application tests
- `go test ./...` from `smart-recruit-identity-service`

### Risks

权限范围或 token invalidation 行为漂移属于高风险。

### Notes

不得改变 secret validation 或 internal auth policy。

## TASK-014 - Identity infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 Identity persistence/cache/audit adapter、gRPC interface、runtime 装配和测试迁移。

### Scope

仅限 Identity 服务。

### Allowed Files

- `smart-recruit-identity-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-013。

### Acceptance Criteria

- Identity runtime 使用本地 implementation 注册 Auth/Admin 能力。
- 安全语义和 audit 行为兼容。
- 对共享 auth/admin service 的直接依赖消除或记录为临时债务。

### Required Tests

- `go test ./...` from `smart-recruit-identity-service`
- table ownership check

### Risks

AdminService 当前跨多个上下文暴露能力，需保持路由和注册兼容。

### Notes

公共权限/角色变更必须停止确认。

## TASK-015 - Recruitment DDD 骨架与核心域盘点

### Goal

在 `smart-recruit-recruitment-service` 内创建 DDD 骨架并盘点 job、candidate、resume、application、collaboration、taxonomy、usage stats 契约。

### Scope

仅限 Recruitment 服务。

### Allowed Files

- `smart-recruit-recruitment-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-014。

### Acceptance Criteria

- Recruitment DDD 骨架存在。
- 核心域表、服务接口、状态机、跨上下文读依赖盘点完成。
- 不改变招聘 API 行为。

### Required Tests

- `go test ./...` from `smart-recruit-recruitment-service`

### Risks

Recruitment 是核心最大上下文，TASK 必须避免超范围重构。

### Notes

保留已完成 Offer/Interview/Notification/Identity 边界。

## TASK-016 - Recruitment job/candidate/resume domain/application 迁移

### Goal

迁移 Job、Candidate Profile、Resume、Resume Profile 相关 domain/application 能力。

### Scope

仅限 Recruitment 服务本地 job/candidate/resume 相关代码。

### Allowed Files

- `smart-recruit-recruitment-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-015。

### Acceptance Criteria

- Job/Candidate/Resume domain model 与 repository port 本地化。
- Application 层保留 OSS presign、resume confirm、usage log、权限语义。
- 覆盖 job lifecycle、candidate profile、resume upload confirm 测试。

### Required Tests

- Recruitment targeted tests
- `go test ./...` from `smart-recruit-recruitment-service`

### Risks

OSS 与简历解析 outbox 语义不能退化。

### Notes

AI Agent 对 resume/profile 的读依赖暂作为受控过渡。

## TASK-017 - Recruitment application/collaboration/taxonomy 迁移

### Goal

迁移 Application lifecycle、Collaboration、Taxonomy、UsageStats 相关 domain/application 能力。

### Scope

仅限 Recruitment 服务。

### Allowed Files

- `smart-recruit-recruitment-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-016。

### Acceptance Criteria

- Application 状态机和 collaboration/taxonomy 规则本地化。
- 保留 application status transitions、outbox、权限和 pagination 语义。
- 覆盖 apply/update status/collaboration/taxonomy 测试。

### Required Tests

- Recruitment targeted tests
- `go test ./...` from `smart-recruit-recruitment-service`

### Risks

Application 是 Offer/Interview/AI/Analytics 的关键关联点。

### Notes

新增 snapshot API 或事件 schema 需要确认。

## TASK-018 - Recruitment infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 Recruitment persistence/OSS/outbox/cache adapter、gRPC interface、runtime 装配和测试迁移。

### Scope

仅限 Recruitment 服务。

### Allowed Files

- `smart-recruit-recruitment-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-017。

### Acceptance Criteria

- Recruitment runtime 使用本地 implementation 注册 Job/Candidate/Application/Collaboration/Admin 子集。
- 对共享 recruitment 业务 service 的直接依赖消除或记录为临时债务。
- 所有核心招聘 API 语义兼容。

### Required Tests

- `go test ./...` from `smart-recruit-recruitment-service`
- table ownership check

### Risks

AdminService 在 Identity/Recruitment/Analytics 间切分复杂。

### Notes

不得改变 protobuf AdminService 契约。

## TASK-019 - Analytics DDD 骨架与投影策略盘点

### Goal

在 `smart-recruit-analytics-service` 内创建 DDD 骨架并盘点 reporting API、projection table、只读跨表依赖和事件投影目标。

### Scope

仅限 Analytics 服务。

### Allowed Files

- `smart-recruit-analytics-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-018。

### Acceptance Criteria

- Analytics DDD 骨架存在。
- 现有 reporting/read model 依赖和过渡只读跨表债务被记录。
- 不改变报表 API 行为。

### Required Tests

- `go test ./...` from `smart-recruit-analytics-service`

### Risks

事件投影终态可能需要 schema，当前只盘点和本地化。

### Notes

Schema/proto 变化必须独立确认。

## TASK-020 - Analytics reporting/projection application 迁移

### Goal

迁移 Analytics reporting query、projection application 和只读 owner guard 到服务本地。

### Scope

仅限 Analytics 服务本地 domain/application。

### Allowed Files

- `smart-recruit-analytics-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-019。

### Acceptance Criteria

- Analytics 不写回事务业务状态。
- Query/application 层明确事件投影终态和过渡只读债务。
- 覆盖 dashboard/funnel/time-in-stage/interview-offer metrics 测试。

### Required Tests

- Analytics tests
- `go test ./...` from `smart-recruit-analytics-service`

### Risks

报表口径不能因迁移改变。

### Notes

只读跨表长期例外不允许静默固化。

## TASK-021 - Analytics infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 Analytics persistence/projection adapter、gRPC interface、runtime 装配和测试迁移。

### Scope

仅限 Analytics 服务。

### Allowed Files

- `smart-recruit-analytics-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-020。

### Acceptance Criteria

- Analytics runtime 使用本地 implementation。
- Reporting API 兼容。
- 对共享 analytics service/repository 的直接依赖消除或记录为临时债务。

### Required Tests

- `go test ./...` from `smart-recruit-analytics-service`
- table ownership check

### Risks

Analytics AdminService 子集注册不能与 Identity/Recruitment 冲突。

### Notes

保持 projection 终态债务可追踪。

## TASK-022 - AI Agent DDD 骨架与复杂依赖盘点

### Goal

在 `smart-recruit-ai-agent-service` 内创建 DDD 骨架并盘点 AI chat、agent run、prompt、agent config、MCP、skill、embedding、recruiting intelligence 依赖。

### Scope

仅限 AI Agent 服务。

### Allowed Files

- `smart-recruit-ai-agent-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-021。

### Acceptance Criteria

- AI Agent DDD 骨架存在。
- 外部 provider、MCP policy、encryption key、long task、streaming、audit 依赖盘点完成。
- 不改变 AI API 行为。

### Required Tests

- `go test ./...` from `smart-recruit-ai-agent-service`

### Risks

AI Agent 依赖最复杂，必须分阶段迁移。

### Notes

真实 provider 调用必须 fake/env-gated。

## TASK-023 - AI Agent chat/agent-run/prompt domain/application 迁移

### Goal

迁移 AI chat/session、agent run、prompt/model config 相关 domain/application 能力。

### Scope

仅限 AI Agent 服务相关代码。

### Allowed Files

- `smart-recruit-ai-agent-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-022。

### Acceptance Criteria

- Chat/session/agent run/prompt application 本地化。
- 保留 stream、cancel、confirm、audit、provider fallback 语义。
- 覆盖 agent run state、durability、prompt config 测试。

### Required Tests

- AI Agent targeted tests
- `go test ./...` from `smart-recruit-ai-agent-service`

### Risks

流式响应和持久 agent run 不能丢事件或乱序。

### Notes

新增 AI provider 行为不在范围内。

## TASK-024 - AI Agent MCP/skill/embedding/intelligence 迁移

### Goal

迁移 MCP、Skill、AgentSkill、Embedding、RecruitingIntelligence、CandidateMatch 相关 domain/application 能力。

### Scope

仅限 AI Agent 服务。

### Allowed Files

- `smart-recruit-ai-agent-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-023。

### Acceptance Criteria

- MCP/Skill/Embedding/intelligence use cases 本地化。
- 保留私网限制、allowlist、credential encryption、fallback、semantic retrieval 语义。
- 覆盖 MCP policy、skill version、embedding config、candidate match 测试。

### Required Tests

- AI Agent targeted tests
- `go test ./...` from `smart-recruit-ai-agent-service`

### Risks

安全策略和加密凭据处理不能退化。

### Notes

Schema/proto 变化必须停止确认。

## TASK-025 - AI Agent infrastructure/interfaces/runtime/tests 收敛

### Goal

完成 AI Agent provider/MQ/persistence/MCP/embedding adapter、gRPC interface、runtime 装配和测试迁移。

### Scope

仅限 AI Agent 服务。

### Allowed Files

- `smart-recruit-ai-agent-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-024。

### Acceptance Criteria

- AI Agent runtime 使用本地 implementation 注册全部 AI 相关 gRPC services。
- Embedding/agent-run worker control 兼容。
- 对共享 AI service/runtime 的直接依赖消除或记录为临时债务。

### Required Tests

- `go test ./...` from `smart-recruit-ai-agent-service`
- table ownership check

### Risks

长任务 worker 和 request-serving runtime 依赖边界复杂。

### Notes

不得泄露 provider credentials。

## TASK-026 - Worker DDD/workload 骨架与边界盘点

### Goal

在 `smart-recruit-worker-service` 内创建 workload 分层骨架并盘点所有后台 workload 与 owner context contract。

### Scope

仅限 Worker 服务。

### Allowed Files

- `smart-recruit-worker-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-025。

### Acceptance Criteria

- Worker workload profile/toggle 骨架存在。
- Outbox、Inbox、DLQ、notification、email、resume parsing、embedding、agent run、analytics projection owner contract 被记录。
- 不改变 workload 默认启停语义。

### Required Tests

- `go test ./...` from `smart-recruit-worker-service`

### Risks

Worker 容易绕过 owner service 直接写业务表。

### Notes

本 TASK 不拆 worker binary。

## TASK-027 - Worker workload application/runtime 迁移

### Goal

按 owner context contract 迁移 Worker workload application/runtime 编排。

### Scope

仅限 Worker 服务。

### Allowed Files

- `smart-recruit-worker-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-026。

### Acceptance Criteria

- Worker workload 通过 profile/toggle 分类运行。
- 后台写入遵守 owner contract，不新增绕过 owner domain 的写路径。
- 覆盖 workload parse/toggle/graceful shutdown 测试。

### Required Tests

- Worker targeted tests
- `go test ./...` from `smart-recruit-worker-service`

### Risks

Consumer lifecycle、retry、DLQ、shutdown 时序容易出错。

### Notes

新增 workload 不是本 TASK 目标。

## TASK-028 - Worker infrastructure/tests 与 owner contract 收敛

### Goal

完成 Worker MQ/persistence/client adapter、health/readiness、测试和共享依赖收敛。

### Scope

仅限 Worker 服务。

### Allowed Files

- `smart-recruit-worker-service/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- shared module 删除、protobuf、schema、deployment、package/lockfile。

### Dependencies

TASK-027。

### Acceptance Criteria

- Worker 不再依赖共享业务 service 编排，或剩余依赖记录为 shared cleanup 债务。
- Workload readiness 和 idempotency 测试覆盖。
- 不改变默认 workload 行为。

### Required Tests

- `go test ./...` from `smart-recruit-worker-service`
- table ownership check

### Risks

Worker 是最终服务迁移前最后一道跨上下文写入收敛点。

### Notes

完成后才能进入 shared cleanup。

## TASK-029 - 收缩 smart-recruit-commons 至 commons-ready shared kernel

### Goal

清理或迁出 `smart-recruit-commons` 中残留的具体业务 `model/repository/service`，使其达到可安全重命名为 commons 的状态。

### Scope

允许修改 `smart-recruit-commons`、相关 docs/scripts/knowledge 和各服务中残留 import 的最小修正。

### Allowed Files

- `smart-recruit-commons/**`
- `smart-recruit-*-service/**`
- `smart-recruit-gateway/**`
- `docs/architecture/**`
- `scripts/**`
- `.knowledge/**`
- `.spec/microservice-ddd-evolution/reports/**`

### Forbidden Files

- `smart-recruit-proto/**`
- schema/migration 内容变更，除非仅迁移通用 migration runner 且不改变 SQL。
- package/lockfile，除非用户确认。

### Dependencies

TASK-028。

### Acceptance Criteria

- `smart-recruit-commons` 剩余内容分类为 shared kernel、platform-adjacent、testutil、migration helper 或明确删除。
- 不再包含具体业务上下文的 active `model/repository/service` 实现。
- 所有服务对旧业务共享实现的依赖清零或转为允许保留的 shared kernel 依赖。

### Required Tests

- All Go modules impacted by import changes
- `node scripts/check-mysql-table-ownership.mjs`
- backend boundary checks

### Risks

这是大范围共享模块清理任务，容易触发 import path 和隐式依赖问题。

### Notes

若需要修改 module path，留到 TASK-030。

## TASK-030 - 最终重命名 smart-recruit-commons 为 smart-recruit-commons

### Goal

执行最终重命名：`smart-recruit-commons -> smart-recruit-commons`，同步 Go module path、import path、workspace、构建、部署、文档、脚本和边界检查。

### Scope

跨仓库 rename 和引用更新。

### Allowed Files

- `smart-recruit-commons/**`
- `smart-recruit-commons/**`
- `smart-recruit-*-service/**`
- `smart-recruit-gateway/**`
- `smart-recruit-platform-go/**`
- `smart-recruit-proto/**`
- `go.work`
- `go.work.sum`
- `smart-recruit-deploy/**`
- `deploy/**`
- `docker/**`
- `start-dev.sh`
- `start-docker-all.sh`
- `stop-dev.sh`
- `README.md`
- `docs/**`
- `scripts/**`
- `.knowledge/**`
- `.spec/microservice-ddd-evolution/**`

### Forbidden Files

- `package.json`, `pnpm-lock.yaml`，除非用户确认。
- protobuf behavior/schema behavior/auth behavior changes.
- 真实 secret/config files.

### Dependencies

TASK-029。

### Acceptance Criteria

- 旧目录与旧 Go module/import path 不再被业务代码、测试、脚本、部署或文档引用。
- 新 `smart-recruit-commons` 只包含 shared kernel 和通用技术能力。
- 全仓库 Go 测试、边界检查、表归属检查、构建脚本引用检查通过或有明确不可运行原因。

### Required Tests

- `go test ./...` in all Go modules or workspace-equivalent checks.
- `node scripts/check-mysql-table-ownership.mjs`
- backend boundary checks.
- `rg "smart-recruit-commons"` should only show approved historical references, if any.

### Risks

这是最终高风险横切 TASK，必须人工确认后执行，并要求全仓库验证。

### Notes

此 TASK 是本功能点最后一个迁移任务。
