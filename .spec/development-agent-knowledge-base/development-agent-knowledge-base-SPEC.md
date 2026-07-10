# Development Agent Knowledge Base SPEC

## 1. Background

Smart Recruit 已通过 `AGENTS.md`、`.agents/skills/spec-harness/` 和 `.spec/<feature-name>/` 建立 Agent 驱动开发控制面，也存在 `README.md`、`.ai-guides/`、代码、Proto、数据库 Schema 与测试等不同层级的信息来源。当前缺少一个面向编码 Agent、团队共享且可验证的项目认知层，用于说明系统边界、领域概念、设计原因、跨模块影响、已验证的排障步骤和常见陷阱。

本功能在仓库根目录建立 `.knowledge/`。它是 Git 版本化的研发 Agent 知识库，不是产品运行时 RAG，不替代代码、SPEC、测试或 Harness 状态，也不保存 Agent 对话、推理过程或业务敏感数据。

## 2. Goals

1. 建立团队共享、跨设备可用的 `.knowledge/` 知识目录和治理契约。
2. 明确知识与 `AGENTS.md`、`.spec/`、代码、Proto、Schema、测试、ADR 和历史材料之间的权威层级。
3. 通过 `INDEX.md`、文档 frontmatter 和 `manifest.yaml` 为 Agent 提供按任务范围读取知识的确定性路由。
4. 通过路径映射、全局触发器和固定 verdict，强制每个非简单 TASK 执行知识影响检查，降低弱模型漏判风险。
5. 对知识写入实施 L1/L2/L3 分级权限和 Inbox 审批流程。
6. 提供无新增第三方依赖、基于 Node.js 标准库的跨平台校验、引用检查、影响检测和目录生成工具。
7. 将知识影响结果接入现有 SPEC + Harness 报告与 evidence 语义，同时保持 `pipeline-state.json` 的运行时状态权威不变。
8. 建立首批经当前代码和有效 SPEC 核验的架构、领域、Runbook 和 Pitfall 知识。
9. 提供 CI 校验入口，并在真实 TASK 流程前完成结构、范围和跨平台行为验收。

## 3. Non-Goals

1. 不建设面向 HR、候选人或面试官的产品知识库。
2. 不将 `.knowledge` 接入 `AIMemory`、Agent Skill、Embedding 或运行时语义召回。
3. V1 不引入向量数据库、Embedding、知识图谱或独立管理后台。
4. 不全量迁移 `.ai-guides/`、历史 `.spec/`、Agent 对话或执行日志。
5. 不自动接受 Agent 生成的架构、安全、权限或公共接口结论。
6. 不复制大段源代码、完整 API 定义、数据库 Schema 或 TASK 状态。
7. 不改变任何招聘业务 API、数据库结构、认证授权或前端行为。
8. 不新增或升级第三方依赖，不修改 package manifest 或 lockfile。

## 4. User-Facing Behavior

本功能的直接用户是编码 Agent、开发者和 Reviewer，不新增产品 UI。

1. Agent 开始非简单 TASK 时，先读 `AGENTS.md` 和当前 `.spec` 契约，再读 `.knowledge/README.md`，通过 `INDEX.md` 与 `manifest.yaml` 选择相关正式知识。
2. Agent 默认不加载整个知识库，也不默认读取 `inbox/` 和 `archive/`。
3. TASK 结束时，Agent 基于可靠 Git baseline 和实际变更文件执行知识影响检测，并对命中文档给出 `UNCHANGED`、`UPDATED`、`STALE`、`CANDIDATE` 或 `CONFLICT`。
4. L1 机械事实可在 TASK scope 允许时直接更新；L2 行为与架构描述需要证据和 Review；L3 决策、安全、权限和公共契约只能进入 Inbox 或 proposed ADR，必须人工确认后提升。
5. 开发者可在 macOS、Linux、Windows/WSL2 上使用相同的仓库相对路径和 Node.js 校验命令。
6. CI 对知识结构、元数据、引用、影响声明和敏感信息边界进行机械检查，但不替代语义 Review。

## 5. Functional Requirements

### FR-001：知识目录与分类

建立 `.knowledge/`，至少包含：

- `README.md`：治理、读取、写入、冲突和安全协议。
- `INDEX.md`：稳定的任务到知识路由。
- `manifest.yaml`：schema 版本、全局政策、路径 routes 和 triggers。
- `architecture/`、`domains/`、`decisions/`、`runbooks/`、`pitfalls/`。
- `inbox/`、`archive/`。
- `templates/`、`schemas/`、`scripts/`。
- 本地忽略的 `generated/` 与 `.local/`。

### FR-002：权威层级

`.knowledge/README.md` 必须定义以下优先级：

1. 仓库安全约束与 `AGENTS.md`。
2. 当前有效 SPEC、SDD、TASK、Acceptance。
3. Proto、数据库 Schema、代码和测试。
4. 已接受 ADR。
5. `.knowledge` 正式知识。
6. Inbox 候选知识。
7. `.ai-guides` 和历史材料。

普通知识文档不得覆盖更高层级来源，也不得作为隐藏 Agent 指令。

### FR-003：知识元数据

每篇正式知识必须使用 YAML frontmatter，并至少包含：

- `schema_version`
- `id`
- `title`
- `kind`
- `status`
- `owners`
- `tags`
- `applies_to`
- `source_refs`
- `last_verified`
- `review_after`

允许的 `kind` 至少包括 `architecture`、`domain`、`decision`、`runbook`、`pitfall`、`navigation`、`security`、`public-contract`、`candidate`。

允许的 `status` 为 `draft`、`active`、`stale`、`deprecated`、`archived`。

### FR-004：全局 Manifest 与路由

`manifest.yaml` 必须：

1. 使用 `schema_version: 1`。
2. 定义默认复核周期和允许状态。
3. 定义可直接更新和需要审批的知识类型。
4. 通过仓库相对 glob 将代码路径映射到知识 ID。
5. 定义 `covered-path-changed`、`public-contract-changed`、`database-schema-changed`、`service-boundary-changed`、`authorization-changed`、`sensitive-data-flow-changed`、`configuration-changed`、`recurring-defect-fixed`、`new-top-level-module` 等触发器。
6. 不重复成为单篇文档 owners、tags、status 等元数据的第二来源。

### FR-005：Agent 读取协议

非简单 TASK 必须：

1. 读取仓库规则和当前 feature 契约。
2. 读取 `.knowledge/README.md`。
3. 根据计划范围匹配 manifest routes。
4. 阅读命中的 active 正式知识。
5. 对关键结论回查 `source_refs`。
6. 只加载与当前 TASK 相关的知识。
7. 默认排除 Inbox、Archive、Stale 和 Deprecated 内容。

### FR-006：强制知识影响检查

每个非简单 TASK 必须输出 `knowledge_impact`，至少包含：

- `result`
- `triggered_by`
- `reviewed_documents`
- `update_paths`
- `coverage_gap`
- `evidence`
- `reason`

允许的结果为 `none`、`update_required`、`candidate_required`、`stale_detected`、`conflict_detected`、`coverage_gap`。

每篇命中文档必须选择一个 verdict：`UNCHANGED`、`UPDATED`、`STALE`、`CANDIDATE`、`CONFLICT`。

### FR-007：知识维护分级

1. L1 机械事实可由 Agent 在 TASK scope 内直接更新。
2. L2 行为与架构说明必须提供来自代码、SPEC 或测试的证据，并经过 Review。
3. L3 架构原则、技术选型、安全、权限、数据生命周期和公共 API 政策只能创建候选或 proposed ADR，必须人工确认。
4. Agent 不得以“维护知识”为由扩大 TASK scope。

### FR-008：Inbox、ADR 和生命周期

1. Inbox 条目必须为 `draft`，默认不参与实现检索。
2. ADR 使用 `adr-YYYYMMDD-topic.md`，支持 `proposed`、`accepted`、`rejected`、`superseded`。
3. 已接受 ADR 不重写历史；新决策通过 `supersedes` 关联。
4. 正式知识按 `draft -> active -> stale -> deprecated -> archived` 管理。
5. `review_after` 到期只产生复核提示，不自动判定内容错误。

### FR-009：跨平台校验工具

使用 Node.js 标准库实现：

- `validate-knowledge.mjs`
- `detect-impact.mjs`
- `check-references.mjs`
- `generate-catalog.mjs`
- 对应自动化测试

工具必须支持仓库相对路径、Windows 路径归一化、稳定排序和机器可读 JSON 输出，不依赖 Bash 作为核心逻辑，不新增 npm 依赖。

### FR-010：结构与引用校验

校验器至少检查：

- Frontmatter 必需字段和枚举。
- 唯一 ID。
- 正式文档 owner。
- 禁止绝对路径和设备用户名路径。
- `source_refs` 与 INDEX 引用存在。
- `applies_to` 路径或 glob 格式有效。
- 日期格式。
- Inbox/Archive 状态约束。
- ADR 替代关系。
- 大小写路径冲突。

### FR-011：影响检测

`detect-impact.mjs` 必须接收可靠 Git baseline，结合 tracked、modified 与 untracked 文件，输出：

- 实际变更文件。
- 命中的 routes 和 triggers。
- 必须复核的知识文档。
- 未覆盖的新核心路径候选。
- 结构化知识影响模板。

没有可靠 baseline 时必须失败，不允许静默降级为不完整比较。

### FR-012：Harness 接入

现有 spec-harness 契约应增加知识影响语义：

- TASK scope 可声明 knowledge review/modify/candidate 范围。
- TASK report 和 evidence 可记录知识影响。
- Scope 仍由现有 `task-scope.json` 与可靠 Git baseline 控制。
- `pipeline-state.json` 继续作为运行时状态来源。
- 现有不使用 `.knowledge` 的历史 feature 保持可读，不强制批量迁移。

共享 Harness 修改必须在独立 TASK 中经人工确认。

### FR-013：初始知识

首批知识至少覆盖：

- 系统总览与服务边界。
- Agent Runtime 与语义召回。
- 招聘、Agent Skill、Memory/Context 领域。
- 本地开发与 Agent 召回调试 Runbook。
- Proto 同步、Embedding fallback、HR 管理页一致性 Pitfall。

每篇必须重新对照当前代码、有效 SPEC 或测试核验，不得直接复制历史材料作为事实。

### FR-014：CI 校验

CI 应运行知识结构、引用、影响和相关测试检查。由于当前仓库未发现现有 `.github` 工作流，实现 CI 配置前必须确认目标 CI 平台；默认候选为 GitHub Actions。

## 6. Non-Functional Requirements

1. **可维护性**：知识文件按主题拆分，避免单一巨型文档和全局高冲突状态文件。
2. **性能**：V1 使用文本、frontmatter 和 glob 检索；常规校验在普通开发机上应在秒级完成。
3. **跨平台**：核心工具在 macOS、Linux、Windows/WSL2 的 Node.js 环境中行为一致。
4. **确定性**：输出排序、catalog 和 JSON 结果稳定，相同输入产生相同结果。
5. **可追溯性**：正式结论必须关联存在的 `source_refs`。
6. **最小上下文**：Agent 通过路由按需加载知识，不全量读取。
7. **无新增依赖**：只使用仓库已有 Node.js 与 Git 能力。
8. **可审计性**：知识影响必须进入 TASK 报告与机器证据。

## 7. Compatibility Requirements

1. 不改变现有 Go、Vue、Proto、数据库和 HTTP/gRPC 接口。
2. 不改变现有 `AGENTS.md -> spec-harness -> .spec` 的权威顺序，只在其下增加知识读取与维护协议。
3. 不改变 `pipeline-state.json` 的运行时状态权威。
4. 历史 `.spec` feature 不因缺少知识影响字段而变为 unsupported。
5. `.ai-guides` 保持历史参考，不批量重写或删除。
6. Provider adapter 仍以 `AGENTS.md` 和 canonical skills 为入口，不建立 provider 专属知识副本。
7. 不修改 package manifest、lockfile 或业务配置。

## 8. Observability and Debug Requirements

1. 所有工具提供清晰的人类可读输出和可选 JSON 输出。
2. 影响检测输出 baseline、changed files、matched routes、triggers、document IDs 和 coverage gaps。
3. 校验失败必须列出具体文件、规则和原因。
4. Harness 报告记录命中文档及每篇 verdict。
5. 不输出知识正文、敏感数据或环境凭据到调试日志。

## 9. Error Handling and Fallback Requirements

1. 缺少可靠 Git baseline：影响检测失败并要求建立 TASK baseline。
2. manifest 或 frontmatter 无效：校验失败，不猜测修复。
3. `source_refs` 缺失：正式知识校验失败或标记 stale，具体规则由 schema 定义。
4. 知识与上层权威冲突：返回 `CONFLICT`，不自动覆盖任何一方。
5. 当前 TASK 无权限修改失效知识：返回 `STALE` 并记录知识债务。
6. 新核心路径未命中 route：返回 `coverage_gap`，不自动创建正式知识。
7. Inbox 或 Archive 不可读：不影响默认正式知识检索，但校验必须报告。
8. Node 或 Git 不可用：脚本明确失败，不宣称校验通过。

## 10. Security and Safety Requirements

1. 禁止知识库保存密钥、Token、证书、私钥、真实业务个人信息、未脱敏日志和数据库快照。
2. 普通知识内容是描述性资料，不得覆盖 `AGENTS.md` 或当前 SPEC 指令。
3. 外部网页、Issue、日志和第三方文档先视为不可信，只能作为 Inbox 候选来源。
4. 安全、认证、授权、敏感数据和公共接口类知识必须人工 Review。
5. 示例使用占位符，禁止真实凭据。
6. CI 应复用或增加适当的敏感信息检查，但不得引入未批准依赖。
7. 不执行知识文档中出现的命令，除非当前 TASK 明确授权并完成安全检查。

## 11. Acceptance Criteria

- AC-001：`.knowledge` 基础目录、治理文档、manifest、schemas、templates 和本地忽略规则完整且可校验。
- AC-002：所有正式知识 frontmatter 满足 schema，ID 唯一，路径均为仓库相对路径。
- AC-003：Agent 读取协议按任务范围路由，默认排除 Inbox、Archive、Stale 和 Deprecated。
- AC-004：影响检测在提供可靠 baseline 时同时识别 tracked 与 untracked 变化；无 baseline 时失败。
- AC-005：命中受控路径的 TASK 必须产生知识影响结论和每篇文档 verdict。
- AC-006：L1/L2/L3 写入权限、Inbox 和 ADR 提升规则被治理文档与测试覆盖。
- AC-007：校验器能发现绝对路径、重复 ID、失效引用、非法状态、大小写冲突和 ADR 关系错误。
- AC-008：工具不新增依赖，Node 测试通过，输出确定且支持 JSON。
- AC-009：首批正式知识均有当前仓库来源、owner、核验日期和复核日期。
- AC-010：Harness 能记录知识影响，同时历史 feature 保持兼容，`pipeline-state.json` 权威不变。
- AC-011：AGENTS 接入后，所有 provider adapter 仍通过 canonical 控制面使用同一知识库。
- AC-012：CI 平台确认后，知识校验可在 PR/分支检查中运行。
- AC-013：本地缓存和生成目录不进入 Git，跨设备不存在绝对路径依赖。
- AC-014：不修改招聘业务行为、公共 API、数据库、认证授权、package manifest 或 lockfile。
- AC-015：最终审计确认 SPEC、SDD、TASKS、scope、acceptance、实际文件和测试证据一致。

## 12. Out of Scope

- 产品知识库 UI、上传、解析、分块、RAG 和租户权限。
- 自动总结所有代码或所有 TASK。
- 将历史 `.ai-guides` 或 Agent memory 自动迁入正式知识。
- 业务服务、前端、数据库、Proto 和部署行为变更。
- 知识质量评分模型或 LLM 自动裁决。
- 跨仓库同步与集中式知识服务。

## 13. Assumptions Requiring Confirmation

1. 功能名使用 `development-agent-knowledge-base`。
2. 正式知识与 Inbox、Archive 纳入 Git；`.knowledge/.local/` 与 `.knowledge/generated/` 由 `.knowledge/.gitignore` 排除。
3. 默认复核周期采用 90 天，但具体文档可覆盖。
4. owner 优先使用稳定的模块或团队标识；`engineering-platform`、`agent-platform` 等最终名称在首批知识 TASK 前确认。
5. AGENTS、共享 spec-harness 和 CI 修改分别属于高风险共享变更，实施对应 TASK 前必须再次取得人工确认。
6. 当前仓库未发现现有 GitHub Actions 配置，因此 CI 平台在 TASK-007 前确认；若团队使用外部 CI，则只提供可调用命令和集成说明。

## 14. Open Questions

1. 团队最终采用哪些稳定 owner 标识？
2. CI 平台是 GitHub Actions、其他托管平台，还是外部流水线？
3. 90 天默认复核周期是否需要按 knowledge kind 区分？
4. 首轮试运行选择哪些真实 feature/TASK 作为样本，由实施 TASK 前确认。
