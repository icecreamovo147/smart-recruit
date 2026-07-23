# Microservice DDD Evolution Guidelines

## 1. 适用范围

本文是 `.spec/microservice-ddd-evolution` 的执行型架构总则，适用于后端微服务从“独立运行时 + 共享业务内核”演进为“服务自治微服务 + DDD 分层”的所有 TASK。

当前后端已经拆出以下 Go 服务根：

- `smart-recruit-offer-service`
- `smart-recruit-interview-service`
- `smart-recruit-notification-service`
- `smart-recruit-identity-service`
- `smart-recruit-recruitment-service`
- `smart-recruit-analytics-service`
- `smart-recruit-ai-agent-service`
- `smart-recruit-worker-service`

迁移期间，HTTP Gateway、protobuf、数据库 schema、部署入口、认证授权语义和前端行为默认保持兼容。任何公共契约或安全语义变化都必须作为 Hard Stop 处理。

## 2. 固定迁移顺序

服务迁移顺序固定为：

```text
Offer -> Interview -> Notification -> Identity -> Recruitment -> Analytics -> AI Agent -> Worker -> Final Commons Rename
```

执行规则：

- 不得在当前服务迁移完成前开始后续服务迁移。
- Offer 是首个 DDD 试点，后续服务复用其分层范式和报告格式。
- Identity、AI Agent、最终 commons rename 属于高风险阶段；在本 pipeline 中按用户授权继续执行，但报告必须记录原 `requiresHumanConfirmation` 状态和授权来源。
- TASK-030 之前不得重命名 `smart-recruit-commons`。

## 3. 目标 DDD 分层

每个业务服务最终应在本服务 `internal/` 下表达以下职责：

```text
internal/
  domain/
  application/
  infrastructure/
  interfaces/
  runtime/
```

### domain

`domain` 层承载实体、值对象、聚合、状态机、领域服务、领域事件和仓储接口。它不得依赖外层技术，包括 GORM、Redis、RabbitMQ、gRPC、HTTP、Nacos、OSS、SMTP、AI provider SDK、环境变量、protobuf、共享 repository 或共享业务 service。

允许依赖 Go 标准库中的纯领域工具，如 `context`、`errors`、`fmt`、`time`、`strings`。如果需要跨上下文信息，必须通过 application port 或领域事件表达，而不是直接读取外部表或调用基础设施。

### application

`application` 层承载 use case 编排、命令/查询处理、事务边界、权限策略调用、跨聚合协调、跨上下文 port、事件发布时机和 DTO。它可以依赖本服务 `domain`，但不得直接依赖 GORM repository、Redis、RabbitMQ、gRPC server、HTTP handler 或本服务 `infrastructure` 实现。

### infrastructure

`infrastructure` 层实现 repository port、事务适配、Redis/MQ/OSS/SMTP/AI/gRPC client、缓存、外部系统访问和持久化细节。业务状态机不得下沉到该层。

### interfaces

`interfaces` 层承载 gRPC server、event handler、proto request/response 映射和传输层校验。它必须调用 application 层，不得直接操作 GORM repository 或 persistence adapter。

### runtime

`runtime` 层负责配置加载、DB/Redis/RabbitMQ/Nacos/health/metrics/trace/logger wiring、依赖注入、gRPC 注册和 graceful shutdown。运行时装配可以依赖 application、infrastructure 和 interfaces，但不得承载核心业务规则。

## 4. Shared Kernel 和 legacy bridge

`smart-recruit-commons` 在迁移期间是 legacy bridge 与 shared kernel 候选，不得新增具体业务上下文代码。

长期允许保留的内容：

- 通用分页、轻量错误契约、稳定 authz 常量、通用加密 helper。
- 通用 event envelope、Outbox/Inbox 基础元数据或 helper，前提是不包含具体业务状态机。
- 通用 test helper 或 migration runner，前提是 owner 和移除条件明确。
- 临时 legacy adapter，必须记录 owner、移除条件和后续 TASK。

长期不应保留的内容：

- Offer、Interview、Application、Notification、Auth、AI 等具体业务 service。
- 服务 owner 的 GORM repository 实现。
- 服务 owner 的业务实体、状态机和跨上下文聚合服务。
- 需要访问具体业务表的共享业务编排。

所有业务迁出、shared kernel 范围确认和 Worker owner contract 收敛后，才允许执行最终 rename：`smart-recruit-commons -> smart-recruit-commons`。

## 5. 数据和跨上下文边界

- 当前单 MySQL 实例作为过渡运行事实保留。
- 表写入必须遵守 `smart-recruit-deploy/mysql-table-ownership.json` 的 owner 规则。
- 新增直接跨服务写表是 Hard Stop。
- 跨上下文写协作应通过 owner service API、domain event、Outbox/Inbox、幂等 consumer 或明确 Saga/补偿流程完成。
- 跨上下文读协作优先通过 owner service query API、只读 projection、快照表或明确兼容 adapter 完成。
- 新增 schema、migration、`db.sql` 或表归属语义变化必须单独确认。

## 6. Hard Stop 清单

遇到以下情况必须暂停当前 TASK 并记录阻塞原因：

- 修改 protobuf 或 public HTTP/gRPC 行为。
- 修改数据库 schema、migration SQL、`db.sql` 或表归属语义。
- 修改认证、授权、RBAC、data scope、JWT、Refresh Token、内部 gRPC 鉴权或 TLS 安全行为。
- 修改 package manifest、lockfile、CI/CD 或全局配置。
- 引入、删除或升级依赖。
- 需要修改当前 TASK scope 之外的文件。
- 需要新增直接跨服务写表。
- 需要删除现有测试。
- 无法建立可靠 TASK baseline。
- scope check 无法判断真实变更范围。
- SPEC、SDD、TASKS、acceptance 互相冲突。

本 pipeline 的用户指令已授权继续处理 `requiresHumanConfirmation=true` 的 TASK，但上述 Hard Stop 仍然适用于实际契约、安全、schema、依赖和 scope 风险。

## 7. 边界检查基线

`node scripts/check-backend-boundaries.mjs` 是本功能点的根级架构边界检查入口。TASK-001 后，该脚本至少覆盖：

- 禁止新后端服务引用历史 `logic-grpc-service` 或 `web-gin-service` 路径。
- 服务本地 `internal/domain` 不得导入 GORM、Redis、RabbitMQ、gRPC、HTTP、Nacos、protobuf、共享 repository、共享业务 service 或本服务外层分层。
- 服务本地 `internal/application` 不得导入 GORM、Redis、RabbitMQ、gRPC、HTTP、共享 repository 或本服务 `infrastructure` 实现。
- 服务本地 `internal/interfaces` 不得直接导入 persistence 依赖。

该检查只建立边界基线，不要求一次性修复所有历史共享依赖。后续服务 TASK 应在创建本地 DDD 分层时持续让该脚本通过。

## 8. 验证要求

每个 TASK 完成后必须运行或记录无法运行原因：

```bash
git diff --name-only
bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh <TASK-ID>
bash .spec/microservice-ddd-evolution/scripts/agent-check.sh
```

服务迁移 TASK 还必须运行对应服务目录下的：

```bash
go test ./...
```

涉及表访问的 TASK 必须运行：

```bash
node scripts/check-mysql-table-ownership.mjs
```

涉及本文件或边界规则的 TASK 必须运行：

```bash
node scripts/check-backend-boundaries.mjs
```

## 9. 报告和知识影响

每个 TASK 必须在 `.spec/microservice-ddd-evolution/reports/` 生成 Markdown 报告和 JSON evidence。报告必须真实记录：

- 修改文件列表。
- scope check 结果。
- SPEC、SDD、acceptance 对比。
- 测试命令、exit code、耗时和结果。
- skipped checks 及原因。
- knowledge impact 结果。
- 剩余风险和下一 TASK 是否可开始。

知识影响遵守 `.knowledge/README.md`。如果现有 active knowledge 与当前微服务根事实不一致，且当前 TASK 不适合直接更新 active 文档，应写入 `.knowledge/inbox/` 候选或在报告中记录 `candidate_required`。
