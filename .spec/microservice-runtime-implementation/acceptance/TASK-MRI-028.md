# Acceptance - TASK-MRI-028

## 验收标准

- 每个 `smart-recruit-*` 服务有 Dockerfile 或 compose build target。
- 构建脚本能构建 Gateway 和全部服务镜像。
- 镜像不包含 secrets。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-028`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
