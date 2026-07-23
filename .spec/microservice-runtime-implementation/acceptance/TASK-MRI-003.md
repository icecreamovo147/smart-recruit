# Acceptance - TASK-MRI-003

## 验收标准

- `smart-recruit-platform-go` 有独立 Go module。
- 提供 config、service metadata、gRPC server/client 基础接口。
- 单元测试覆盖基础配置和错误处理。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-003`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
