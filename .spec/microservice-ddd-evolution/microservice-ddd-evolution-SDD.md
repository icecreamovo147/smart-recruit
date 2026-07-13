# Microservice DDD Evolution SDD

## 1. Existing Architecture Summary

当前仓库处于微服务运行时拆分后的中间状态。

后端服务根已经存在于 `go.work` 中：

- `smart-recruit-gateway`
- `smart-recruit-identity-service`
- `smart-recruit-recruitment-service`
- `smart-recruit-interview-service`
- `smart-recruit-offer-service`
- `smart-recruit-notification-service`
- `smart-recruit-ai-agent-service`
- `smart-recruit-analytics-service`
- `smart-recruit-worker-service`
- `smart-recruit-commons`
- `smart-recruit-platform-go`
- `smart-recruit-proto`

`README.md` 当前描述的架构是：

- Gateway 层使用 Gin，处理 HTTP API、RBAC、限流、请求体限制、SSE 和 HTTP 到 gRPC 转换。
- 后端微服务层使用 gRPC，Identity、Recruitment、Interview、Offer、Notification、AI Agent、Analytics、Worker 独立构建和启动。
- 各服务共享 protobuf、platform 与 domain 模块。

当前 gRPC 暴露边界已经按服务拆出：

- `identity-service` 注册 `AuthService`、部分 `AdminService`。
- `recruitment-service` 注册 `JobService`、`AdminService`、`CandidateService`、`ApplicationService`、`CollaborationService`。
- `interview-service` 注册 `InterviewService`。
- `offer-service` 注册 `OfferService`。
- `notification-service` 注册 `NotificationService`。
- `ai-agent-service` 注册 `AIService`、`LlmConfigService`、`PromptService`、`AgentConfigService`、`MCPService`、`SkillService`、`AgentSkillService`、`RecruitingIntelligenceService`、`EmbeddingConfigService`。
- `analytics-service` 注册 analytics/reporting 相关 `AdminService` 子集。
- `worker-service` 承载后台 workload。

但业务代码仍明显集中在 `smart-recruit-commons`：

- `smart-recruit-commons/model`: 领域实体、状态、配置模型。
- `smart-recruit-commons/repository`: GORM repository。
- `smart-recruit-commons/service`: 大量业务服务、消费者、runtime helper、AI/Embedding/Notification/Offer/Interview/Application 等业务规则。
- `smart-recruit-commons/mq`、`oss`、`email`、`ai`: 具体基础设施能力和外部系统适配。
- `smart-recruit-commons/migrations`: 统一数据库迁移。

微服务 `cmd/*/main.go` 仍通过 `repository.New*Repo` 和 `service.New*Service` 或 `service.NewServices` 装配共享领域服务。例如 Offer、Interview、AI Agent 仍构造大量共享 repository 并调用 `service.NewServices`，Recruitment 虽然局部手动装配，但仍直接使用 `smart-recruit-commons/repository` 和 `smart-recruit-commons/service`。

表归属已有基础约束：`smart-recruit-deploy/mysql-table-ownership.json` 定义单 MySQL 实例下的 logical owner，`docs/mysql-table-ownership.md` 要求每张表只有一个 owner，跨服务访问需声明，新直接跨服务写应避免并优先使用 RabbitMQ 事件和 Outbox/Inbox。

## 2. Problem Analysis

当前架构的主要问题不是“微服务目录不存在”，而是服务内部尚未形成自治 DDD 边界。

关键问题：

- 业务规则、状态机、repository、infrastructure adapter 仍在共享 `smart-recruit-commons` 中，服务本地只承担启动装配和 gRPC facade。
- `smart-recruit-commons/service.Services` 是跨上下文大聚合，容易让一个服务为了使用一个能力而构造大量无关 repository 和 service。
- domain 层、application 层、infrastructure 层、interfaces 层边界不清，许多 `service/*.go` 同时处理 proto、权限、事务、repository、外部依赖、业务规则和响应组装。
- 仓储接口与 GORM 实现未按服务 owner 反转依赖，导致服务自治和单元测试隔离困难。
- 跨上下文依赖仍通过直接 repository/table 读取大量存在，虽然表归属 manifest 已记录，但服务内部边界未完全收敛。
- Analytics、AI Agent、Worker 与事务域存在较多读写/事件/后台协作，需要在后期阶段谨慎迁移。
- 如果直接大规模移动 `smart-recruit-commons`，会产生 protobuf、schema、部署和测试的大面积风险。

因此，本迁移应采用“先建立目标分层范式，再按服务顺序复制范式”的方式。

## 3. Proposed Design

### 3.1 Target Service Shape

每个业务微服务最终应收敛到如下结构。具体文件名可按服务需要调整，但职责必须等价：

```text
smart-recruit-<context>-service/
  cmd/<context>-service/
    main.go
  internal/
    domain/
      model/
      valueobject/
      repository/
      service/
      event/
      policy/
    application/
      command/
      query/
      dto/
      port/
      service/
    infrastructure/
      persistence/
      mq/
      cache/
      client/
      oss/
      email/
      ai/
    interfaces/
      grpc/
      event/
      mapper/
    runtime/
```

职责定义：

- `domain`: 只表达本上下文业务概念、规则、状态转换、领域事件和仓储 port，不依赖外部技术。
- `application`: 编排 use case，定义事务边界、权限策略调用、跨上下文 port、事件发布时机和 DTO。
- `infrastructure`: 实现 repository port、外部 client port、MQ publisher/consumer、cache、OSS、SMTP、AI provider 等技术细节。
- `interfaces`: gRPC server、event handler、proto mapper、transport validation。
- `runtime`: 组合配置、DB、Redis、RabbitMQ、Nacos、health、metrics、trace、logger 和依赖注入。

### 3.2 Shared Module Target

`smart-recruit-commons` 应从“大共享业务域”收缩为 shared kernel / legacy migration bridge。长期允许保留：

- 通用分页、加密、JWT 辅助、authz 常量等稳定通用包。
- 通用 event envelope、Outbox/Inbox 基础类型或 helper，前提是不包含具体业务状态机。
- 通用 test helper 或 migration runner，前提是 owner 明确。
- 过渡期 legacy adapter，必须有迁移债务记录和删除条件。

长期不应保留：

- Offer/Interview/Application/Notification/Auth/AI 等具体业务服务实现。
- 具体服务 owner 的 GORM repository 实现。
- 服务 owner 的业务实体和状态机。
- 需要访问具体业务表的跨上下文聚合服务。

在所有服务迁移和 shared kernel 范围收敛后，`smart-recruit-commons` 必须作为本功能点最后阶段重命名为 `smart-recruit-commons`。该重命名不得提前执行，避免出现“commons 名称下仍包含大量业务 model/repository/service”的语义错位。重命名完成后，旧 module/import path 不应再出现在业务代码、测试、构建脚本、部署配置或文档中。

### 3.3 Required Migration Order

后续 TASK 必须按以下阶段组织：

1. Offer
2. Interview
3. Notification
4. Identity
5. Recruitment
6. Analytics
7. AI Agent
8. Worker
9. Final Commons Rename

顺序理由：

- Offer 边界相对清晰，适合作为 DDD 服务自治范式试点。
- Interview 与 Application/Identity 有交互，但领域范围比 Recruitment 小，适合作为第二个迁移对象。
- Notification 可验证 Outbox/Inbox、邮件、实时通知和异步幂等。
- Identity 涉及认证授权安全语义，需在前三个服务边界经验稳定后迁移。
- Recruitment 是核心域最大上下文，应在范式、auth、通知、面试、Offer 边界稳定后迁移。
- Analytics 应在事务域 owner 更明确后迁向投影/read model。
- AI Agent 依赖复杂、外部系统多、状态和异步多，靠后迁移。
- Worker 应最后重组，避免后台任务绕过尚未稳定的 owner service contract。
- Final Commons Rename 必须在 Worker 之后执行，因为只有所有 owner context 和后台 workload 收敛后，才能判断哪些共享内容真正属于 commons。

### 3.4 Per-Service Migration Pattern

每个服务迁移应使用一致的步骤：

1. 盘点当前服务暴露的 protobuf service、runtime deps、repository、表访问、事件、配置、测试。
2. 在服务本地创建 DDD 目录骨架和架构文档。
3. 迁出 domain model/value object/state machine/domain event/repository interface。
4. 迁出 application service 或 command/query handler，保留原有业务语义。
5. 在 infrastructure 中实现 GORM repository、event publisher/consumer、cache/client adapter。
6. 在 interfaces/grpc 中实现 proto 到 application DTO 的映射和 gRPC server。
7. 在 runtime 中装配本地 application + infrastructure，替代对 `smart-recruit-commons/service` 的依赖。
8. 迁移或新增测试。
9. 删除或标记 shared domain 中对应业务实现为 deprecated，直到后续 TASK 安全移除。
10. 更新边界检查、表访问声明、报告和 knowledge impact。

所有服务完成上述步骤后，执行 shared cleanup 与最终 rename：

1. 清点 `smart-recruit-commons` 剩余包并分类为 shared kernel、platform-adjacent、legacy debt 或应删除业务残留。
2. 删除或迁出所有具体业务 `model/repository/service` 残留。
3. 将允许保留的通用能力整理到 commons 命名和目录结构。
4. 重命名目录和 Go module path 为 `smart-recruit-commons`。
5. 更新全仓库 import path、`go.work`、Dockerfile、Compose/K8s/Nacos/脚本、README、SPEC/Harness/knowledge 文档和边界检查。
6. 运行全仓库 Go 测试、边界检查、表归属检查和启动/构建验证。

### 3.5 Offer Pilot Target

Offer 作为第一个试点，目标结构建议为：

```text
smart-recruit-offer-service/internal/
  domain/
    model/offer.go
    model/offer_event.go
    repository/offer_repository.go
    service/offer_policy.go
    event/offer_events.go
  application/
    command/create_offer.go
    command/send_offer.go
    command/withdraw_offer.go
    command/decide_offer.go
    query/get_offer.go
    query/list_offers.go
    port/application_snapshot_reader.go
    port/outbox_publisher.go
  infrastructure/
    persistence/gorm_offer_repository.go
    persistence/gorm_offer_event_repository.go
    client/recruitment_client.go
    mq/outbox_publisher.go
  interfaces/
    grpc/offer_server.go
    mapper/offer_mapper.go
  runtime/
    runtime.go
```

Offer 阶段不得直接改变 protobuf。若需要新的 Recruitment snapshot API 或事件，必须作为 Hard Stop 或后续独立 TASK 明确确认。

### 3.6 Final Commons Target

最终 `smart-recruit-commons` 建议只保留如下类别：

```text
smart-recruit-commons/
  pkg/
    pagination/
    crypto/
    authz/
    errors/
  event/
    envelope.go
  testutil/
  migration/
```

具体保留规则：

- `pkg/pagination`、`pkg/crypto`、稳定 authz 常量、轻量错误契约可保留为 shared kernel。
- event envelope、Outbox/Inbox 通用元数据可以保留，但不得包含具体业务状态机。
- migration runner 可短期保留，长期是否独立为 platform/migration 能力由后续 TASK 决定。
- `ai`、`oss`、`email`、`mq` 按性质拆分：通用 client/factory 可进入 commons 或 platform，带业务语义的编排必须迁入 owner service。
- `model`、`repository`、`service` 不应以当前业务混合形态保留在 commons 中。

## 4. Data Structure Changes

本 draft 阶段不实施任何数据结构变化。

后续迁移原则：

- 优先复用现有表和 `smart-recruit-deploy/mysql-table-ownership.json`。
- 每个服务本地 repository 只能写 owner 表，非 owner 表只允许按 manifest 声明的 transitional read。
- 新增表、修改 migration、修改 `db.sql`、修改 table ownership manifest 都是高风险 TASK，必须显式授权。
- 如果服务需要本地只读 projection，应先在 SPEC/SDD/TASK 中说明事件来源、投影表 owner、重建策略、幂等策略和回滚策略。
- 如果迁移需要按服务拆分 schema 或物理数据库，必须作为后续独立 feature 或 SPEC 修订，不在本次初始 SDD 中直接要求。
- migrations 短期统一保留；当 `smart-recruit-commons` 收缩为 `smart-recruit-commons` 时，可将 migration runner 作为 commons/platform-adjacent 能力保留，但不得在该阶段顺带执行物理拆库。

## 5. API and Interface Changes

默认不改变 public API。

接口迁移策略：

- `smart-recruit-proto/proto/recruitment.proto` 继续作为 canonical protobuf source。
- protobuf 短期保持单文件和现有契约；按服务拆分 proto 包不属于本功能点默认实现，应另起 feature 或修订 SPEC。
- 服务内部 `interfaces/grpc` 可以重新组织 server 实现，但必须保持现有 rpc 入参、出参、错误码和语义兼容。
- Gateway 仍通过已有 route mode 与 service gRPC address 调用后端。
- 如果需要新增内部 service-to-service query API，例如 Offer 查询 Application snapshot，应优先复用已有 protobuf；若必须新增 rpc，则作为 Hard Stop 等待确认。
- 事件接口必须定义 producer、consumer、event type、version、aggregate id、idempotency key 和兼容策略。
- Interfaces 层不得直接访问 GORM repository；必须调用 application 层。

## 6. Algorithm or Workflow Changes

本迁移关注架构归属，不主动改变业务算法。

工作流迁移策略：

- 先保持当前同步调用和表访问语义，再通过后续 TASK 逐步替换为 owner API 或事件。
- 对每个跨上下文写，识别当前行为、事务边界和补偿方式，再迁入 owner context。
- 对每个异步流程，保留 Outbox 写入与消费语义，迁移后补充幂等测试。
- 对每个状态机，例如 application status、interview lifecycle、offer lifecycle，迁移时先建立 domain unit test，验证状态转换与错误处理不变。
- 对每个 query/list/detail API，先确认当前排序、分页、筛选、权限范围、空值处理、错误码，再迁移 mapper 和 application query。

推荐的服务级工作流：

```text
interfaces/grpc request
  -> mapper converts proto to command/query DTO
  -> application handler validates use case and starts transaction if needed
  -> domain model/policy performs invariant and state transition
  -> repository port persists through infrastructure implementation
  -> event/outbox port records domain event when required
  -> mapper converts result to proto response
```

## 7. Configuration Design

默认不改变现有配置机制。

当前配置事实：

- 微服务使用 `CONFIG_PATH` 加载 YAML，并通过环境变量覆盖敏感或环境相关配置。
- Docker Compose 微服务配置使用 `CONFIG_PATH: /app/config/config.example.yaml` 与环境变量注入 DSN、Redis、RabbitMQ、Nacos、internal auth 等。
- Nacos seed config 中维护 gateway 和 service targets。
- `smart-recruit-platform-go/serviceconfig` 与 `smart-recruit-gateway/config` 承担运行时配置校验和 fallback。

迁移设计：

- 服务本地 DDD 分层不应改变配置来源。
- Runtime 层负责将配置转换为 application/infrastructure 需要的 typed config。
- Domain 层不得读取环境变量或配置文件。
- Infrastructure 层可以接收 runtime 注入的配置对象，但不得自行散落读取全局 env，除非现有实现暂时无法迁移且已记录债务。
- 真实 `config.yaml` 必须保持不被 git 跟踪。

## 8. Compatibility Strategy

兼容策略分四层。

第一层：外部行为兼容。

- HTTP API、gRPC response、错误码、分页、排序、权限和用户可见流程保持兼容。

第二层：运行时兼容。

- 每个服务继续支持当前 `--serve --addr`、health、metrics、Nacos、gRPC internal auth、config env override。
- 最终 rename 阶段必须保持启动脚本、Docker 构建、Compose/K8s 引用、Nacos seed config 和 Go workspace 可用。

第三层：数据兼容。

- 继续使用现有表和 migrations。
- 每个服务迁移后通过 table ownership check 验证 owner 写权限。
- 不在未授权 TASK 中修改 schema。
- Analytics 的目标数据来源为事件投影；短期只读跨表访问仅作为受控过渡债务，不应成为长期终态。

第四层：测试兼容。

- 迁移前后服务 `go test ./...` 必须通过。
- 关键旧测试应迁移到新服务本地对应层，不能删除现有测试来让迁移通过。

## 9. Error Handling and Fallback Design

- Domain 层返回可识别的业务错误或 typed error，不直接构造 protobuf response。
- 错误处理采用 shared lightweight error contract 加服务本地 typed errors；interfaces 层负责映射为现有 protobuf `Code` / `Message`。
- Application 层负责将 domain/infrastructure 错误分类为业务失败、权限失败、依赖失败、幂等重复、可重试失败。
- Interfaces 层负责把 application error 映射为现有 protobuf `Code` / `Message` 语义。
- Infrastructure 层必须保留底层错误上下文，但不得暴露密钥、DSN 或敏感 payload。
- 异步 consumer 必须显式处理重复、乱序、依赖缺失、外部依赖失败、poison message 和 DLQ。
- AI/Embedding/SMTP/OSS/RabbitMQ 的现有 fallback 行为迁移前必须有测试或手工验证记录。

## 10. Observability and Debug Output Design

每个服务迁移后应保留或补强：

- service name、operation、request id/trace id、actor/account type、aggregate id/event id。
- gRPC request latency、error count、dependency latency、DB latency、Redis/RabbitMQ/OSS/SMTP/AI dependency errors。
- Outbox/Inbox/consumer backlog、retry、DLQ、idempotency hit。
- Runtime startup dependency summary 和 readiness failure reason。
- TASK report 中的验证命令、耗时、exit code 和失败输出摘要。

日志设计要求：

- Domain 层默认不直接打日志，除非记录纯业务诊断且不含敏感数据。
- Application 层记录 use case 边界和业务失败上下文。
- Infrastructure 层记录外部依赖失败、重试和连接状态。
- Interfaces 层记录请求级别映射失败和 contract violation。

## 11. Testing Strategy

后续 TASK 生成时应按服务和层级分配测试。

通用测试类型：

- Domain unit tests: 纯 Go 测试，不连接 DB/Redis/RabbitMQ，验证实体、值对象、状态机、策略和领域事件。
- Application tests: 使用 fake repository/client/event publisher 验证用例编排、事务意图、权限调用、错误处理。
- Infrastructure tests: 针对 GORM repository、MQ、Redis、OSS/SMTP/AI adapter 的现有测试迁移或补充；需要外部依赖的测试必须可跳过或明确 env gate。
- Interfaces/runtime tests: 验证 gRPC server 注册、nil dependency guard、proto mapper、response 兼容、health/metrics wiring。
- Architecture tests: 验证 import boundary、domain 无 infrastructure 依赖、interfaces 不直接访问 persistence、服务不写非 owner 表。
- Boundary checks: 根目录脚本负责全局规则，服务本地 architecture tests 负责服务内部 DDD 分层依赖。
- Integration checks: `go test ./...`、`node scripts/check-mysql-table-ownership.mjs`、可用时运行后端边界检查。

服务专项测试建议：

- Offer: offer lifecycle、offer event append、candidate accept/reject、withdraw、list by application、candidate list。
- Interview: schedule/update/cancel/batch cancel、feedback、interviewer/candidate query、application-linked edge cases。
- Notification: unread count、mark read/all read、summary、email log、Outbox/Inbox idempotency。
- Identity: register/login/refresh/revoke/principal/RBAC/data scope/auth audit。
- Recruitment: job lifecycle、candidate profile/resume/application status/collaboration/taxonomy/usage stats。
- Analytics: projection read model、report query、owner read-only guarantees。
- AI Agent: chat/session/agent run/prompt/skill/MCP/embedding/fallback/audit。
- Worker: workload toggles、consumer lifecycle、retry/DLQ、idempotency、graceful shutdown。

## 12. Migration Risks

- RISK-001: 大量业务代码共享在 `smart-recruit-commons`，迁移时容易产生循环依赖或重复业务规则。
- RISK-002: `smart-recruit-proto` 是单一大 proto 文件，按服务拆分接口时容易触发 public contract 风险。
- RISK-003: 单 MySQL 实例下，跨服务读写边界容易被 GORM repository 直接访问绕过。
- RISK-004: `service.NewServices` 聚合大量依赖，拆分时可能遗漏 AI/Notification/Outbox/Usage/Authz 等隐式依赖。
- RISK-005: Identity 迁移涉及安全语义，任何错误都可能影响登录、权限和审计。
- RISK-006: AI Agent 迁移复杂，涉及长任务、流式响应、MCP、Embedding、provider fallback、加密密钥和审计。
- RISK-007: Worker 最后迁移前，如果 owner context 尚未稳定，后台任务可能继续保留跨上下文写债务。
- RISK-008: 若 TASK 过大，harness-pipeline 虽能串行执行，但 review 和 rollback 粒度会变差。
- RISK-009: 历史 `.spec/backend-ddd-microservices-evolution` 是早期两服务阶段文档，若误当当前合同会与当前仓库事实冲突。
- RISK-010: 配置和密钥曾出现真实 `config.yaml` 被提交的风险，后续 TASK 必须保持 secrets hygiene。
- RISK-011: 最终重命名 `smart-recruit-commons -> smart-recruit-commons` 影响 Go module path、import path、`go.work`、Dockerfile、脚本、部署配置和文档，必须作为最后的独立高风险 TASK 执行。
- RISK-012: 如果过早重命名，`smart-recruit-commons` 可能变成新的业务垃圾桶；因此必须在业务代码迁出和 shared kernel 范围确认后执行。

## 13. Implementation Boundaries

本 draft 阶段只允许创建：

- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md`
- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md`

后续 prepare-harness 应生成但本阶段不得生成：

- `TASKS.md`
- `AGENT_RULES.md`
- `task-scope.json`
- `acceptance/`
- `prompts/`
- `scripts/`
- `reports/`

后续 TASK scope 设计建议：

- 先创建跨服务 DDD 分层规范和边界检查 TASK。
- 每个服务至少拆成骨架、domain、application、infrastructure、interfaces/runtime、tests、shared cleanup 七类 TASK。
- 所有服务完成后，必须增加最终 commons rename TASK，范围覆盖目录名、Go module path、import path、workspace、构建、部署、文档、脚本和边界检查。
- 每个涉及 shared module、proto、schema、auth、deployment 的 TASK 默认 `requiresHumanConfirmation: true`。
- 最终 commons rename TASK 必须 `requiresHumanConfirmation: true`，并要求全仓库验证。
- 每个服务完成迁移前不得开始下一个服务迁移，除非只是文档/检查类前置 TASK 且不改变业务代码。

禁止边界：

- 未授权修改 public API。
- 未授权修改 protobuf。
- 未授权修改 DB schema/migration。
- 未授权修改 auth/authz/security 行为。
- 未授权修改 package manifests/lockfiles。
- 未授权引入新依赖。
- 未授权修改部署配置。
- 直接删除现有测试。
- 通过 `any`、吞错、降低校验来绕过编译或测试。

## 14. Alternatives Considered

### Alternative A: 一次性把 `smart-recruit-commons` 拆到所有服务

拒绝。风险过高，容易造成大规模编译破坏、行为漂移、重复代码和难以 review 的巨型 diff。

### Alternative B: 保持共享 `smart-recruit-commons`，只增加文档

拒绝。它不能解决服务自治、仓储 owner、DDD 分层和跨上下文边界问题。

### Alternative C: 先拆数据库

拒绝作为当前阶段目标。当前已有单 MySQL + logical ownership，直接物理拆库会放大迁移风险。应先完成代码边界、API/事件协作和表 owner 守护。

### Alternative D: 从 Recruitment 开始迁移

拒绝。Recruitment 是核心域最大上下文，涉及 job/candidate/resume/application/collaboration/taxonomy，多依赖、多风险。先用 Offer 建立范式更稳。

### Alternative E: 从 Worker 开始迁移

拒绝。Worker 是跨上下文后台执行器，应该在 owner service contract 稳定后收敛，否则容易把旧共享 domain 写路径固化到后台任务中。

### Alternative F: 迁移开始时立即重命名为 `smart-recruit-commons`

拒绝。当前 `smart-recruit-commons` 仍包含大量具体业务 model、repository、service 和基础设施编排。提前重命名会造成命名与内容不一致，并扩大所有后续 TASK 的 import path 变更噪音。重命名必须作为最后阶段执行。

## 15. Assumptions Requiring Confirmation

以下决策已由用户在本次修订中确认，后续 `prepare-harness` 应将其作为 TASK 拆分依据；若要变更，必须修订 SPEC/SDD。

- ARC-001: 用户确认本次 feature name 使用 `microservice-ddd-evolution`。
- ARC-002: 用户确认历史 `.spec/backend-ddd-microservices-evolution` 不作为本次 Harness 执行合同。
- ARC-003: 用户确认后续可以拆出大量 TASK，由 `harness-pipeline` 和 goal 模式持续执行。
- ARC-004: 用户确认迁移顺序不可被 pipeline 自动优化或重排。
- ARC-005: 用户确认每个服务的标准 DDD 目录可以放在 `smart-recruit-*-service/internal/` 下。
- ARC-006: 用户确认 `smart-recruit-commons` 最终保留 shared kernel 是允许的，而不是必须完全删除。
- ARC-007: 用户确认本阶段不主动修改 proto、schema、部署和 auth 行为。
- ARC-008: 每个服务最终拥有自己的 `internal/domain/model`，不直接复用 proto 或 GORM model 作为 domain model。
- ARC-009: 错误处理采用 shared lightweight error contract 加服务本地 typed errors。
- ARC-010: 架构边界检查采用根脚本与服务本地 architecture tests 双层机制。
- ARC-011: `smart-recruit-commons/ai`、`oss`、`email`、`mq` 按通用技术能力与业务编排语义拆分归属。
- ARC-012: Analytics 最终完全事件投影化，短期只读跨表作为受控过渡债务。
- ARC-013: Worker 短期保持单服务，通过 workload profile/toggle 拆分；是否拆 binary 后置。
- ARC-014: `smart-recruit-commons` 在所有服务迁移、共享业务代码清理和 shared kernel 范围确认后，作为本功能点最后阶段重命名为 `smart-recruit-commons`。

## 16. Open Questions

截至本次修订，无待确认 Open Questions。原 Open Questions 已按用户确认的推荐方案固化到 `Assumptions Requiring Confirmation` 中。
