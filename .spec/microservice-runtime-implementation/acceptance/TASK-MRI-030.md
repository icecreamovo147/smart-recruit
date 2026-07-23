# Acceptance - TASK-MRI-030

## 验收标准

- export 脚本可把每个 `smart-recruit-*` 源码根导出为独立本地 Git repo。
- manifest 记录 repo 名、源目录、默认分支、构建命令。
- export dry-run 可运行。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-030`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
