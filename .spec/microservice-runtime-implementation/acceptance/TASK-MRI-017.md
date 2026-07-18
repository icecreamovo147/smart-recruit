# Acceptance - TASK-MRI-017

## 验收标准

- Gateway Interview 路由可切到 Interview 服务。
- Schedule、feedback、interviewer task smoke 通过。
- rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-017`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
