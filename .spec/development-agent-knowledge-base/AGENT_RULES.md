# AGENT RULES - development-agent-knowledge-base

## Feature Scope

本 feature 建立面向编码 Agent 的 Git 版本化 `.knowledge` 项目认知层，以及确定性的读取、维护、影响检测、Harness 和 CI 接入。它不实现产品运行时知识库，不修改招聘业务行为。

## Required Reading

每个 TASK 开始前必须读取：

- `.spec/development-agent-knowledge-base/development-agent-knowledge-base-SPEC.md`
- `.spec/development-agent-knowledge-base/development-agent-knowledge-base-SDD.md`
- `.spec/development-agent-knowledge-base/TASKS.md`
- `.spec/development-agent-knowledge-base/AGENT_RULES.md`
- `.spec/development-agent-knowledge-base/task-scope.json`
- `.spec/development-agent-knowledge-base/acceptance/<TASK-ID>.md`
- `.spec/development-agent-knowledge-base/prompts/implement-task.md`

TASK-001 完成后，后续 TASK 还必须读取 `.knowledge/README.md` 和由 manifest/INDEX 路由到的相关知识。

## Feature-Specific Rules

1. `.knowledge` 只能作为 canonical 控制面的下游认知层。
2. 不得复制或重新定义 `AGENTS.md`、当前 `.spec`、代码、测试或 pipeline 状态。
3. 所有知识路径使用仓库相对 POSIX 路径，禁止设备绝对路径。
4. 正式知识必须有可验证来源；历史 `.ai-guides` 只能作为线索。
5. Inbox、Archive、Stale 和 Deprecated 默认不作为实现输入。
6. Agent 不得自行批准 L3 架构、安全、权限和公共接口结论。
7. 不得为 YAML、frontmatter 或 glob 解析新增第三方依赖。
8. 生成目录和本地状态不得提交 Git。
9. 不得创建 Provider 专属知识副本。
10. 不得在普通知识正文中加入覆盖 canonical 规则的 Agent 指令。

## Scope Boundaries

- 一次只执行一个 TASK。
- 只修改 `task-scope.json` 允许的文件。
- SPEC 和 SDD 在实施模式下只读。
- `.spec/development-agent-knowledge-base/**` 的允许范围仅用于当前 TASK 的 report/evidence/Harness 运行文件，不授权修改 SPEC、SDD、TASKS 或 acceptance。
- 共享 AGENTS、spec-harness 和 CI 变更分别限制在 TASK-005、TASK-006、TASK-007。
- TASK-008 为只读审计；发现问题必须返回所属 TASK。

## Forbidden Changes

- 业务 Go、Vue、Proto、数据库或部署行为。
- Package manifests、lockfiles、Go modules。
- 认证、授权、权限或敏感数据处理。
- `pipeline-state.json` 权威语义。
- 批量迁移或重写 `.ai-guides` 和历史 `.spec`。
- 未经确认的共享控制面或 CI 变更。

## Testing Requirements

每个 TASK 后必须运行：

- `git diff --name-only`
- `bash .spec/development-agent-knowledge-base/scripts/check-task-scope.sh <TASK-ID>`
- `bash .spec/development-agent-knowledge-base/scripts/agent-check.sh`
- Acceptance 文件列出的 TASK 专项检查

所有跳过检查必须在报告和 evidence 中说明原因，不能描述为通过。

## Compatibility Rules

- 保持现有业务 API、数据库、前端和服务行为不变。
- 保持历史 feature 可读，不强制批量添加 knowledgeImpact。
- 保持可靠 Git baseline 的 scope 语义。
- 保持 `pipeline-state.json` 为运行时状态来源。
- 保持 Provider adapter 为 canonical 控制面的薄适配器。

## Logging and Debug Rules

- CLI 同时提供清晰文本输出和可选 JSON。
- 输出必须确定性排序。
- 失败必须包含文件、规则和原因。
- 不输出知识全文、Secret、Token、真实个人信息或未脱敏日志。

## Report Requirements

每个 TASK 的 Markdown 报告和 JSON evidence 必须包含：

- TASK ID 与可靠 baseline
- 修改文件及逐文件摘要
- Scope、SPEC、SDD、Acceptance 比较
- 实际命令、退出码和跳过原因
- Review 类型、轮次和 verdict
- 知识影响结果或本 TASK 为知识系统建设阶段的说明
- 风险、例外、后续项和下一 TASK 是否可开始

## Hard Stop Conditions

除 canonical spec-harness Hard Stop 外，发生以下情况必须停止：

- TASK-005/006/007 缺少明确人工确认。
- TASK-007 的 CI 平台未确认或 allowed/forbidden scope 未修订消除冲突。
- 需要新增依赖或修改 package manifest/lockfile。
- 需要修改 harness-pipeline、业务代码、公共 API、数据库或权限。
- 没有可靠 TASK base tree。
- 需要修改当前 TASK scope 外的知识或 schema。
- 当前代码无法证明拟写入的正式结论。
- 知识与更高层级权威来源冲突且无法判定责任方。
- 无法生成真实报告/evidence 而不隐瞒失败或跳过检查。
