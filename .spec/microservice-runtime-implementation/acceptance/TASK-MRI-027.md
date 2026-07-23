# Acceptance - TASK-MRI-027

## 验收标准

- Platform Redis helper 强制服务 prefix。
- 检查脚本能发现新增无 prefix Redis key。
- Gateway 与服务缓存使用方迁移到 prefix helper 或记录例外。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-027`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
