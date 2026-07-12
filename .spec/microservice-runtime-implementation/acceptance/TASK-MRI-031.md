# Acceptance - TASK-MRI-031

## 验收标准

- 审计 Gateway route mode，确认迁移域不再必须依赖 monolith。
- 保留可配置 rollback，除非明确 retirement evidence 通过。
- 生成 monolith fallback retirement gate 报告。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-031`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
