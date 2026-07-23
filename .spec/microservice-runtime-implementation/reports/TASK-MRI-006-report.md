# TASK-MRI-006 Report

## TASK ID

TASK-MRI-006

## 修改文件列表

- `.spec/microservice-runtime-implementation/pipeline-state.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-006-evidence.json`
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-006-report.md`
- `smart-recruit-deploy/README.md`
- `smart-recruit-deploy/docker-compose.microservices.yml`
- `smart-recruit-deploy/nacos/seed-config/README.md`
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`
- `smart-recruit-deploy/nacos/seed-config/services.yaml`
- `smart-recruit-deploy/observability/grafana/provisioning/datasources/datasources.yml`
- `smart-recruit-deploy/observability/loki.yml`
- `smart-recruit-deploy/observability/prometheus.yml`

## 每个文件的变更摘要

- `.spec/microservice-runtime-implementation/pipeline-state.json`: 记录 `TASK-MRI-006` 的基线与 evidence 路径。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-006-evidence.json`: 记录本 TASK 的机器可读检查、自审、scope 与知识影响证据。
- `.spec/microservice-runtime-implementation/reports/TASK-MRI-006-report.md`: 记录本 TASK 的人工可读验收报告。
- `smart-recruit-deploy/README.md`: 增加微服务 Compose 启动说明和 profile 列表。
- `smart-recruit-deploy/docker-compose.microservices.yml`: 新增 infra、observability、services、compat profiles，包含 Nacos、MySQL、Redis、RabbitMQ、Prometheus、Grafana、Jaeger、Loki 和 monolith fallback。
- `smart-recruit-deploy/nacos/seed-config/README.md`: 说明 seed config 非敏感约束。
- `smart-recruit-deploy/nacos/seed-config/gateway.yaml`: 新增 Gateway route mode 和 timeout 示例配置。
- `smart-recruit-deploy/nacos/seed-config/services.yaml`: 新增服务地址、metrics、Redis prefix 示例配置。
- `smart-recruit-deploy/observability/grafana/provisioning/datasources/datasources.yml`: 新增 Grafana Prometheus/Jaeger/Loki datasource provisioning。
- `smart-recruit-deploy/observability/loki.yml`: 新增本地 Loki 配置。
- `smart-recruit-deploy/observability/prometheus.yml`: 新增 Gateway 和服务 metrics scrape 示例。

## Scope check 结果

通过。命令：

```bash
TASK_BASE_TREE=95b71b2d76fdf3c9b76dd5ae7930df00c4f040c1 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-006
```

所有变更均匹配 `smart-recruit-deploy/**` 或 `.spec/microservice-runtime-implementation/**`。

## SPEC 对照结果

通过。新增 Compose 基础设施覆盖 Nacos、MySQL、Redis、RabbitMQ、Prometheus、Grafana、Jaeger 和 Loki；保留单 MySQL 实例；未提交真实 `.env` 或 secrets。

## SDD 对照结果

通过。部署根提供 infra/observability/compat/services profiles，支持后续服务 runtime、Dockerfile 和 smoke TASK 继续扩展。

## Acceptance 对照结果

通过。Compose profiles 存在；Nacos seed config 示例不包含 secrets；legacy `docker-compose` config 对所有 profiles 可解析。

## 测试命令和结果

- `git diff --name-only`: passed
- `docker-compose -f smart-recruit-deploy/docker-compose.microservices.yml --profile infra --profile observability --profile services --profile compat config`: passed
- `if rg -i "(password|secret|token|access[_-]?key|credential|private[_-]?key)" smart-recruit-deploy/nacos/seed-config/*.yaml; then exit 1; else echo "nacos_seed_yaml_secret_scan: PASS"; fi`: passed
- `node .knowledge/scripts/detect-impact.mjs --root . --base-tree 95b71b2d76fdf3c9b76dd5ae7930df00c4f040c1 --json`: passed
- `TASK_BASE_TREE=95b71b2d76fdf3c9b76dd5ae7930df00c4f040c1 bash .spec/microservice-runtime-implementation/scripts/check-task-scope.sh TASK-MRI-006`: passed
- `bash .spec/microservice-runtime-implementation/scripts/agent-check.sh`: passed
- `node .agents/skills/spec-harness/scripts/validate-evidence.mjs --file .spec/microservice-runtime-implementation/reports/TASK-MRI-006-evidence.json`: passed

## Self-review

- 是否真实落地，而不是只写文档：通过，已新增 Compose、observability 配置和 Nacos seed YAML。
- 是否越过 TASK scope：通过，scope check 无越界或 forbidden 文件。
- 是否破坏 HTTP/protobuf 兼容：通过，本 TASK 未修改 HTTP/protobuf 行为。
- 是否提交 secrets 或真实 `.env`：通过，未创建 `.env`；seed YAML 敏感词扫描通过。
- 是否违反单 MySQL 约束：通过，Compose 只有一个 MySQL 服务。
- 是否缺少测试、scope check、agent-check 或 evidence validation：通过，Compose config、secret scan、scope check、agent-check 和 evidence validation 均已通过并写入 evidence。

verdict: 通过

## Knowledge Impact

`update_required`。已按路由审阅 `.knowledge/architecture/system-overview.md` 和 `.knowledge/runbooks/local-development.md`，结论均为 `UNCHANGED`；现有启动路径未被替换。

## 风险

- `services` profile 目前是占位服务，真实服务镜像和 Dockerfile 在 TASK-MRI-028/029 落地。
- 本地 `docker-compose config` 会读取现有 `docker/.env` 做插值；报告不记录插值后的敏感值。

## 下一 TASK 是否可以开始

可以，在 evidence validator 通过、pipeline-state 记录完成并创建本地 commit 后进入 TASK-MRI-007。
