# Acceptance - TASK-MRI-026

## 验收标准

- 表归属 manifest 覆盖所有核心表。
- 检查脚本能发现未批准跨服务写表或未声明访问。
- 明确记录数据库保持单实例。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-026`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
