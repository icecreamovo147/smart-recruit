# AGENT_RULES - microservice-ddd-evolution

## 1. Canonical Inputs

每个 TASK 必须读取：

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SPEC.md`
- `.spec/microservice-ddd-evolution/microservice-ddd-evolution-SDD.md`
- `.spec/microservice-ddd-evolution/TASKS.md`
- `.spec/microservice-ddd-evolution/task-scope.json`
- 对应 `.spec/microservice-ddd-evolution/acceptance/<TASK-ID>.md`
- `.knowledge/README.md`，并按 AGENTS 规则选择相关知识文档

## 2. Global Migration Rules

- 必须严格按 TASK 顺序执行。
- 服务迁移顺序固定为 `Offer -> Interview -> Notification -> Identity -> Recruitment -> Analytics -> AI Agent -> Worker -> Final Commons Rename`。
- 不得在当前服务完成前开始后续服务迁移。
- 不得把迁移简化为机械复制文件；每个服务必须建立 `domain/application/infrastructure/interfaces/runtime` 职责边界。
- `domain` 层不得依赖 GORM、Redis、RabbitMQ、gRPC、HTTP、Nacos、OSS、SMTP、AI provider SDK 或环境变量。
- `interfaces` 层不得直接操作 GORM repository。
- `infrastructure` 实现 repository/client/event ports，但业务状态机必须留在 domain/application。
- Gateway 仍是 transport/policy boundary，不得承载核心业务状态机和 persistence 规则。

## 3. Shared Module Rules

- `smart-recruit-domain-go` 在迁移期间是 legacy bridge/shared kernel 候选，不得新增具体业务上下文代码。
- 服务完成迁移后，必须减少或消除对 `smart-recruit-domain-go/model`、`repository`、`service` 的依赖。
- `smart-recruit-domain-go` 最终只保留 shared kernel 和通用技术能力。
- 最终重命名为 `smart-recruit-commons` 只能在 TASK-030 执行。

## 4. Hard Stop Conditions

遇到以下情况必须停止并请求确认：

- 修改 protobuf 或 public HTTP/gRPC 行为。
- 修改数据库 schema、migration SQL、`db.sql` 或表归属语义。
- 修改认证、授权、RBAC、data scope、JWT、Refresh Token、内部 gRPC 鉴权或 TLS 安全行为。
- 修改 package manifest、lockfile、CI/CD 或全局配置。
- 引入、删除、升级依赖。
- 需要跨越当前 TASK scope。
- 需要新增直接跨服务写表。
- 需要删除现有测试。
- 无法建立可靠 TASK baseline 或 scope check 无法判断。

## 5. Testing Rules

每个实现 TASK 完成后必须运行或说明无法运行：

```bash
git diff --name-only
bash .spec/microservice-ddd-evolution/scripts/check-task-scope.sh <TASK-ID>
bash .spec/microservice-ddd-evolution/scripts/agent-check.sh
```

服务 TASK 至少运行对应服务目录下的：

```bash
go test ./...
```

若修改 `smart-recruit-domain-go`、`smart-recruit-platform-go`、`smart-recruit-proto`、gateway 或最终 rename，必须运行所有受影响 Go module 的 `go test ./...`，或在报告中记录不可运行原因。

涉及表访问的 TASK 必须运行：

```bash
node scripts/check-mysql-table-ownership.mjs
```

## 6. Knowledge Impact Rules

本功能点属于非平凡架构迁移。每个实施 TASK 必须执行知识影响检查：

- 读取 `.knowledge/README.md`
- 使用 `.knowledge/manifest.yaml` 和 `.knowledge/INDEX.md` 选择相关 active knowledge
- 若 `.knowledge/scripts/detect-impact.mjs` 可用且 baseline 可靠，则运行检测
- 在 TASK report 和 evidence 中记录 `knowledge_impact`

## 7. Report Rules

每个实施 TASK 必须生成：

- `.spec/microservice-ddd-evolution/reports/<TASK-ID>-report.md`
- `.spec/microservice-ddd-evolution/reports/<TASK-ID>-evidence.json`

报告必须真实记录：

- 修改文件列表
- scope check 结果
- SPEC/SDD/acceptance 对比
- 测试命令、exit code、耗时和结果
- skipped checks 及原因
- 风险和下一 TASK 是否可开始

不得把失败命令、失败 review、缺失 evidence 或越界修改描述为完成。

## 8. Compatibility Rules

- 默认保持 HTTP API、protobuf、Gateway route mode、本地启动、Docker Compose、K8s 示例、Nacos seed config、Outbox/Inbox、auth/authz 行为兼容。
- 如必须改变兼容行为，当前 TASK 必须停止并请求用户确认。
- 真实 `config.yaml`、`.env`、Token、云密钥、SMTP 密码、AI API Key 不得进入 git。

## 9. Final Rename Rules

TASK-030 之前不得重命名 `smart-recruit-domain-go`。

TASK-030 必须满足：

- 所有服务迁移完成。
- `smart-recruit-domain-go` 已收缩为 commons-ready shared kernel。
- 旧业务 `model/repository/service` 不再作为 active shared business code 存在。
- 旧 import path 可清零。
- 用户已确认执行高风险横切 rename。
