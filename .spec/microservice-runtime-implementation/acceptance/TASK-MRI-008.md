# Acceptance - TASK-MRI-008

## 验收标准

- Gateway 支持 Nacos discovery 和 Nacos Config。
- 本地静态地址 fallback 可用。
- 测试覆盖 discovery 成功、失败 fallback、配置缺失错误。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-008`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
