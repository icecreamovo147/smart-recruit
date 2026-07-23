# Acceptance - TASK-MRI-012

## 验收标准

- `smart-recruit-recruitment-service` 可独立构建并启动 gRPC。
- 注册 Job、Candidate、Application 服务。
- 使用共享 MySQL，写表范围符合 Recruitment ownership。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-012`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
