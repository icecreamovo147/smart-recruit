# Acceptance - TASK-MRI-021

## 验收标准

- Gateway AI 相关路由可切到 AI Agent 服务。
- Chat、agent run、prompt/config、embedding config smoke 通过。
- rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-021`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
