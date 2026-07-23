# Acceptance - TASK-MRI-013

## 验收标准

- Gateway recruitment 相关路由可切到 Recruitment 服务。
- Jobs、candidate profile、resume、application smoke 通过。
- rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-013`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
