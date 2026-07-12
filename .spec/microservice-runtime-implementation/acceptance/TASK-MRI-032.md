# Acceptance - TASK-MRI-032

## 验收标准

- 生成最终 pipeline summary。
- 验证所有 TASK evidence 存在并通过 validator。
- 记录真实运行时落地清单、剩余风险、已知例外和下一步远程 repo 操作。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-032`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
