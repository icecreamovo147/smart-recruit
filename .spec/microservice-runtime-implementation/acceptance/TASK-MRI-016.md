# Acceptance - TASK-MRI-016

## 验收标准

- `smart-recruit-interview-service` 可独立构建并启动 gRPC。
- 注册 InterviewService。
- Recruitment lifecycle 协作通过 gRPC 或事件完成。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-016`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
