# Acceptance - TASK-MRI-001

## 验收标准

- 创建所有 `smart-recruit-*` 独立源码根目录。
- 创建或更新 `go.work`，能纳入新增 Go module。
- 每个源码根有 README，说明职责、启动方式和与旧 monolith 的关系。
- 没有提交 secrets、真实 `.env` 或数据库拆分。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-001`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
