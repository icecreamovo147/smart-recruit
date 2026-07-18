# Acceptance - TASK-MRI-019

## 验收标准

- Gateway Notification 路由可切到 Notification 服务。
- List/unread/SSE 或实时通知兼容路径验证通过。
- rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-019`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
