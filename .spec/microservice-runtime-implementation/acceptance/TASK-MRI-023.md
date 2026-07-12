# Acceptance - TASK-MRI-023

## 验收标准

- Gateway Analytics/reporting 路由可切到 Analytics 服务。
- Dashboard/funnel/time-in-stage/interview-offer metrics smoke 通过。
- rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-023`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
