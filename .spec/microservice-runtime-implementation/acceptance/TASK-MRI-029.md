# Acceptance - TASK-MRI-029

## 验收标准

- Compose smoke test 能启动 infra、Gateway、全部服务和 observability。
- Smoke test 验证 Nacos 注册、health、Gateway 至至少三个服务的路由。
- 失败时输出可诊断日志路径。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-029`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
