# TASK-MRI-024 Report

## TASK ID

TASK-MRI-024

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-024-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-024-report.md`
- `smart-recruit-worker-service/README.md`
- `smart-recruit-worker-service/cmd/worker-service/main.go`
- `smart-recruit-worker-service/cmd/worker-service/main_test.go`
- `smart-recruit-worker-service/doc.go`
- `smart-recruit-worker-service/go.mod`
- `smart-recruit-worker-service/go.sum`
- `smart-recruit-worker-service/internal/runtime/runtime.go`
- `smart-recruit-worker-service/internal/runtime/runtime_test.go`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-024` 的基线、通过状态和 evidence 路径，并推进当前任务到 `TASK-MRI-025`。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-024-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-024-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-worker-service/README.md`: 更新 Worker 独立 build、check、serve 与 workload toggle 说明。
- `smart-recruit-worker-service/cmd/worker-service/main.go`: 新增独立 Worker 服务入口，接入配置、Nacos、Nacos Config、MySQL、RabbitMQ、health/readiness、metrics、trace 和日志。
- `smart-recruit-worker-service/cmd/worker-service/main_test.go`: 覆盖 discovery 名称、Nacos fallback、RabbitMQ 队列映射和 `/readyz`。
- `smart-recruit-worker-service/doc.go`: 将服务文档名更新为 `worker-service`。
- `smart-recruit-worker-service/go.mod`: 增加 Worker 独立服务模块依赖。
- `smart-recruit-worker-service/go.sum`: 锁定 Worker 独立服务模块依赖校验和。
- `smart-recruit-worker-service/internal/runtime/runtime.go`: 新增 Worker runtime、workload toggle 解析、starter 校验、幂等策略校验和 RabbitMQ/MySQL readiness。
- `smart-recruit-worker-service/internal/runtime/runtime_test.go`: 覆盖默认 workload、禁用项、启动顺序、RabbitMQ/MySQL readiness 和缺失 starter。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=b48e129250ae164d422f34cf6739c23d57a3b040 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-024
```

所有变更均匹配 `smart-recruit-worker-service/**`、`**/go.mod`、`**/go.sum` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。Worker 服务源码根现在具备独立模块、独立命令入口、Nacos discovery/config、health/readiness、metrics、trace、日志、MySQL 与 RabbitMQ readiness；未拆分 MySQL，也未新增跨服务直接写表。

## SDD 对照结果

通过。实现按 workload toggle 管理 outbox、notification、email、resume parse、embedding、agent run、analytics projection 等工作负载，且每个启用 workload 必须有 starter 和幂等策略，避免重复消费未幂等任务。

## Acceptance 对照结果

通过。`smart-recruit-worker-service` 可独立 build 与 `--check`；Worker 可按配置选择并启动受控 workload supervisor；readiness 正确反映 RabbitMQ 和 MySQL 硬依赖。

## 测试命令和结果

- `cd smart-recruit-worker-service && GOWORK=off go test ./...`: passed
- `cd smart-recruit-worker-service && GOWORK=off go build ./cmd/worker-service`: passed
- `cd smart-recruit-worker-service && GOWORK=off go run ./cmd/worker-service --check`: passed
- `git diff --name-only`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree b48e129250ae164d422f34cf6739c23d57a3b040 --json`: passed
- `TASK_BASE_TREE=b48e129250ae164d422f34cf6739c23d57a3b040 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-024`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-024-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 Worker 独立 Go module runtime、cmd、health/readiness、测试和 README。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，未修改 HTTP routes、handlers、protobuf 或生成代码。
- 是否提交 secrets 或真实 `.env`：通过，未创建或修改 env/secrets。
- 是否违反单 MySQL 约束：通过，本服务复用同一 MySQL 配置和 schema，不新增物理数据库或跨服务直接写表。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，目标测试、独立 build/check、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；本 TASK 增加 Worker 服务根和独立运行时，不改变公开 HTTP API 或默认本地启动路径。

## 风险

- 本 TASK 建立 worker runtime、toggle 和 readiness；真实 Outbox/Inbox/DLQ 消费、replay 与端到端消息处理证据在 TASK-MRI-025 落地。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-025。
