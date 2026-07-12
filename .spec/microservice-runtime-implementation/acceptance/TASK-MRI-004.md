# Acceptance - TASK-MRI-004

## 验收标准

- Platform 实现 Nacos 注册发现和 Nacos Config provider。
- 支持本地静态 fallback 和非本地 fail-fast。
- 测试覆盖 Nacos 地址解析、配置加载失败和 fallback。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-004`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
