# Acceptance - TASK-MRI-011

## 验收标准

- Gateway auth/RBAC/principal 路径可切到 Identity 服务。
- Login、refresh、principal、RBAC、audit smoke 通过。
- route mode rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-011`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
