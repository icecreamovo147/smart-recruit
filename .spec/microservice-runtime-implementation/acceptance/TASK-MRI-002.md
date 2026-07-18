# Acceptance - TASK-MRI-002

## 验收标准

- `smart-recruit-proto` 成为 proto 源和生成策略的独立源码根。
- 提供 proto sync/generation 脚本或说明，并验证现有 generated code 不漂移。
- Gateway 与服务可引用统一 proto 产物。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-002`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
