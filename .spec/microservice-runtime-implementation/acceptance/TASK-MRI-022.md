# Acceptance - TASK-MRI-022

## 验收标准

- `smart-recruit-analytics-service` 可独立构建并启动。
- Reporting APIs 使用 Analytics-owned projection/read model。
- 不写 transactional domain state。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-022`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
