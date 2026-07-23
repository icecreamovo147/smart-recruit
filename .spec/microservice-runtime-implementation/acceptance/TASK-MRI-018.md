# Acceptance - TASK-MRI-018

## 验收标准

- `smart-recruit-notification-service` 可独立构建并启动。
- 注册 NotificationService，并接入 notification persistence、email coordination、realtime delivery。
- RabbitMQ/outbox/inbox 幂等语义明确并测试。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-018`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
