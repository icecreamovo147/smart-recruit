# TASK-MRI-025 Report

## TASK ID

TASK-MRI-025

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-025-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-025-report.md`
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`
- `logic-grpc-service/internal/platform/events/boundary.go`
- `logic-grpc-service/internal/platform/events/boundary_test.go`
- `logic-grpc-service/mq/envelope.go`
- `logic-grpc-service/mq/envelope_test.go`
- `scripts/check-idempotent-consumers.mjs`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-025` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-026`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-025-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-025-report.md`: 记录本 TASK 的人工可读验收报告。
- `.spec/microservice-runtime-implementation/scripts/agent-check.sh`: 将 logic-grpc-service 测试改为 `GOWORK=off go test ./...`，避免根 workspace 未包含该 module 时误失败。
- `logic-grpc-service/internal/platform/events/boundary.go`: 新增默认 consumer boundary、Inbox/retry/DLQ/replay 规则和 replay rule 校验。
- `logic-grpc-service/internal/platform/events/boundary_test.go`: 覆盖默认边界、幂等要求、unsafe boundary 拒绝和 replay 规则。
- `logic-grpc-service/mq/envelope.go`: 新增 `PublishEnvelope`、`ConsumeEnvelope`、queue retry/DLQ plan 与 replay plan。
- `logic-grpc-service/mq/envelope_test.go`: 覆盖无效 envelope 先于 handler 被拒绝、有效 envelope 投递到 handler、retry/DLQ/replay plan 完整性。
- `scripts/check-idempotent-consumers.mjs`: 新增可运行的幂等消费边界检查脚本。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=813f042dfec1bcd1a876ce1231abbde65e62f513 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-025
```

所有变更均匹配 `logic-grpc-service/internal/platform/events/**`、`logic-grpc-service/mq/**`、`scripts/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。RabbitMQ event envelope、Outbox/Inbox idempotency、retry、DLQ 和 replay 边界已经在共享 events/mq 层落地；未新增跨服务直接写表，也未改变数据库实例。

## SDD 对照结果

通过。跨服务副作用现在有显式 consumer boundary 和 envelope-aware mq API；消费者必须具备 Inbox/idempotency、retry、DLQ、replay 安全规则，且幂等检查脚本可运行。

## Acceptance 对照结果

通过。Envelope、Outbox/Inbox、DLQ、retry、replay 规则在服务边界层有代码和测试；跨服务副作用以事件边界描述优先；`node scripts/check-idempotent-consumers.mjs` 可运行并通过。

## 测试命令和结果

- `cd logic-grpc-service && GOWORK=off go test ./internal/platform/events ./mq`: passed
- `node scripts/check-idempotent-consumers.mjs`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 813f042dfec1bcd1a876ce1231abbde65e62f513 --json`: passed
- `TASK_BASE_TREE=813f042dfec1bcd1a876ce1231abbde65e62f513 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-025`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-025-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 events/mq 边界代码、测试和可运行幂等检查脚本。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes、handlers、protobuf 或生成代码。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本 TASK 未修改数据库实例、schema 或跨服务写表边界。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、幂等脚本、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/service-boundaries.md`、`.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 强化事件边界，不改变公开 HTTP API 或默认本地启动路径。

## 风险

- Live RabbitMQ integration test 仍需 `RABBITMQ_TEST_URL`；端到端 broker smoke 会在后续 compose/smoke TASK 中覆盖。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-026。
