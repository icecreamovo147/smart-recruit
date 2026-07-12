# Acceptance - TASK-MRI-015

## 验收标准

- Gateway Offer 路由可切到 Offer 服务。
- Offer create/send/accept/reject/withdraw/list smoke 通过。
- rollback 到 logic 可用。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-015`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
