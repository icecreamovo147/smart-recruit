# Acceptance - TASK-MRI-010

## 验收标准

- `smart-recruit-identity-service` 可独立构建并启动 gRPC。
- 注册 AuthService 和 Identity-owned AdminService 子集。
- 接入 Nacos、Config、health、metrics、trace、shared MySQL。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-010`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
