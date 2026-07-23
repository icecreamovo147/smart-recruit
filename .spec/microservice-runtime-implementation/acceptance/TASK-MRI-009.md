# Acceptance - TASK-MRI-009

## 验收标准

- Gateway 按服务支持 route mode：logic、identity、recruitment、interview、offer、notification、ai-agent、analytics。
- 每个 route mode 可回滚到 logic。
- Ready check 覆盖当前启用目标。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-009`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
