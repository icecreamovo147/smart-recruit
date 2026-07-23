# Acceptance - TASK-MRI-006

## 验收标准

- `smart-recruit-deploy` 提供 Docker Compose 基础设施：Nacos、MySQL、Redis、RabbitMQ、Prometheus、Grafana、Jaeger/Tempo、Loki/ELK 或 Loki。
- 提供 Nacos seed config 示例，不包含 secrets。
- Compose profiles 支持 infra、observability、services、compat。

## 必需检查

- `git diff --name-only`
- `TASK_BASE_TREE=<base_tree> bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-006`
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`
