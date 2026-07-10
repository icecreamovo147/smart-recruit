# TASKS - development-agent-knowledge-base

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|
| TASK-001 | 建立知识库基础契约 | pending | `.knowledge` 治理、schema、模板与局部 Git 规则 | acceptance/TASK-001.md |
| TASK-002 | 实现知识校验与影响检测工具 | pending | `.knowledge/scripts` 与 Node 测试 | acceptance/TASK-002.md |
| TASK-003 | 建立首批架构与领域知识 | pending | Architecture 与 Domain 知识 | acceptance/TASK-003.md |
| TASK-004 | 建立首批 Runbook 与 Pitfall | pending | Runbook 与 Pitfall 知识 | acceptance/TASK-004.md |
| TASK-005 | 将知识协议接入 AGENTS | pending | 仓库级 Agent 入口 | acceptance/TASK-005.md |
| TASK-006 | 将知识影响接入 Spec Harness | pending | 共享 spec-harness 契约与校验器 | acceptance/TASK-006.md |
| TASK-007 | 接入知识库 CI 校验 | pending | 确认后的 CI 工作流 | acceptance/TASK-007.md |
| TASK-008 | 执行最终一致性与试运行审计 | pending | 只读审计与 feature 报告 | acceptance/TASK-008.md |

## TASK-001 - 建立知识库基础契约

### Goal

创建 `.knowledge` 治理入口、权威层级、元数据 schema、manifest、模板、Inbox/Archive/ADR 规则及目录局部 Git 配置。

### Scope

只创建基础契约文件，不创建首批业务知识，不实现校验工具，不修改仓库全局规则。

### Allowed Files

- `.knowledge/README.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.knowledge/.gitignore`
- `.knowledge/.gitattributes`
- `.knowledge/decisions/README.md`
- `.knowledge/inbox/README.md`
- `.knowledge/archive/README.md`
- `.knowledge/templates/knowledge-entry.md`
- `.knowledge/templates/adr.md`
- `.knowledge/templates/inbox-candidate.md`
- `.knowledge/schemas/frontmatter.schema.json`
- `.knowledge/schemas/manifest.schema.json`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

None.

### Acceptance Criteria

- `.knowledge` 的定位、权威顺序、读取/写入协议和安全边界完整。
- Frontmatter 与 manifest schema 可由标准 JSON 解析器读取。
- `.local/`、`generated/` 被目录局部 `.gitignore` 排除。
- `.knowledge` 文本文件由局部 `.gitattributes` 约束 LF。
- Inbox、Archive 和 ADR 生命周期清晰。

### Required Tests

- `git diff --check`
- JSON schema 文件语法检查。
- `bash .spec/development-agent-knowledge-base/scripts/check-task-scope.sh TASK-001`
- `bash .spec/development-agent-knowledge-base/scripts/agent-check.sh`

### Risks

- 治理文档与 SPEC/SDD 重复或产生新的权威源。
- YAML 支持范围描述不够明确，导致无依赖解析器难以实现。

### Notes

不得创建任何声称已核验的架构知识；首批内容属于 TASK-003/004。

## TASK-002 - 实现知识校验与影响检测工具

### Goal

使用 Node.js 标准库实现结构校验、引用检查、Git 影响检测、catalog 生成及自动化测试。

### Scope

仅限 `.knowledge/scripts/`；若基础 schema 有实现矛盾，停止并请求调整 TASK scope，不得静默修改 TASK-001 文件。

### Allowed Files

- `.knowledge/scripts/validate-knowledge.mjs`
- `.knowledge/scripts/check-references.mjs`
- `.knowledge/scripts/detect-impact.mjs`
- `.knowledge/scripts/generate-catalog.mjs`
- `.knowledge/scripts/knowledge-validator.test.mjs`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.knowledge/README.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.knowledge/schemas/**`
- `.knowledge/templates/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 completed and confirmed.

### Acceptance Criteria

- 工具接口、退出码和 JSON 输出符合 SDD。
- 无可靠 Git baseline 时影响检测失败。
- tracked 与 untracked 变化均被识别。
- 绝对路径、重复 ID、非法状态、失效引用、大小写冲突、ADR 关系和 coverage gap 有测试。
- 不新增第三方依赖。

### Required Tests

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- `git diff --check`
- Harness scope 与 agent check。

### Risks

- 自行实现 YAML 子集解析可能接受歧义语法。
- Git baseline/untracked 处理错误可能漏报或误报。

### Notes

YAML 支持应严格限制为模板使用的简单子集；不允许通过修改 package manifest 引入 parser。

## TASK-003 - 建立首批架构与领域知识

### Goal

基于当前代码、README 和有效 `.spec` 建立系统总览、服务边界、Agent Runtime、语义召回及核心领域知识。

### Scope

只创建指定 architecture/domain 文档并更新稳定路由所必需的 INDEX/manifest；不修改代码或历史资料。

### Allowed Files

- `.knowledge/architecture/system-overview.md`
- `.knowledge/architecture/service-boundaries.md`
- `.knowledge/architecture/agent-runtime.md`
- `.knowledge/architecture/semantic-retrieval.md`
- `.knowledge/domains/recruitment.md`
- `.knowledge/domains/agent-skill.md`
- `.knowledge/domains/memory-and-context.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 and TASK-002 completed and confirmed.
- Owner taxonomy confirmed or explicitly accepted from SPEC assumptions.

### Acceptance Criteria

- 每篇知识有有效 frontmatter、owner、sources、核验与复核日期。
- 内容基于当前仓库重新核验，不复制历史指南作为事实。
- INDEX 与 manifest routes 能将代表性任务路径路由到相关文档。
- 不包含业务敏感数据、绝对路径或隐藏指令。

### Required Tests

- 知识结构与引用校验。
- 使用代表性 Agent Skill、Memory 和 Embedding 路径运行影响检测。
- `git diff --check`。
- Harness scope 与 agent check。

### Risks

- 历史 `.spec` 的完成状态不同，可能将旧设计误写为当前行为。
- 文档过长或复制实现细节会快速过时。

### Notes

关键结论必须回查代码；`.ai-guides` 只能提供调查线索。

## TASK-004 - 建立首批 Runbook 与 Pitfall

### Goal

建立本地开发、Agent 召回调试 Runbook，以及 Proto 同步、Embedding fallback、HR 管理页一致性 Pitfall。

### Scope

只创建指定 runbook/pitfall 并更新 INDEX/manifest 路由。

### Allowed Files

- `.knowledge/runbooks/local-development.md`
- `.knowledge/runbooks/debug-agent-retrieval.md`
- `.knowledge/pitfalls/protobuf-synchronization.md`
- `.knowledge/pitfalls/embedding-fallback.md`
- `.knowledge/pitfalls/frontend-menu-consistency.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 through TASK-003 completed and confirmed.

### Acceptance Criteria

- 所有命令、路径和流程可从当前仓库验证。
- Runbook 明确通用步骤和 OS 差异，不保存设备本地配置。
- Pitfall 说明触发条件、风险、验证方式和来源。
- Proto、Embedding 和 HR 管理页路径能路由到对应知识。

### Required Tests

- 知识结构与引用校验。
- 代表性路径影响检测。
- `git diff --check`。
- Harness scope 与 agent check。

### Risks

- Runbook 命令随环境变化而过时。
- Pitfall 可能重复 AGENTS 规则而产生双重权威。

### Notes

只描述经过验证的流程；安全凭据统一使用占位符。

## TASK-005 - 将知识协议接入 AGENTS

### Goal

在仓库级 Agent 入口中增加知识读取、影响检查、权威层级、scope 和安全协议，使不同 Provider 通过同一控制面使用 `.knowledge`。

### Scope

修改共享 `AGENTS.md` 及为保持治理一致所必需的 `.knowledge` 入口文件。不得创建 Provider 专属知识副本。

### Allowed Files

- `AGENTS.md`
- `.knowledge/README.md`
- `.knowledge/INDEX.md`
- `.knowledge/manifest.yaml`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `.agents/**`
- `.github/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 through TASK-004 completed and confirmed.
- Explicit human confirmation for shared `AGENTS.md` change.

### Acceptance Criteria

- `AGENTS.md` 明确 `.knowledge` 位于 canonical 控制面下游。
- 非简单 TASK 强制知识影响检查。
- 普通知识不能覆盖 AGENTS、SPEC、代码或测试。
- Scope 外知识只记录债务，不静默修改。
- CLAUDE 等现有 adapter 无需复制规则且仍指向 canonical 入口。

### Required Tests

- `git diff --check`。
- 知识结构与引用校验。
- 搜索 Provider adapter，确认没有引入独立知识源。
- Harness scope 与 agent check。

### Risks

- 新规则过度增加所有 TASK 的上下文与文档负担。
- AGENTS 与 `.knowledge/README` 表述不一致。

### Notes

`requiresHumanConfirmation: true`。没有明确确认不得实施。

## TASK-006 - 将知识影响接入 Spec Harness

### Goal

向共享 spec-harness 增加向后兼容的知识影响 scope、报告和 evidence 契约及校验覆盖。

### Scope

只修改 spec-harness 技能和共享校验器/测试，以及必要的知识 schema/治理说明。不得修改 harness-pipeline 状态语义。

### Allowed Files

- `.agents/skills/spec-harness/SKILL.md`
- `.agents/skills/spec-harness/scripts/validate-feature.mjs`
- `.agents/skills/spec-harness/scripts/check-task-scope.mjs`
- `.agents/skills/spec-harness/scripts/validate-evidence.mjs`
- `.agents/skills/spec-harness/scripts/validator.test.mjs`
- `.knowledge/README.md`
- `.knowledge/schemas/**`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/skills/harness-pipeline/**`
- `.github/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 through TASK-005 completed and confirmed.
- Explicit human confirmation for shared spec-harness changes.

### Acceptance Criteria

- 新 feature 可声明 knowledge scope 和 required knowledgeImpact。
- 历史 feature 不因缺失新字段变为 unsupported。
- Scope 仍要求可靠 TASK baseline。
- Evidence 失败、跳过或冲突不能被描述为完成。
- `pipeline-state.json` 权威语义不变。

### Required Tests

- `node .agents/skills/spec-harness/scripts/validator.test.mjs`
- 知识工具测试与校验。
- 至少验证一个 current 和一个 legacy-compatible 历史 feature。
- Harness scope 与 agent check。

### Risks

- 共享校验器回归会阻塞所有 feature。
- 错误地强制历史 evidence 迁移会破坏兼容性。

### Notes

`requiresHumanConfirmation: true`。若实现需要修改 harness-pipeline，停止并请求扩大 scope。

## TASK-007 - 接入知识库 CI 校验

### Goal

在确认团队 CI 平台后，将知识结构、引用、测试和影响检查接入 PR/分支校验。

### Scope

默认候选为创建单一 GitHub Actions 知识校验工作流；若团队使用其他平台，必须先修订本 TASK scope，不得猜测创建错误配置。

### Allowed Files

- `.github/workflows/knowledge-validation.yml`
- `.knowledge/README.md`
- `.spec/development-agent-knowledge-base/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 through TASK-006 completed and confirmed.
- CI platform explicitly confirmed.
- Explicit human confirmation for CI configuration change.

### Acceptance Criteria

- 工作流只读取仓库并运行已存在的知识校验命令。
- 不需要 Secret，不执行业务部署。
- Node 版本明确且与仓库开发约束兼容。
- PR/分支触发范围避免无关高成本任务。
- CI 文档说明本地等价命令。

### Required Tests

- CI 配置语法检查（使用仓库已有能力；无可用解析器时记录人工验证）。
- 本地执行工作流中的全部命令。
- `git diff --check`。
- Harness scope 与 agent check。

### Risks

- 当前仓库未发现既有 CI 约定。
- 新工作流可能与外部 CI 重复。

### Notes

`requiresHumanConfirmation: true`。CI 平台未确认时即使文件位于 allowed scope 也不得实施；若确认的平台不是 GitHub Actions，必须先请求修订 TASK scope。

## TASK-008 - 执行最终一致性与试运行审计

### Goal

只读验证完整知识库、路由、Harness 兼容性、跨设备约束和代表性 TASK 影响检测，并生成最终审计证据。

### Scope

只允许创建/更新本 feature 报告和 evidence；发现实现缺陷必须返回所属 TASK，不得直接修业务或共享文件。

### Allowed Files

- `.spec/development-agent-knowledge-base/reports/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/**`
- `.github/**`
- `.knowledge/**`
- `.ai-guides/**`
- `CLAUDE.md`
- `.claude/**`
- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `deploy/**`
- `docker/**`
- `db.sql`
- `pnpm-workspace.yaml`
- `pnpm-lock.yaml`
- `**/package.json`
- `**/go.mod`
- `**/go.sum`

### Dependencies

- TASK-001 through TASK-007 completed and confirmed, or documented approved exception for an explicitly deferred CI platform.

### Acceptance Criteria

- SPEC、SDD、TASKS、scope、acceptance 和实际文件一致。
- 知识校验、引用检查、工具测试和 Harness 回归通过。
- 代表性路径覆盖 Agent Skill、Memory、Proto 和 HR 管理页。
- 无绝对路径、敏感数据、Provider 专属副本和未记录 coverage gap。
- 最终报告明确风险、例外和是否可完成 feature。

### Required Tests

- 全部知识工具测试与校验。
- 共享 spec-harness validator tests。
- `node .agents/skills/spec-harness/scripts/validate-feature.mjs --feature .spec/development-agent-knowledge-base`。
- `git diff --check`。
- Harness scope 与 agent check。

### Risks

- 只读审计无法直接修复发现的问题。
- 未选择合适的真实样本会导致路由误报/漏报未被发现。

### Notes

若 CI 经用户批准延期，必须在 evidence 中记录 exception，不能描述为无条件完成。
