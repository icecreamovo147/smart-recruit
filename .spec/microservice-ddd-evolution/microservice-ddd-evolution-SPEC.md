# Microservice DDD Evolution SPEC

## 1. Background

Smart Recruit 后端已经完成一次微服务运行时拆分。当前仓库包含独立可构建的 Go 服务根：

- `smart-recruit-gateway`
- `smart-recruit-identity-service`
- `smart-recruit-recruitment-service`
- `smart-recruit-interview-service`
- `smart-recruit-offer-service`
- `smart-recruit-notification-service`
- `smart-recruit-ai-agent-service`
- `smart-recruit-analytics-service`
- `smart-recruit-worker-service`

同时，当前核心业务模型、GORM 仓储、领域/应用服务、消息、OSS、AI、email、migration 仍主要集中在 `smart-recruit-commons`。各微服务通过 `cmd/*/main.go` 装配 `smart-recruit-commons/repository` 与 `smart-recruit-commons/service`，再通过各自 `internal/runtime` 暴露 gRPC 服务。`smart-recruit-proto` 是当前唯一 protobuf 契约来源，`smart-recruit-platform-go` 承载配置、Nacos、日志、健康检查、metrics、trace、metadata、gRPC helper 等平台能力。

因此，当前状态是“微服务部署形态 + 共享领域内核”的迁移中间态，还不是标准的服务自治微服务 + DDD 架构。用户要求将本次架构迁移作为 `.spec/` 下的新功能点，后续通过 `harness-pipeline` skill + goal 模式串行执行所有 TASK，并按以下顺序迁移所有后端微服务：

```text
Offer -> Interview -> Notification -> Identity -> Recruitment -> Analytics -> AI Agent -> Worker
```

本 SPEC 定义目标能力、兼容要求、验收标准和边界规则。本模式仅创建 SPEC + SDD，不生成 TASKS 和 Harness 文件，不修改业务代码。

## 2. Goals

- 将后端从“共享 `smart-recruit-commons` 领域内核”演化为更标准、规范的“服务自治微服务 + DDD”架构。
- 每个业务微服务最终拥有本服务上下文内的 `domain`、`application`、`infrastructure`、`interfaces`、`runtime` 等组件，而不是长期依赖共享业务 `model/repository/service`。
- 严格按 `Offer -> Interview -> Notification -> Identity -> Recruitment -> Analytics -> AI Agent -> Worker` 顺序迁移。
- 保持当前前端、HTTP Gateway、gRPC protobuf、启动脚本、部署拓扑和核心业务行为兼容，除非后续 TASK 明确获得人类确认允许变更。
- 在迁移过程中逐步收缩 `smart-recruit-commons` 的职责，使其最终仅保留真正跨上下文共享、稳定、无业务归属争议的 shared kernel / platform-adjacent 能力。
- 在所有服务自治迁移、共享业务代码清理和 shared kernel 范围收敛完成后，将 `smart-recruit-commons` 作为本功能点最后的重命名任务迁移为 `smart-recruit-commons`。
- 使每个微服务的业务规则、状态机、仓储接口、事务边界、消息发布/消费、外部依赖适配和对外接口边界可审查、可测试、可替换。
- 保持现有单 MySQL 实例作为过渡运行事实，但通过表归属、仓储边界、跨服务 API/事件协作逐步强化数据所有权。
- 支持后续由 `spec-harness prepare-harness` 生成大量 TASK，并由 `harness-pipeline` 串行执行到完成。

## 3. Non-Goals

- 本 SPEC 不要求一次性完成所有微服务迁移。
- 本 SPEC 不要求立即拆分物理数据库、schema 或引入新基础设施。
- 本 SPEC 不要求在 `draft-spec-sdd` 阶段生成 TASKS、acceptance、scripts、reports 或 `task-scope.json`。
- 本 SPEC 不允许在未确认的 TASK 中修改 public HTTP API、protobuf 行为、认证授权语义、数据库 schema、部署安全策略或前端行为。
- 本 SPEC 不要求把 `smart-recruit-commons` 的能力全部删除；迁移后它的 shared kernel 内容会保留，但最终目录与 Go module 名称应在最后阶段重命名为 `smart-recruit-commons`。
- 本 SPEC 不鼓励机械搬目录。迁移必须以限界上下文、业务规则归属、事务边界和依赖反转为准，而不是简单复制文件。
- 本 SPEC 不要求每个服务使用完全相同的内部文件命名；但每个服务必须满足一致的 DDD 分层职责和边界约束。

## 4. User-Facing Behavior

- HR 管理端、候选人端、面试官端的可见行为必须保持兼容。
- Gateway 的 HTTP 路由、JWT/Refresh Token、RBAC、限流、SSE、请求体限制、错误响应结构、Swagger 可用性不得因 DDD 内部重组而发生未确认变化。
- 当前 gRPC protobuf 契约不得因内部代码搬迁而改变语义。
- Offer、Interview、Notification、Identity、Recruitment、Analytics、AI Agent、Worker 的迁移必须对用户工作流透明，除非某个 TASK 明确获得人类确认并包含兼容计划。
- 任何服务迁移过程中，如果出现新实现与旧实现并存、影子运行或切流，只有被确认的生产路径可以写入用户可见状态。
- 异步能力不可用时必须保留现有降级语义。例如通知、邮件、AI、Embedding、后台 Worker 的失败不应破坏核心事务路径，除非当前业务本身已有强依赖。

## 5. Functional Requirements

- FR-001: 每个迁移后的微服务必须拥有服务本地的 DDD 分层目录，至少表达 `domain`、`application`、`infrastructure`、`interfaces`、`runtime` 的职责边界。
- FR-002: 每个服务的 `domain` 层必须承载本上下文实体、值对象、聚合、状态机、领域服务、领域事件和仓储接口，不得依赖 GORM、Redis、RabbitMQ、gRPC、HTTP、Nacos、OSS、SMTP 或具体外部 SDK。
- FR-003: 每个服务的 `application` 层必须承载用例编排、事务边界、权限/策略调用、跨聚合协调、事件发布时机和命令/查询处理。
- FR-004: 每个服务的 `infrastructure` 层必须承载 GORM repository 实现、Redis/MQ/OSS/SMTP/AI/gRPC client 等具体适配。
- FR-005: 每个服务的 `interfaces` 层必须承载 gRPC server、proto request/response 映射、传输层校验和对外接口适配。
- FR-006: 每个服务的 `runtime` 层必须负责启动装配、配置加载、健康检查、metrics/trace/logging wiring、Nacos/discovery、依赖注入和 graceful shutdown。
- FR-007: 迁移顺序必须固定为 `Offer -> Interview -> Notification -> Identity -> Recruitment -> Analytics -> AI Agent -> Worker`。后续 TASKS 必须按该顺序组织，除非用户显式修订 SPEC。
- FR-008: Offer 迁移必须作为第一个服务自治 DDD 试点，迁出 offer lifecycle、offer events、offer repository interface、GORM persistence、application service、gRPC adapter 和相关测试。
- FR-009: Interview 迁移必须在 Offer 之后进行，迁出 interview schedules、feedback、batch cancellation、interviewer/candidate listing、interview lifecycle 规则和相关跨上下文协作。
- FR-010: Notification 迁移必须在 Interview 之后进行，迁出 notification persistence、unread count、summary、mark read、email coordination、SSE/realtime 相关接口、Outbox/Inbox 消费适配和幂等处理。
- FR-011: Identity 迁移必须在 Notification 之后进行，迁出 auth、refresh token、principal、RBAC、data scope、invite code、auth audit、staff user 等身份权限能力。
- FR-012: Recruitment 迁移必须在 Identity 之后进行，迁出 job、candidate profile、resume、application lifecycle、collaboration、taxonomy、usage stats 等招聘核心域能力。
- FR-013: Analytics 迁移必须在 Recruitment 之后进行，收敛为报表查询、投影/读模型和分析查询能力，不得写回事务型业务状态。
- FR-014: AI Agent 迁移必须在 Analytics 之后进行，迁出 AI chat、agent run、prompt、agent config、MCP、skill、agent skill、recruiting intelligence、embedding config、AI usage audit、provider fallback 等复杂 AI 上下文能力。
- FR-015: Worker 迁移必须最后进行，重组后台任务运行器，使 Outbox、Inbox、DLQ、notification、email、resume parsing、embedding、agent run、analytics projection 等 workload 以 owner service 的领域事件和 application contract 为边界。
- FR-016: 每迁移一个服务，必须减少该服务对 `smart-recruit-commons/service`、`smart-recruit-commons/repository`、共享业务 model 的直接依赖，并记录剩余依赖是否为临时兼容、shared kernel 或待迁移债务。
- FR-017: 每个服务的仓储接口必须定义在本服务 `domain` 或 application port 内，具体 GORM 实现必须在本服务 `infrastructure` 内。
- FR-018: 当前共享 MySQL 允许作为过渡运行模式，但写权限必须遵守 `smart-recruit-deploy/mysql-table-ownership.json` 的 owner 规则；新直接跨服务写必须禁止。
- FR-019: 跨上下文写协作必须通过 owner service API、domain event、Outbox/Inbox、幂等 consumer 或明确 Saga/补偿流程完成。
- FR-020: 跨上下文读协作必须优先通过 owner service query API、只读 projection、快照表或明确兼容 adapter 完成；新增直接跨 owner 表 join 必须作为 Hard Stop。
- FR-021: Gateway 必须继续作为 transport/policy boundary，不得承载核心业务状态机、事务规则或具体 GORM persistence。
- FR-022: `smart-recruit-proto` 必须继续作为 canonical protobuf source；任何 protobuf 变更必须单独确认并具备兼容策略。
- FR-023: `smart-recruit-platform-go` 必须继续承载服务运行时平台能力；业务规则不得迁入 platform module。
- FR-024: 迁移必须加入或强化架构边界检查，至少能发现 domain 层依赖 infrastructure、interfaces 直接访问 persistence、服务跨上下文直接 import 非授权内部包、服务直接写非 owner 表等问题。
- FR-025: 每个服务迁移完成后必须有对应 domain unit tests、application tests、infrastructure tests、runtime/interface tests 或说明为什么某类测试不适用。
- FR-026: 后续 TASK 实施时必须遵守 SPEC + SDD + Harness 顺序，不得绕过 `TASKS.md`、scope check、agent-check、TASK report 和 evidence。
- FR-027: 所有服务迁移、共享业务代码清理、shared kernel 范围确认和依赖收敛完成后，本功能点必须以最终重命名 TASK 将 `smart-recruit-commons` 重命名为 `smart-recruit-commons`，并同步更新 Go module path、import path、`go.work`、构建脚本、部署引用、文档和边界检查。
- FR-028: 最终 `smart-recruit-commons` 只允许承载 shared kernel 和通用技术能力，不得重新接收具体业务上下文的 `model/repository/service` 实现。

## 6. Non-Functional Requirements

- NFR-001: 迁移必须小步、串行、可审查、可回滚；每个 TASK 必须在当前 TASK scope 内完成。
- NFR-002: 每个服务迁移后必须保持独立 `go test ./...` 可运行能力。
- NFR-003: 迁移后服务内部依赖方向必须保持单向：`interfaces/runtime` 可以依赖 `application`，`application` 可以依赖 `domain` port，`infrastructure` 实现 port，`domain` 不依赖外层。
- NFR-004: 新增或迁移后的代码必须使用 Go 标准测试、table-driven tests 和现有仓库风格；不得引入新依赖，除非 TASK 明确获批。
- NFR-005: 迁移不得降低现有健康检查、metrics、trace、logging、Nacos 注册、gRPC internal auth、配置覆盖和 graceful shutdown 能力。
- NFR-006: 迁移不得引入全局状态、进程内单例或本地内存状态作为多实例请求路径的必要依赖。
- NFR-007: Worker workload 必须可独立扩缩容，且不得绕过 owner service 或 owner domain 直接执行未授权业务写入。
- NFR-008: 迁移必须持续维护本地开发、Docker Compose、Kubernetes 示例和 Nacos seed config 的可理解性；部署配置变更必须由单独 TASK 明确授权。
- NFR-009: 所有日志、报告和错误输出不得泄露真实密钥、Token、AI API Key、OSS 密钥、SMTP 密码或候选人非必要个人数据。
- NFR-010: 架构迁移文档、TASK report 和 evidence 必须记录知识库影响；涉及 `.knowledge/` 的 TASK 必须按 AGENTS 规则完成 knowledge impact 检查。

## 7. Compatibility Requirements

- CR-001: 当前 HTTP API 行为必须保持兼容，除非后续 TASK 明确获批。
- CR-002: 当前 protobuf service 和 rpc 语义必须保持兼容，除非后续 TASK 明确获批。
- CR-003: 当前 `smart-recruit-gateway` 到各后端服务的 route mode 和 gRPC target 配置必须保持可用。
- CR-004: 当前本地开发入口 `./start-dev.sh`、Docker Compose 微服务部署和各服务 `--serve --addr` 启动方式必须保持可用，除非后续 TASK 明确修改并更新文档。
- CR-005: 当前单 MySQL 实例与 `smart-recruit-deploy/mysql-table-ownership.json` 的表归属检查必须继续有效。
- CR-006: 当前 Outbox/Inbox、RabbitMQ 队列、事件发布/消费、DLQ/retry 语义必须保持兼容。
- CR-007: 当前 JWT、Refresh Token、RBAC、data scope、内部 gRPC token/TLS 配置语义不得被未确认改变。
- CR-008: 当前 AI Agent runtime、ADK/legacy 配置、MCP policy、embedding fallback、agent run stream 语义不得被未确认改变。
- CR-009: 当前三端前端不应因为内部 DDD 重组需要同步改动；如必须改动前端，必须作为 Hard Stop 并独立确认。
- CR-010: 当前 `smart-recruit-commons/config/config.example.yaml` 作为本地配置模板的用途必须保持；真实 `config.yaml` 不得重新进入 git 跟踪。
- CR-011: 在最终重命名前，所有仍引用 `smart-recruit-commons` 的服务必须明确分类为待迁移业务依赖、允许保留的 shared kernel 依赖或临时兼容债务；重命名后不得残留旧 module/import path。

## 8. Observability and Debug Requirements

- ODR-001: 每个服务迁移后必须保留或改善当前 metrics registry、trace runtime、health server、structured logging 和 request/trace correlation。
- ODR-002: 每个服务的 application command/query 和异步 consumer 必须具备足够日志上下文，包括 service name、operation、request id/trace id、aggregate id/event id、错误原因和重试状态。
- ODR-003: Outbox/Inbox/Worker 相关迁移必须记录 pending、retry、failed、DLQ、consume latency、publish latency、idempotency hit 等可诊断信息。
- ODR-004: Gateway 到服务的 gRPC 调用失败必须保持当前可诊断性，不能因 interfaces 层迁移吞掉错误。
- ODR-005: TASK 报告必须记录实际执行的验证命令、失败命令、跳过原因和剩余风险。

## 9. Error Handling and Fallback Requirements

- EFR-001: 迁移不得吞掉已有错误；跨层错误必须保持上下文，并转换为当前兼容的 protobuf/HTTP response。
- EFR-002: 任何 owner service API 调用失败时，application 层必须明确区分硬失败、可重试失败、软依赖降级和幂等重复。
- EFR-003: 跨服务事件消费必须幂等；重复事件、乱序事件、缺失依赖和 poison message 必须有可诊断处理路径。
- EFR-004: AI/Embedding/SMTP/OSS/RabbitMQ 等外部依赖的现有 fallback 或不可用语义不得因代码迁移丢失。
- EFR-005: 如迁移需要临时 adapter，adapter 必须清楚标注生命周期、owner、移除条件和测试覆盖。

## 10. Security and Safety Requirements

- SSR-001: 迁移不得提交真实 `config.yaml`、`.env`、Token、密钥、证书、私钥或云服务凭据。
- SSR-002: 认证、授权、RBAC、data scope、JWT、Refresh Token、内部 gRPC 鉴权和 TLS 相关变更必须作为 Hard Stop，除非对应 TASK 已明确授权。
- SSR-003: 每个服务迁移后必须保持最小权限原则；非 owner 服务不得新增非授权写表能力。
- SSR-004: domain 层不得引入会泄露敏感信息的日志或错误消息。
- SSR-005: AI Agent、MCP、Embedding 迁移必须保留现有私网访问限制、命令 allowlist、provider credential 加密和审计语义。
- SSR-006: Worker 不得成为绕过业务授权和领域规则的后门；后台写入必须通过 owner context 的 application/domain contract 或已授权 repository port。

## 11. Acceptance Criteria

- AC-001: `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md` 和 `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md` 存在，内容为中文，并符合 `spec-harness` 的 `draft-spec-sdd` 章节结构要求。
- AC-002: SPEC 和 SDD 明确记录迁移顺序：`Offer -> Interview -> Notification -> Identity -> Recruitment -> Analytics -> AI Agent -> Worker`。
- AC-003: SPEC 和 SDD 明确当前架构事实：已拆出独立服务根，但核心业务 model/repository/service 仍集中在 `smart-recruit-commons`。
- AC-004: SPEC 和 SDD 明确目标架构：每个微服务拥有本地 `domain`、`application`、`infrastructure`、`interfaces`、`runtime` 分层。
- AC-005: SPEC 和 SDD 明确 `smart-recruit-commons` 的收缩方向和允许保留的 shared kernel 范围。
- AC-006: SPEC 和 SDD 明确兼容策略：HTTP、protobuf、Gateway route mode、本地启动、部署配置、表归属、Outbox/Inbox、认证授权均默认保持兼容。
- AC-007: SPEC 和 SDD 明确 Hard Stop 类变更：public API、protobuf、schema/migration、auth/authz/security、package/lockfile、全局配置、新依赖、跨服务写表等。
- AC-008: SPEC 和 SDD 明确后续 TASK 必须按 `spec-harness` 与 `harness-pipeline` 规则串行执行、记录报告和 evidence。
- AC-009: `draft-spec-sdd` 模式不得创建 `TASKS.md`、`AGENT_RULES.md`、`task-scope.json`、`acceptance/`、`prompts/`、`scripts/` 或 `reports/`。
- AC-010: 本阶段不得修改业务代码、package manifests、lockfiles、shared modules、proto、schema 或部署配置。
- AC-011: SPEC 和 SDD 明确 Open Questions 已按用户确认的推荐方案固化，不再保留未决架构问题。
- AC-012: SPEC 和 SDD 明确最终重命名策略：`smart-recruit-commons` 在 shared kernel 收敛后作为本功能点最后阶段重命名为 `smart-recruit-commons`。

## 12. Out of Scope

- 生成 TASKS 与 Harness 文件。
- 实施任何服务迁移。
- 在 `draft-spec-sdd` 阶段删除、移动或重命名 `smart-recruit-commons` 业务代码。
- 修改 `go.mod`、`go.sum`、`go.work`、`pnpm-lock.yaml`、`package.json`。
- 修改 protobuf、数据库迁移、`db.sql` 或表归属 manifest。
- 修改 Gateway 行为、前端行为、部署配置、Nacos 配置或 Docker/K8s 配置。
- 执行性能压测、线上迁移、密钥轮换或生产切流。

## 13. Assumptions Requiring Confirmation

以下决策已由用户在本次修订中确认，后续 `prepare-harness` 应将其作为 TASK 拆分依据；若要变更，必须修订 SPEC/SDD。

- ARC-001: 本次新功能点名称使用 `microservice-ddd-evolution`；历史 `.spec/backend-ddd-microservices-evolution` 仅作为参考，不作为本次可执行契约。
- ARC-002: 后续 `prepare-harness` 会基于本 SPEC/SDD 生成大量 TASK，并允许 TASK 数量随服务复杂度增加。
- ARC-003: 迁移期间允许保留单 MySQL 实例，只强化逻辑表归属和访问边界，不立即拆库。
- ARC-004: `smart-recruit-commons` 可以阶段性继续被多个服务依赖，但每个服务完成迁移后必须减少或消除对其中业务 `model/repository/service` 的直接依赖。
- ARC-005: 如果服务迁移发现必须修改 protobuf、schema、auth/authz 或部署配置，应在对应 TASK 中停下并请求确认，而不是在 pipeline 中静默推进。
- ARC-006: 用户接受先以 Offer 做端到端范式试点，再复制范式到后续服务。
- ARC-007: `smart-recruit-commons` 最终保留 shared kernel，不保留具体业务 `model/repository/service`。
- ARC-008: migrations 短期统一保留，长期可迁为独立 migration/platform 能力；本功能点第一阶段不拆库。
- ARC-009: protobuf 短期保持单文件和现有契约；按服务拆分 proto 应另起 feature 或 SPEC 修订。
- ARC-010: Analytics 目标为事件投影，短期只读跨表作为受控过渡债务。
- ARC-011: Worker 短期保持单服务，通过 workload profile/toggle 拆分；是否拆成多个 binary 后置确认。
- ARC-012: 架构边界检查采用根脚本与服务本地 architecture tests 双层机制。
- ARC-013: 每个服务拥有自己的 domain model，不直接复用 proto 或 GORM model 作为 domain model。
- ARC-014: 错误处理采用 shared lightweight error contract 加服务本地 typed errors。
- ARC-015: `ai`、`oss`、`email`、`mq` 按通用技术能力与业务编排语义拆分归属。
- ARC-016: 本功能点最后必须执行 `smart-recruit-commons -> smart-recruit-commons` 的重命名 TASK，且该 TASK 只能在业务代码迁出、shared kernel 范围收敛、旧 import path 可清零后开始。

## 14. Open Questions

截至本次修订，无待确认 Open Questions。原 Open Questions 已按用户确认的推荐方案固化到 `Assumptions Requiring Confirmation` 中。
