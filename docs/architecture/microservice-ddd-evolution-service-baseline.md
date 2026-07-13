# Microservice DDD Evolution Service Baseline

## 1. 目的

本文是 `.spec/microservice-ddd-evolution` 的 TASK-002 基线盘点，记录当前微服务根对 `smart-recruit-domain-go` 的依赖、表归属、过渡只读关系和潜在违规写风险。

本文只记录当前事实和迁移风险，不修改业务代码、表归属、schema、protobuf、部署配置或 package/lockfile。

## 2. 盘点依据

- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md`
- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md`
- `smart-recruit-deploy/mysql-table-ownership.json`
- `db.sql`
- `smart-recruit-*-service/go.mod`
- `smart-recruit-*-service/cmd/*/main.go`
- `smart-recruit-*-service/internal/runtime/*.go`
- `scripts/check-mysql-table-ownership.mjs`

校验结果：

```text
mysql_table_ownership: PASS (67 tables, single MySQL instance)
```

## 3. 全局基线结论

- 8 个业务/后台服务根均仍直接依赖 `smart-recruit-domain-go` module。
- 8 个服务根均保留 `smart-recruit-domain-go/config/config.yaml` 或 `../smart-recruit-domain-go/config/config.yaml` 的本地配置 fallback 路径。
- Offer、Interview、AI Agent 仍通过 `service.NewServices` 构造跨上下文聚合服务，迁移时风险最高。
- Recruitment 已局部手动装配具体 shared service，但仍直接构造跨上下文 repository。
- Identity、Analytics、Notification 已按服务职责更窄地装配 shared service，但仍依赖 shared repository/service。
- Worker 当前主要依赖 `smart-recruit-domain-go/mq` 和 shared 配置，尚未拥有 owner 表。
- `smart-recruit-deploy/mysql-table-ownership.json` 当前声明 67 张表，单 MySQL 过渡模式有效。
- 当前所有非 owner write 都集中在平台型共享表：`event_outbox`、`event_inbox`、`third_party_usage_logs`。这些应视为授权 shared kernel/platform-adjacent 债务，不等同于业务 owner 表写入许可。

## 4. 服务依赖与表边界

### 4.1 Offer

服务根：`smart-recruit-offer-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/offer-service/main.go` 构造 `repository.NewUserRepo`、`RefreshTokenRepo`、`JobRepo`、`ProfileRepo`、`ResumeRepo`、`ApplicationRepo`、`InterviewRepo`、`OfferRepo`、`NotificationRepo`、`OutboxRepo`、`AuthzRepo`、AI/chat/memory/usage/email 等 repository。
- 通过 `service.NewServices` 获取共享 `OfferService` 所在的大聚合。
- 本地 `internal/runtime` 仍是 gRPC 注册 facade，没有本地 `domain/application/infrastructure/interfaces` 业务实现。

表边界：

- Owner 表：`offers`、`offer_events`
- 过渡只读表：`applications`（owner: recruitment）
- 授权 shared write：`event_outbox`（owner: platform）

潜在违规写风险：

- 当前装配了多个非 Offer owner repository。后续 TASK-003 至 TASK-005 必须证明 Offer runtime 只通过本地 application port 和 owner contract 使用非 owner 数据。
- Offer lifecycle 若继续直接更新 `applications` 或非 owner 表，将违反 FR-018 至 FR-020。

### 4.2 Interview

服务根：`smart-recruit-interview-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/interview-service/main.go` 构造 User、RefreshToken、Job、Profile、Resume、Application、Interview、Offer、Notification、Outbox、Authz、AI/chat/memory/usage/email 等 repository。
- 通过 `service.NewServices` 获取共享 `InterviewService` 所在的大聚合。

表边界：

- Owner 表：`interview_schedules`、`interview_feedback`
- 过渡只读表：`applications`（owner: recruitment）
- 授权 shared write：`event_outbox`（owner: platform）

潜在违规写风险：

- 面试取消、批量取消、反馈提交与 application lifecycle、通知/outbox 强相关，必须避免直接写 recruitment owner 表。
- 后续迁移应把 application 状态协作收敛为 owner API、domain event 或明确 process manager。

### 4.3 Notification

服务根：`smart-recruit-notification-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/email`
- `smart-recruit-domain-go/mq`
- `smart-recruit-domain-go/pkg/cache`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/notification-service/main.go` 构造 `NotificationRuntime`，依赖 User、Notification、Outbox、Inbox、EmailLog、Authz、EmailSender、MQ、cache。
- `internal/runtime` 直接引用 `smart-recruit-domain-go/service` 注册 notification runtime。

表边界：

- Owner 表：`notifications`、`email_logs`
- 过渡只读表：无
- 授权 shared write：`event_outbox`、`event_inbox`（owner: platform）

潜在违规写风险：

- 通知消费依赖 Outbox/Inbox 幂等和 EmailLog，一旦迁移应保留 retry/DLQ/idempotency。
- User/Authz 读取需要明确为 Identity owner query、快照或临时 adapter。

### 4.4 Identity

服务根：`smart-recruit-identity-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/identity-service/main.go` 构造 User、RefreshToken、Authz、InviteCode、UsageLog、Analytics repository。
- 直接构造 `AuthService`、`AdminService`、`AnalyticsService` 和 `ServiceAuthorizer`。

表边界：

- Owner 表：`users`、`refresh_tokens`、`invite_codes`、`roles`、`permissions`、`role_permissions`、`user_roles`、`user_data_scopes`、`authorization_audit_logs`
- 过渡只读表：`jobs`（owner: recruitment）、`interview_schedules`（owner: interview）
- 授权 shared write：`event_outbox`（owner: platform）

潜在违规写风险：

- UsageLog/Analytics repository 装配超出 Identity 纯 auth/RBAC 边界，后续 TASK-012 至 TASK-014 必须审慎分类。
- Auth/Authz 行为属于安全语义，迁移只允许保持兼容，不允许隐式改变 token、RBAC 或 data scope。

### 4.5 Recruitment

服务根：`smart-recruit-recruitment-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/oss`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/recruitment-service/main.go` 手动装配 Job、Candidate、Application、Admin、UsageStats、Collaboration、Taxonomy 等 shared service。
- 仍构造 Interview、Offer、Notification、Authz、Outbox 等跨上下文 repository 或 service dependency。

表边界：

- Owner 表：`jobs`、`candidate_profiles`、`resumes`、`resume_parse_runs`、`resume_profiles`、`resume_educations`、`resume_experiences`、`resume_projects`、`resume_skills`、`applications`、`application_status_transitions`、`departments`、`job_locations`、`department_locations`、`candidate_notes`、`candidate_tags`、`candidate_tag_assignments`、`follow_up_tasks`
- 过渡只读表：`interview_schedules`（owner: interview）
- 授权 shared write：`event_outbox`、`event_inbox`（owner: platform）

潜在违规写风险：

- Recruitment 是核心最大上下文，当前还承载 application lifecycle 与多服务协作焦点。
- 后续 TASK-015 至 TASK-018 必须特别保护 application status transitions、OSS/resume parsing outbox、collaboration 和 taxonomy 语义。

### 4.6 Analytics

服务根：`smart-recruit-analytics-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/analytics-service/main.go` 构造 `AnalyticsService`，依赖 AnalyticsRepo、AuthzRepo 和 ServiceAuthorizer。
- 当前服务是 reporting/read model facade，尚未完成事件投影终态隔离。

表边界：

- Owner 表：`analytics_projection_events`、`analytics_projection_checkpoints`
- 过渡只读表：`jobs`、`applications`、`application_status_transitions`、`interview_schedules`、`interview_feedback`、`offers`、`notifications`、`candidate_match_evaluations`
- 授权 shared write：`event_inbox`（owner: platform）

潜在违规写风险：

- Analytics 读多张事务 owner 表是明确过渡债务，后续只能向 projection/read model 收敛，不得写回事务业务状态。
- Reporting 口径迁移必须保持兼容，不能因 DDD 分层改变统计口径。

### 4.7 AI Agent

服务根：`smart-recruit-ai-agent-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/ai`
- `smart-recruit-domain-go/mq`
- `smart-recruit-domain-go/repository`
- `smart-recruit-domain-go/service`

当前装配事实：

- `cmd/ai-agent-service/main.go` 通过 `service.NewServices` 构造 AI chat、agent run、memory、prompt、skill、MCP、embedding、notification、outbox、usage 等大聚合。
- `internal/runtime` 和 adapters 仍直接引用 shared `service`。

表边界：

- Owner 表：`ai_chat_sessions`、`ai_chat_history`、`ai_session_summaries`、`ai_tool_traces`、`agent_runs`、`agent_run_events`、`agent_run_steps`、`candidate_match_evaluations`、`candidate_match_evidence`、`ai_memories`、`ai_embeddings`、`ai_usage_auth_contexts`、`llm_providers`、`llm_models`、`embedding_providers`、`embedding_models`、`prompt_templates`、`prompt_versions`、`agent_configs`、`agent_tool_bindings`、`mcp_servers`、`mcp_tool_logs`、`mcp_tool_policies`、`agent_capability_bindings`、`ai_skills`、`ai_skill_versions`、`ai_skill_tools`、`agent_skills`、`agent_skill_versions`
- 过渡只读表：`applications`、`candidate_profiles`、`resumes`、`resume_parse_runs`、`resume_profiles`、`resume_educations`、`resume_experiences`、`resume_projects`、`resume_skills`
- 授权 shared write：`event_outbox`、`event_inbox`、`third_party_usage_logs`（owner: platform）

潜在违规写风险：

- AI Agent 依赖外部 provider、MCP policy、credential encryption、long-running agent run、embedding fallback 和招聘读模型，风险最高。
- 后续 TASK-022 至 TASK-025 必须保持真实 provider 调用 fake/env-gated，且不得泄露 credential。

### 4.8 Worker

服务根：`smart-recruit-worker-service`

当前 `smart-recruit-domain-go` 依赖：

- `smart-recruit-domain-go`
- `smart-recruit-domain-go/mq`

当前装配事实：

- `cmd/worker-service/main.go` 主要装配 worker runtime、DB/Redis/RabbitMQ health、workload toggles 和 MQ dependency。
- 当前尚未拥有本地 owner 表。

表边界：

- Owner 表：无
- 过渡只读表：无
- 授权 shared write：`event_outbox`、`event_inbox`（owner: platform）

潜在违规写风险：

- Worker 最终必须通过 owner service contract 或 owner domain/application port 执行业务写入，不能成为绕过领域规则的后台后门。
- TASK-026 至 TASK-028 应记录每个 workload 的 owner、输入事件、幂等 key、retry/DLQ 和停止条件。

## 5. 迁移顺序风险基线

| 顺序 | 服务 | 当前主要债务 | 后续迁移关注点 |
|---:|---|---|---|
| 1 | Offer | `service.NewServices` 大聚合；非 owner repository 被装配 | 本地 Offer domain/application；application snapshot 只读边界；outbox |
| 2 | Interview | `service.NewServices` 大聚合；application lifecycle 协作 | schedule/cancel/feedback 状态机；通知/outbox 一致性 |
| 3 | Notification | shared notification runtime；email/MQ/cache 适配在 shared module | unread/summary/mark read；Inbox 幂等；email env gate |
| 4 | Identity | shared auth/admin/analytics service；安全语义高风险 | JWT/refresh/RBAC/data scope/audit 行为兼容 |
| 5 | Recruitment | 最大核心域；application/collaboration/taxonomy/OSS/outbox 混合 | job/candidate/resume/application 分阶段本地化 |
| 6 | Analytics | 过渡只读多 owner 表 | 投影化；只读债务清单；reporting 口径兼容 |
| 7 | AI Agent | provider/MCP/embedding/agent-run/credential 复杂 | domain/application 分段；fake/env-gated provider；审计 |
| 8 | Worker | 后台 workload 可能绕过 owner 规则 | workload owner contract；retry/DLQ；幂等；readiness |

## 6. 当前不处理项

- 不删除或移动 `smart-recruit-domain-go` 中的业务代码。
- 不改变 `smart-recruit-deploy/mysql-table-ownership.json`。
- 不修改 `db.sql`、migration、protobuf 或服务 go.mod。
- 不判断 shared repository 的每一处实际 SQL 写路径是否已完全符合 owner 规则；本 TASK 只建立后续可比较 baseline。
- 不把历史 `.spec/backend-ddd-microservices-evolution` 当作当前执行合同；它只作为过往知识和风险参考。
