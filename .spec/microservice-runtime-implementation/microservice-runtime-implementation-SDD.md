# 微服务运行时全量落地 SDD

## 1. 现有架构概述

`dev` 分支当前保留 `web-gin-service` 和 `logic-grpc-service`。Gateway 已具备多个 route-mode 字段和服务专属 gRPC client 结构，但地址来自静态环境变量。`logic-grpc-service/internal/*` 已有多个 DDD 边界，`logic-grpc-service/cmd/*` 已有服务 skeleton。Docker Compose 当前只启动 MySQL、Redis、RabbitMQ、logic、web 和前端。

## 2. 问题分析

历史 spec 完成的是文档和合同，没有真实修改运行时。本 feature 必须允许并要求修改业务代码、服务启动、Dockerfile、Compose、Go module、部署脚本和 smoke tests。否则 Goal 模式跑完仍然只会得到文档产物。

## 3. 总体设计

本仓库新增独立源码根：

```text
smart-recruit-proto/
smart-recruit-platform-go/
smart-recruit-gateway/
smart-recruit-identity-service/
smart-recruit-recruitment-service/
smart-recruit-interview-service/
smart-recruit-offer-service/
smart-recruit-notification-service/
smart-recruit-ai-agent-service/
smart-recruit-analytics-service/
smart-recruit-worker-service/
smart-recruit-deploy/
```

每个 Go 源码根都是独立 Go module。初始实现可从现有 `web-gin-service` 和 `logic-grpc-service` 迁移代码，允许保留兼容 imports，但最终 TASK 必须消除对旧 monolith runtime 的必需依赖。

## 4. 数据结构变更

不拆数据库。本 feature 可以新增表归属 manifest、检查脚本、outbox/inbox 缺口补充 migration，但任何 schema 变更必须由具体 TASK 明确允许。

## 5. API 和接口变更

公开 HTTP API 不变。内部 gRPC 可新增服务专属接口，但 protobuf 修改必须同步 `smart-recruit-proto`、Gateway client 和服务 generated code。

## 6. 工作流变化

1. 建立 proto/platform/deploy/gateway 基础。
2. 逐个抽取服务源码根。
3. 每个服务独立启动、注册 Nacos、接入 Config、暴露 health/metrics。
4. Gateway 逐个切服务 route mode。
5. Compose 全量运行。
6. export 独立 repo。
7. 验证 monolith fallback 可退场。

## 7. 配置设计

Bootstrap 环境变量：

```text
SERVICE_NAME
SERVICE_ENV
SERVICE_VERSION
NACOS_ADDR
NACOS_NAMESPACE
NACOS_GROUP
NACOS_USERNAME
NACOS_PASSWORD
MYSQL_DSN
REDIS_ADDR
RABBITMQ_URL
GRPC_INTERNAL_TOKEN
OTEL_EXPORTER_OTLP_ENDPOINT
METRICS_ADDR
```

Nacos Config 保存非敏感动态配置：端口、route mode、timeout、feature flag、Redis prefix、queue names、telemetry 开关。

## 8. 兼容策略

旧 `logic-grpc-service` 保留为 fallback。Gateway route mode 从 `logic` 切到服务名；所有切流 TASK 必须提供回滚方式。

## 9. 错误处理设计

Nacos 本地可 fallback，生产 fail-fast。gRPC 写请求不默认重试。RabbitMQ 和 Redis 降级必须不破坏核心事务和安全。

## 10. 可观测性设计

Platform module 统一封装 logger、metrics、trace、health。Compose 提供 Prometheus/Grafana/Jaeger/Loki。

## 11. 测试策略

- 每个 Go module 运行 `go test ./...`。
- Gateway route mode 和 discovery 有单元/集成测试。
- 服务 runtime 有启动、health、registration 测试。
- Compose 有 smoke script。
- 表归属、Redis prefix、secrets、proto sync 有检查脚本。

## 12. 迁移风险

- 全量拆分任务多，Goal 模式可能中途失败；pipeline-state 必须支持续跑。
- 独立 Go module 可能出现依赖漂移；每个 TASK 必须验证。
- 初期共享 MySQL 容易产生边界违规；必须用 manifest 和检查约束。

## 13. 实现边界

本 feature 明确允许真实修改：

- `smart-recruit-*/**`
- `web-gin-service/**`
- `logic-grpc-service/**`
- `docker/**`
- `deploy/**`
- `scripts/**`
- `.github/**`（仅测试/CI 需要时）
- `go.work`、`go.mod`、`go.sum`（仅对应 TASK）
- `.spec/microservice-runtime-implementation/**`

## 14. 备选方案

- 只生成文档：拒绝。
- 只做第一阶段：拒绝，本 feature 要覆盖全量落地。
- 立即远程建 repo：暂不做，先在当前分支生成可导出源码根。

## 15. 需要确认的假设

同 SPEC §13。

## 16. 开放问题

同 SPEC §14。
