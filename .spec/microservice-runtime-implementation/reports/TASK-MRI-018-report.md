# TASK-MRI-018 Report

## TASK ID

TASK-MRI-018

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-018-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-018-report.md`
- `smart-recruit-notification-service/README.md`
- `smart-recruit-notification-service/cmd/notification-service/main.go`
- `smart-recruit-notification-service/cmd/notification-service/main_test.go`
- `smart-recruit-notification-service/doc.go`
- `smart-recruit-notification-service/go.mod`
- `smart-recruit-notification-service/go.sum`
- `smart-recruit-notification-service/internal/runtime/runtime.go`
- `smart-recruit-notification-service/internal/runtime/runtime_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-018` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-019`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-018-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-018-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-notification-service/README.md`: 更新 Notification 服务根为可构建、可运行 gRPC runtime，并说明 persistence、realtime、outbox、inbox、email coordination。
- `smart-recruit-notification-service/cmd/notification-service/main.go`: 新增独立 Notification gRPC runtime 入口，支持 `--check` 和 `--serve`，接入共享 MySQL、可选 Redis/Nacos、RabbitMQ、health、metrics、trace、日志，并启动 NotificationRuntime。
- `smart-recruit-notification-service/cmd/notification-service/main_test.go`: 覆盖 Nacos discovery/static fallback 与 RabbitMQ queue/exchange 映射。
- `smart-recruit-notification-service/doc.go`: 将模块服务名对齐为 `notification-service`。
- `smart-recruit-notification-service/go.mod`: 增加 Notification runtime 所需 Go 依赖、本地 `logic-grpc-service` 和 `smart-recruit-platform-go` replace。
- `smart-recruit-notification-service/go.sum`: 记录 Notification 模块依赖校验和。
- `smart-recruit-notification-service/internal/runtime/runtime.go`: 新增 Notification runtime bridge，注册 `NotificationService`，验证 persistence/realtime/outbox/inbox/email 组件，并记录幂等语义。
- `smart-recruit-notification-service/internal/runtime/runtime_test.go`: 覆盖 gRPC 注册、组件完整性和 RabbitMQ/outbox/inbox/realtime 幂等语义说明。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=daf0a47fec221b77dd278f9dcaec78a9af31871b bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-018
```

所有变更均匹配 `smart-recruit-notification-service/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Notification 服务已具备独立源码根、独立构建入口、独立 gRPC 启动路径，并接入 Nacos、health、metrics、trace、日志、RabbitMQ/outbox/inbox runtime。

## SDD 对照结果

通过。实现沿用现有 `NotificationService` 和 `NotificationRuntime`，共享 MySQL schema，RabbitMQ/outbox/inbox 语义保持现有幂等模型，不改变 HTTP/protobuf contract。

## Acceptance 对照结果

通过。`smart-recruit-notification-service` 可独立 `go test`、`go build` 和 `go run --check`；runtime 注册 `NotificationService`；组件和测试覆盖 notification persistence、email coordination、realtime delivery、RabbitMQ/outbox/inbox 幂等语义。

## 测试命令和结果

- `cd smart-recruit-notification-service && GOWORK=off go test ./...`: passed
- `cd smart-recruit-notification-service && GOWORK=off go build ./cmd/notification-service`: passed
- `cd smart-recruit-notification-service && GOWORK=off go run ./cmd/notification-service --check`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree daf0a47fec221b77dd278f9dcaec78a9af31871b --json`: passed
- `TASK_BASE_TREE=daf0a47fec221b77dd278f9dcaec78a9af31871b bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-018`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-018-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增独立 Notification service runtime、cmd、测试、模块依赖和运行验证。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP route、protobuf 或生成代码。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，服务复用共享 MySQL 配置和 schema，未拆分数据库。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 增加独立 Notification runtime，不改变公开 HTTP API、默认本地启动路径或单 MySQL 约束。

## 风险

- `--serve` 需要本地共享 MySQL、RabbitMQ 和可选 Redis/Nacos 配置；本 TASK 已验证构建和 runtime wiring，完整 broker/live delivery smoke 由后续 Docker Compose/runtime TASK 覆盖。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-019。
