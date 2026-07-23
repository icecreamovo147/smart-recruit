# TASKS - microservice-runtime-implementation

## 任务总览

| TASK | 标题 | 状态 | 主要落地范围 |
|---|---|---|---|
| TASK-MRI-001 | 工作区与独立源码根脚手架 | pending | `smart-recruit-*` 目录、`go.work`、README |
| TASK-MRI-002 | Proto 独立源码根 | pending | `smart-recruit-proto/**`、proto sync/generation |
| TASK-MRI-003 | Platform runtime 基础 | pending | `smart-recruit-platform-go/**` |
| TASK-MRI-004 | Nacos discovery/config 实现 | pending | platform Nacos client、配置 provider |
| TASK-MRI-005 | 可观测性 platform 实现 | pending | logging、metrics、trace、health |
| TASK-MRI-006 | Deploy/Compose 基础设施 | pending | `smart-recruit-deploy/**`、`docker/**` |
| TASK-MRI-007 | Gateway 独立源码根 | pending | `smart-recruit-gateway/**` |
| TASK-MRI-008 | Gateway 接入 Nacos discovery/config | pending | gateway runtime/client/config |
| TASK-MRI-009 | Gateway 服务级 route mode 与 fallback | pending | gateway route target wiring |
| TASK-MRI-010 | Identity 服务源码根与 runtime | pending | `smart-recruit-identity-service/**` |
| TASK-MRI-011 | Identity Gateway 切流与验证 | pending | gateway identity route + tests |
| TASK-MRI-012 | Recruitment 服务源码根与 runtime | pending | `smart-recruit-recruitment-service/**` |
| TASK-MRI-013 | Recruitment Gateway 切流与验证 | pending | gateway recruitment route + tests |
| TASK-MRI-014 | Offer 服务源码根与 runtime | pending | `smart-recruit-offer-service/**` |
| TASK-MRI-015 | Offer Gateway 切流与验证 | pending | gateway offer route + tests |
| TASK-MRI-016 | Interview 服务源码根与 runtime | pending | `smart-recruit-interview-service/**` |
| TASK-MRI-017 | Interview Gateway 切流与验证 | pending | gateway interview route + tests |
| TASK-MRI-018 | Notification 服务源码根与 runtime | pending | `smart-recruit-notification-service/**` |
| TASK-MRI-019 | Notification Gateway 切流与验证 | pending | gateway notification route + tests |
| TASK-MRI-020 | AI Agent 服务源码根与 runtime | pending | `smart-recruit-ai-agent-service/**` |
| TASK-MRI-021 | AI Agent Gateway 切流与验证 | pending | gateway AI route + tests |
| TASK-MRI-022 | Analytics 服务源码根与 runtime | pending | `smart-recruit-analytics-service/**` |
| TASK-MRI-023 | Analytics Gateway 切流与验证 | pending | gateway analytics route + tests |
| TASK-MRI-024 | Worker 服务源码根与 runtime | pending | `smart-recruit-worker-service/**` |
| TASK-MRI-025 | RabbitMQ 事件/Outbox/Inbox 边界落地 | pending | event contracts、worker integration |
| TASK-MRI-026 | 共享 MySQL 表归属检查 | pending | ownership manifest/check script |
| TASK-MRI-027 | Redis prefix 强制检查 | pending | platform helper、cache usage checks |
| TASK-MRI-028 | 全服务 Dockerfile 与镜像构建 | pending | Dockerfiles、compose build targets |
| TASK-MRI-029 | 全量 Compose smoke test | pending | compose smoke script |
| TASK-MRI-030 | 独立 repo export 脚本 | pending | export scripts/manifests |
| TASK-MRI-031 | Monolith fallback 退场门禁 | pending | route audit、fallback retirement gate |
| TASK-MRI-032 | 最终 readiness 与 pipeline summary | pending | final evidence、summary |

## 执行规则

- 每个 TASK 必须真实落地对应代码、配置、脚本、测试或运行证据。
- 禁止用“再生成下游 spec”替代实现。
- 每个 TASK 完成后必须通过 scope check、agent-check、evidence validation 和 self-review。
- 每个 TASK 通过后可立即 commit。
- MySQL 保持单实例共享。

## TASK-MRI-001 - 工作区与独立源码根脚手架

### Goal
创建全量微服务拆分工作台和独立源码根。
### Scope
`smart-recruit-*` 目录、`go.work`、README。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
真实 `.env`、`db.sql`。
### Dependencies
无。
### Acceptance Criteria
见 `acceptance/TASK-MRI-001.md`。
### Required Tests
见 acceptance。
### Risks
源码根命名和 Go workspace 后续会影响全部 TASK。
### Notes
必须真实创建源码根，不得只写文档。

## TASK-MRI-002 - Proto 独立源码根

### Goal
建立 `smart-recruit-proto` 作为 proto 契约源。
### Scope
proto 源、生成策略、同步检查。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
真实 `.env`、未声明 API 行为变更。
### Dependencies
TASK-MRI-001。
### Acceptance Criteria
见 `acceptance/TASK-MRI-002.md`。
### Required Tests
见 acceptance。
### Risks
generated code 漂移会破坏 gateway/service 编译。
### Notes
protobuf 行为变更必须同步测试。

## TASK-MRI-003 - Platform runtime 基础

### Goal
建立 `smart-recruit-platform-go` 的基础 runtime API。
### Scope
config、metadata、gRPC server/client 基础。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
真实 `.env`、数据库拆分。
### Dependencies
TASK-MRI-001。
### Acceptance Criteria
见 `acceptance/TASK-MRI-003.md`。
### Required Tests
见 acceptance。
### Risks
platform API 过度设计会拖慢服务抽取。
### Notes
以当前服务实际需要为准。

## TASK-MRI-004 - Nacos discovery/config 实现

### Goal
在 platform 中落地 Nacos 注册发现和 Config provider。
### Scope
Nacos client、fallback、测试、依赖。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
secrets、真实 `.env`。
### Dependencies
TASK-MRI-003。
### Acceptance Criteria
见 `acceptance/TASK-MRI-004.md`。
### Required Tests
见 acceptance。
### Risks
依赖引入和本地 fallback 行为要清晰。
### Notes
非本地环境默认 fail-fast。

## TASK-MRI-005 - 可观测性 platform 实现

### Goal
落地 logging、metrics、trace、health/readiness。
### Scope
platform observability 代码和测试。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
高基数 metrics、敏感日志。
### Dependencies
TASK-MRI-003。
### Acceptance Criteria
见 `acceptance/TASK-MRI-005.md`。
### Required Tests
见 acceptance。
### Risks
可观测性必须低侵入。
### Notes
优先让 Gateway 和一个服务可接入。

## TASK-MRI-006 - Deploy/Compose 基础设施

### Goal
落地 Nacos、MySQL、Redis、RabbitMQ、observability Compose。
### Scope
`smart-recruit-deploy`、`docker`、`deploy`。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
真实 `.env`。
### Dependencies
TASK-MRI-004、TASK-MRI-005。
### Acceptance Criteria
见 `acceptance/TASK-MRI-006.md`。
### Required Tests
见 acceptance。
### Risks
Compose 过重，需要 profiles。
### Notes
配置示例不得含真实 secret。

## TASK-MRI-007 - Gateway 独立源码根

### Goal
创建可独立构建的 `smart-recruit-gateway`。
### Scope
Gateway 源码迁移、Go module、测试。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
公开 HTTP 行为未声明变更。
### Dependencies
TASK-MRI-002、TASK-MRI-003。
### Acceptance Criteria
见 `acceptance/TASK-MRI-007.md`。
### Required Tests
见 acceptance。
### Risks
Gateway 行为兼容最重要。
### Notes
保留旧 gateway 作为迁移参考。

## TASK-MRI-008 - Gateway 接入 Nacos discovery/config

### Goal
Gateway 使用 Nacos discovery/config 并保留静态 fallback。
### Scope
Gateway config/client/runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
secrets。
### Dependencies
TASK-MRI-004、TASK-MRI-007。
### Acceptance Criteria
见 `acceptance/TASK-MRI-008.md`。
### Required Tests
见 acceptance。
### Risks
服务发现失败必须可诊断。
### Notes
本地 fallback 必须显式配置。

## TASK-MRI-009 - Gateway 服务级 route mode 与 fallback

### Goal
Gateway 具备所有服务的切流和回滚能力。
### Scope
route mode、client target、readiness。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未声明 API 行为变更。
### Dependencies
TASK-MRI-008。
### Acceptance Criteria
见 `acceptance/TASK-MRI-009.md`。
### Required Tests
见 acceptance。
### Risks
route mode 错配会造成流量不可用。
### Notes
默认仍可回滚 logic。

## TASK-MRI-010 - Identity 服务源码根与 runtime

### Goal
抽取并启动 Identity 服务。
### Scope
Identity service module/runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未声明 auth 行为变更。
### Dependencies
TASK-MRI-002、TASK-MRI-004、TASK-MRI-005。
### Acceptance Criteria
见 `acceptance/TASK-MRI-010.md`。
### Required Tests
见 acceptance。
### Risks
auth/RBAC 破坏影响全局。
### Notes
必须保留兼容语义。

## TASK-MRI-011 - Identity Gateway 切流与验证

### Goal
Gateway auth/RBAC 路径切到 Identity 服务。
### Scope
Gateway route/client、smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未测试 auth 行为变更。
### Dependencies
TASK-MRI-010。
### Acceptance Criteria
见 `acceptance/TASK-MRI-011.md`。
### Required Tests
见 acceptance。
### Risks
登录和权限路径高风险。
### Notes
rollback 必须可用。

## TASK-MRI-012 - Recruitment 服务源码根与 runtime

### Goal
抽取并启动 Recruitment 服务。
### Scope
Job/Candidate/Application runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
数据库拆分。
### Dependencies
TASK-MRI-011。
### Acceptance Criteria
见 `acceptance/TASK-MRI-012.md`。
### Required Tests
见 acceptance。
### Risks
核心招聘流程影响面大。
### Notes
共享 MySQL 下只写 owning tables。

## TASK-MRI-013 - Recruitment Gateway 切流与验证

### Goal
Gateway recruitment 路由切到 Recruitment 服务。
### Scope
Gateway clients/routes、smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未声明 HTTP 行为变更。
### Dependencies
TASK-MRI-012。
### Acceptance Criteria
见 `acceptance/TASK-MRI-013.md`。
### Required Tests
见 acceptance。
### Risks
职位、简历、投递链路必须兼容。
### Notes
rollback 到 logic。

## TASK-MRI-014 - Offer 服务源码根与 runtime

### Goal
抽取并启动 Offer 服务。
### Scope
Offer module/runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
跨服务直接写表。
### Dependencies
TASK-MRI-012。
### Acceptance Criteria
见 `acceptance/TASK-MRI-014.md`。
### Required Tests
见 acceptance。
### Risks
Offer 与 application lifecycle 耦合高。
### Notes
协作必须 gRPC 或事件化。

## TASK-MRI-015 - Offer Gateway 切流与验证

### Goal
Gateway Offer 路由切到 Offer 服务。
### Scope
Gateway clients/routes、Offer smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未测试 lifecycle 行为变更。
### Dependencies
TASK-MRI-014。
### Acceptance Criteria
见 `acceptance/TASK-MRI-015.md`。
### Required Tests
见 acceptance。
### Risks
Offer 状态流转需事务一致。
### Notes
rollback 到 logic。

## TASK-MRI-016 - Interview 服务源码根与 runtime

### Goal
抽取并启动 Interview 服务。
### Scope
Interview module/runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
跨服务直接写 application 状态。
### Dependencies
TASK-MRI-012。
### Acceptance Criteria
见 `acceptance/TASK-MRI-016.md`。
### Required Tests
见 acceptance。
### Risks
面试状态与投递生命周期耦合。
### Notes
协作通过 Recruitment API 或事件。

## TASK-MRI-017 - Interview Gateway 切流与验证

### Goal
Gateway Interview 路由切到 Interview 服务。
### Scope
Gateway clients/routes、smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未声明 HTTP 行为变更。
### Dependencies
TASK-MRI-016。
### Acceptance Criteria
见 `acceptance/TASK-MRI-017.md`。
### Required Tests
见 acceptance。
### Risks
面试官工作流需兼容。
### Notes
rollback 到 logic。

## TASK-MRI-018 - Notification 服务源码根与 runtime

### Goal
抽取并启动 Notification 服务。
### Scope
Notification runtime、email/realtime、outbox。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
丢失通知事件。
### Dependencies
TASK-MRI-025 可前后协调。
### Acceptance Criteria
见 `acceptance/TASK-MRI-018.md`。
### Required Tests
见 acceptance。
### Risks
异步重复和丢失需要幂等。
### Notes
消费路径必须可诊断。

## TASK-MRI-019 - Notification Gateway 切流与验证

### Goal
Gateway Notification 路由切到 Notification 服务。
### Scope
Gateway clients/routes、notification smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
破坏 SSE/实时通知兼容。
### Dependencies
TASK-MRI-018。
### Acceptance Criteria
见 `acceptance/TASK-MRI-019.md`。
### Required Tests
见 acceptance。
### Risks
实时体验需要回归验证。
### Notes
rollback 到 logic。

## TASK-MRI-020 - AI Agent 服务源码根与 runtime

### Goal
抽取并启动 AI Agent 服务。
### Scope
AI/MCP/Skill/Prompt/Embedding runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
阻塞核心请求路径。
### Dependencies
TASK-MRI-024、TASK-MRI-025 可前后协调。
### Acceptance Criteria
见 `acceptance/TASK-MRI-020.md`。
### Required Tests
见 acceptance。
### Risks
外部 provider 和长任务复杂。
### Notes
长任务优先 worker 化。

## TASK-MRI-021 - AI Agent Gateway 切流与验证

### Goal
Gateway AI 相关路由切到 AI Agent 服务。
### Scope
Gateway clients/routes、AI smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
泄漏 AI secrets。
### Dependencies
TASK-MRI-020。
### Acceptance Criteria
见 `acceptance/TASK-MRI-021.md`。
### Required Tests
见 acceptance。
### Risks
AI 功能超时和降级需稳定。
### Notes
rollback 到 logic。

## TASK-MRI-022 - Analytics 服务源码根与 runtime

### Goal
抽取并启动 Analytics 服务。
### Scope
Analytics projection/reporting runtime。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
写 transactional domain state。
### Dependencies
TASK-MRI-025。
### Acceptance Criteria
见 `acceptance/TASK-MRI-022.md`。
### Required Tests
见 acceptance。
### Risks
投影延迟和准确性。
### Notes
使用 read model/projection。

## TASK-MRI-023 - Analytics Gateway 切流与验证

### Goal
Gateway reporting 路由切到 Analytics 服务。
### Scope
Gateway clients/routes、report smoke。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未验证报表兼容。
### Dependencies
TASK-MRI-022。
### Acceptance Criteria
见 `acceptance/TASK-MRI-023.md`。
### Required Tests
见 acceptance。
### Risks
报表口径必须一致。
### Notes
rollback 到 logic。

## TASK-MRI-024 - Worker 服务源码根与 runtime

### Goal
抽取并启动 Worker 服务。
### Scope
Worker runtime、consumer toggles、readiness。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
重复消费未幂等任务。
### Dependencies
TASK-MRI-025。
### Acceptance Criteria
见 `acceptance/TASK-MRI-024.md`。
### Required Tests
见 acceptance。
### Risks
队列消费与重试/DLQ。
### Notes
Worker 与 request 服务独立伸缩。

## TASK-MRI-025 - RabbitMQ 事件/Outbox/Inbox 边界落地

### Goal
统一事件、Outbox、Inbox、DLQ 和 replay。
### Scope
event contracts、mq wrapper、tests。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
破坏现有事件语义。
### Dependencies
TASK-MRI-003。
### Acceptance Criteria
见 `acceptance/TASK-MRI-025.md`。
### Required Tests
见 acceptance。
### Risks
事件重复、乱序、丢失。
### Notes
优先补幂等和诊断。

## TASK-MRI-026 - 共享 MySQL 表归属检查

### Goal
建立单 MySQL 下的表归属约束。
### Scope
manifest、check script、docs。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
数据库拆分。
### Dependencies
核心服务抽取 TASK。
### Acceptance Criteria
见 `acceptance/TASK-MRI-026.md`。
### Required Tests
见 acceptance。
### Risks
共享数据库隐藏耦合。
### Notes
例外必须可追踪。

## TASK-MRI-027 - Redis prefix 强制检查

### Goal
确保共享 Redis 下服务 key 隔离。
### Scope
platform helper、cache usage、check script。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
绕过 security cache。
### Dependencies
TASK-MRI-003。
### Acceptance Criteria
见 `acceptance/TASK-MRI-027.md`。
### Required Tests
见 acceptance。
### Risks
无 prefix key 会跨服务污染。
### Notes
过渡例外需登记。

## TASK-MRI-028 - 全服务 Dockerfile 与镜像构建

### Goal
为 Gateway 和全部服务落地镜像构建。
### Scope
Dockerfiles、compose build target、build script。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
镜像中包含 secrets。
### Dependencies
所有服务源码根。
### Acceptance Criteria
见 `acceptance/TASK-MRI-028.md`。
### Required Tests
见 acceptance。
### Risks
多 module 构建上下文复杂。
### Notes
优先可重复构建。

## TASK-MRI-029 - 全量 Compose smoke test

### Goal
验证完整本地微服务运行时。
### Scope
Compose smoke scripts、diagnostics。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
真实 `.env`。
### Dependencies
TASK-MRI-028。
### Acceptance Criteria
见 `acceptance/TASK-MRI-029.md`。
### Required Tests
见 acceptance。
### Risks
本地资源占用和启动顺序。
### Notes
失败日志需可诊断。

## TASK-MRI-030 - 独立 repo export 脚本

### Goal
把源码根导出为独立本地 Git repo。
### Scope
export scripts、manifest、dry-run。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
自动创建远程 repo。
### Dependencies
所有源码根。
### Acceptance Criteria
见 `acceptance/TASK-MRI-030.md`。
### Required Tests
见 acceptance。
### Risks
导出路径覆盖本地数据。
### Notes
默认 dry-run，实际导出需显式参数。

## TASK-MRI-031 - Monolith fallback 退场门禁

### Goal
确认 monolith fallback 是否可退场。
### Scope
route audit、retirement gate、docs。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
未验证就删除 fallback。
### Dependencies
所有服务切流 TASK。
### Acceptance Criteria
见 `acceptance/TASK-MRI-031.md`。
### Required Tests
见 acceptance。
### Risks
过早退场会失去回滚。
### Notes
默认保留可配置 rollback。

## TASK-MRI-032 - 最终 readiness 与 pipeline summary

### Goal
生成最终 readiness 证据和 pipeline summary。
### Scope
reports、docs、knowledge。
### Allowed Files
见 `task-scope.json`。
### Forbidden Files
伪造运行时证据。
### Dependencies
所有 TASK。
### Acceptance Criteria
见 `acceptance/TASK-MRI-032.md`。
### Required Tests
见 acceptance。
### Risks
证据不完整不能标记 completed。
### Notes
必须运行 pipeline-state validator。
