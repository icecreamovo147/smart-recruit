# Acceptance - TASK-MRI-025

## 验收标准

- RabbitMQ event envelope、Outbox、Inbox、DLQ、retry、replay 规则在服务边界中落地。
- 跨服务副作用优先事件化。
- 幂等消费检查可运行。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-025`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
