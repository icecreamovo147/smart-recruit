# Acceptance - TASK-MRI-005

## 验收标准

- Platform 提供 zap logger、Prometheus metrics、OpenTelemetry trace、health/readiness helpers。
- metrics label 低基数，日志不记录 secrets。
- 至少有单元测试或 smoke 测试覆盖初始化和关闭。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-005`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
