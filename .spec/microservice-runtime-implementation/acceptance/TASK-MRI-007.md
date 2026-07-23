# Acceptance - TASK-MRI-007

## 验收标准

- `smart-recruit-gateway` 可独立构建。
- Gateway 保持现有 HTTP 行为兼容。
- Gateway 使用统一 proto/platform 基础。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-007`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
