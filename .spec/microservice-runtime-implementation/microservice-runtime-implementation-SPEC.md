# 微服务运行时全量落地 SPEC

## 1. 背景

当前 `dev` 分支仍是 `web-gin-service` + `logic-grpc-service` 的集中式后端运行时。虽然历史改造已经有 DDD 包边界和多个 `cmd/<service>` skeleton，但这些还没有形成真正的独立服务仓库、独立启动、Nacos 注册发现、Nacos Config 动态配置、Gateway 按服务发现路由、Docker Compose 全量运行时和最终 monolith 退场。

本 feature 是 implementation 级别的全量落地合同，不是路线图文档。Goal 模式执行完成后，仓库中必须出现真实代码、配置、Compose、脚本和测试变更，而不是只生成下游 `.spec` 文档。

为了让当前 Git 分支可以完整追踪全量落地工作，本 feature 采用“本仓库作为拆分工作台”的方式：先在当前仓库根目录生成各个 `smart-recruit-*` 独立源码根，每个源码根拥有独立 `go.mod`、启动入口、Dockerfile、配置和 README；最后提供 export 脚本把这些源码根导出为真正独立 Git repo。

数据库策略保持用户已确认的约束：本 feature 内继续使用一个共享 MySQL 实例，不拆库、不拆 schema、不拆物理实例。

## 2. 目标

- 全量落地独立源码根：
  - `smart-recruit-proto`
  - `smart-recruit-platform-go`
  - `smart-recruit-gateway`
  - `smart-recruit-identity-service`
  - `smart-recruit-recruitment-service`
  - `smart-recruit-interview-service`
  - `smart-recruit-offer-service`
  - `smart-recruit-notification-service`
  - `smart-recruit-ai-agent-service`
  - `smart-recruit-analytics-service`
  - `smart-recruit-worker-service`
  - `smart-recruit-deploy`
- 每个服务源码根可独立构建、独立启动、独立暴露 gRPC 端口或 HTTP 端口。
- Gateway 作为唯一公网入口，通过 gRPC 调用后端服务。
- 服务注册发现使用 Nacos；动态配置使用 Nacos Config。
- RabbitMQ 保留为异步事件、Outbox、worker 任务基础。
- Redis 共享但必须按服务 key prefix 隔离。
- MySQL 保持单实例共享，但必须建立表归属约束和检查。
- 统一接入 OpenTelemetry、Prometheus + Grafana、Jaeger/Tempo、Loki/ELK 或 Loki 优先实现。
- Docker Compose 能启动完整本地微服务运行时。
- 保留 monolith fallback 到最终退场门禁；每个服务切流必须可回滚。

## 3. 非目标

- 不拆分 MySQL 实例、schema 或物理数据库。
- 不要求在本 feature 内创建远程 GitHub/GitLab repo；只生成可导出的独立源码根和 export 脚本。
- 不改变前端产品行为。
- 不重设计公开 HTTP API。
- 不引入分布式事务。
- 不删除旧 `logic-grpc-service` 代码，直到 monolith retirement TASK 验证通过。

## 4. 用户可见行为

- HR、候选人、面试官的 HTTP API 行为保持兼容。
- 前端仍只访问 Gateway。
- 服务切流后响应结构、错误码、权限语义、分页、SSE 行为应保持兼容。
- 任一服务切流失败时，Gateway 可回滚到 monolith fallback 或上一稳定目标。

## 5. 功能需求

- FR-001：创建 `smart-recruit-proto`，集中存放 protobuf 源和 Go 生成策略。
- FR-002：创建 `smart-recruit-platform-go`，提供 Nacos、Config、gRPC server/client、内部鉴权、metadata、logging、metrics、trace、health、Redis prefix、RabbitMQ wrapper 等共享能力。
- FR-003：创建 `smart-recruit-gateway`，从现有 `web-gin-service` 演进为独立 gateway 源码根，支持 Nacos discovery、Nacos Config、静态 fallback、服务级 route mode。
- FR-004：创建所有业务服务独立源码根，每个服务拥有独立 `cmd/<service>/main.go`、`go.mod`、Dockerfile、config 和 runtime。
- FR-005：每个服务启动时注册到 Nacos，关闭时注销。
- FR-006：Gateway 和服务使用 Nacos discovery 解析目标服务，保留静态地址 fallback。
- FR-007：每个服务暴露 liveness/readiness 和 Prometheus metrics。
- FR-008：服务间同步调用使用 gRPC；异步副作用使用 RabbitMQ。
- FR-009：每个服务只能写自己拥有的表；跨服务读写必须通过 gRPC 或事件，过渡例外需记录。
- FR-010：Redis key 必须统一服务前缀。
- FR-011：Docker Compose 启动 Nacos、MySQL、Redis、RabbitMQ、Gateway、全部服务、Prometheus、Grafana、trace 后端、log 后端。
- FR-012：提供 export 脚本，把 `smart-recruit-*` 源码根导出为独立本地 Git repo。
- FR-013：提供 smoke test，验证 Compose 全量启动、Nacos 注册、Gateway 路由、Offer/Identity/Recruitment 等核心 gRPC 路径和 fallback。
- FR-014：提供 monolith retirement gate，确认迁移域不再依赖 `logic-grpc-service` 后才能关闭 fallback。

## 6. 非功能需求

- NFR-001：每个 TASK 必须真实修改代码、配置、部署、脚本、测试或运行时证据；纯文档 TASK 只能作为少量门禁或报告，不得替代实现。
- NFR-002：每个 TASK 必须可独立 scope check、agent-check、自审和提交。
- NFR-003：每个服务必须可独立构建 Docker image。
- NFR-004：gRPC 写请求默认不重试；幂等读请求可受控重试。
- NFR-005：日志不得包含 secrets、token、原始凭据和不必要个人信息。
- NFR-006：metrics label 必须低基数。
- NFR-007：trace id/request id 必须贯穿 Gateway、gRPC、RabbitMQ 和服务日志。
- NFR-008：Compose profiles 应允许只启动 infra、启动兼容 monolith、启动全部微服务。

## 7. 兼容需求

- CR-001：现有前端无需改 API base path。
- CR-002：现有 protobuf 兼容，除非 TASK 明确允许同步更新 proto 和 generated code。
- CR-003：旧 `web-gin-service`、`logic-grpc-service` 在迁移期保留。
- CR-004：旧 `docker/docker-compose.yml` 可以演进或新增 compose 文件，但必须保留本地可运行路径。

## 8. 可观测性需求

- ODR-001：Gateway 和服务必须输出结构化 zap 日志。
- ODR-002：Gateway 和服务必须暴露 Prometheus metrics。
- ODR-003：Gateway、gRPC server/client、RabbitMQ publish/consume 必须传播 trace context。
- ODR-004：Compose 必须包含 Prometheus、Grafana、Jaeger 或 Tempo、Loki 或 ELK。

## 9. 错误处理和回退需求

- EHF-001：Nacos 不可用时，本地开发允许显式静态 fallback；非本地环境默认 fail-fast。
- EHF-002：Nacos Config 不可用时，服务默认 fail-fast，除非 TASK 实现已验证缓存。
- EHF-003：每个 Gateway route mode 都必须可回滚。
- EHF-004：RabbitMQ 不可用时，请求服务不得伪造事件成功；可使用 outbox 积压策略。
- EHF-005：Redis 不可用时可降级缓存，但不能绕过鉴权和权限。

## 10. 安全需求

- SSR-001：内部 gRPC 必须携带 `x-internal-token` 或等价内部鉴权 metadata。
- SSR-002：Nacos 非本地环境必须启用账号密码。
- SSR-003：secrets 不得写入 Git 或 Nacos 普通配置。
- SSR-004：Identity 服务拥有 auth、RBAC、principal、token lifecycle 和 audit。

## 11. 验收标准

- AC-001：所有 `smart-recruit-*` 独立源码根存在，并具备独立构建入口。
- AC-002：Gateway 可通过 Nacos discovery 调用各服务，并支持静态 fallback。
- AC-003：所有服务可注册到 Nacos。
- AC-004：Docker Compose 可启动完整微服务运行时。
- AC-005：共享 MySQL 单实例策略保持，表归属检查可运行。
- AC-006：Redis prefix 检查可运行。
- AC-007：RabbitMQ event/outbox/worker 路径可验证。
- AC-008：Prometheus/Grafana/trace/log 组件可启动并采集至少 Gateway 与一个服务。
- AC-009：核心域服务完成 Gateway 切流和 rollback 证据：Identity、Recruitment、Interview、Offer、Notification、AI Agent、Analytics。
- AC-010：Worker 独立运行并可消费受控任务。
- AC-011：export 脚本可导出独立本地 Git repo。
- AC-012：最终 pipeline-state completed 前，所有 TASK evidence 通过 validator。

## 12. 范围外

- 远程仓库创建。
- 生产 Kubernetes 部署。
- 数据库拆分。
- 前端产品改版。

## 13. 需要确认的假设

- A-001：当前仓库作为拆分工作台，`smart-recruit-*` 目录代表独立 repo 源码根。
- A-002：Compose 第一版 trace 后端优先使用 Jaeger；日志后端优先使用 Loki。
- A-003：服务抽取时允许先复用当前 protobuf package name，后续再做 package rename。
- A-004：旧 monolith 保留为 fallback，直到最终 retirement gate。

## 14. 开放问题

- OQ-001：导出的远程 repo 最终放在哪个 Git host 和组织？
- OQ-002：是否要求所有服务初版都拥有独立数据库迁移执行器，还是先由 deploy repo 统一执行 migration？
- OQ-003：生产是否要求 mTLS，还是先使用 token + 内网隔离？
