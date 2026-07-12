# Acceptance - TASK-MRI-024

## 验收标准

- `smart-recruit-worker-service` 可独立构建并启动。
- Worker 可按配置启动 outbox、notification、email、resume parse、embedding、agent run、analytics projection 等消费者。
- Worker readiness 正确反映 RabbitMQ 等硬依赖。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-024`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
