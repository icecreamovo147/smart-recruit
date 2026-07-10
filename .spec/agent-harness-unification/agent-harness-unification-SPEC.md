# Agent Harness Unification SPEC

## 1. Background

本仓库已经明确采用 Specification-Driven Development：根目录 `AGENTS.md` 规定非平凡功能必须依次经过 SPEC、SDD、TASKS、Harness、单 TASK 实现、检查、报告与人工确认；`.agents/skills/spec-harness/SKILL.md` 和 `.agents/skills/harness-pipeline/SKILL.md` 定义了对应的权威执行协议；`.spec/<feature-name>/` 保存每个功能的需求、设计、任务、验收、状态和报告。

用户已确认以下来源具有权威性：

1. `AGENTS.md`：仓库级长期规则；
2. `.agents/skills/spec-harness/SKILL.md`：SPEC、SDD、Harness 和单 TASK 生命周期；
3. `.agents/skills/harness-pipeline/SKILL.md`：多 TASK 串行编排；
4. `.spec/<feature-name>/`：功能级唯一事实源。

仓库同时保留了两组非权威执行体系：

- `.claude/CLAUDE.md`、`.claude/skills/`、`.claude/agents/` 使用 `docs/agent-harness/tasks/`、`EXECUTION_LOG.md`、固定集成分支以及另一套 Developer / Reviewer / Fixer 协议；
- `.claude/commands/run-phase.md` 和 phase agents 使用 `.ai-guides/<phase>/` 下的 `constitution.md`、`spec.md`、`plan.md`、`tasks.md` 作为执行输入。

这些体系在任务位置、状态模型、Review verdict、修复轮数、分支策略和完成条件上不一致。当前仓库根目录不存在 `CLAUDE.md`，只有 `.claude/CLAUDE.md`，而多个 Claude Skill 又要求读取根目录 `CLAUDE.md`，导致 Claude Code 的规则发现和加载行为存在不确定性。

现有数据还表明权威流程需要补强：

- `docs/agent-harness/EXECUTION_LOG.md` 中有 13 个历史任务已通过或合并，另有 11 个 pending 任务尚未迁移；
- `.spec/semantic-retrieval-score-fixes/` 仍为 pending，但使用旧式扁平 `task-scope.json`，且缺少 `scripts/` 和 `pipeline-state.json`；
- `.spec/agent-skill-selection-confirmation/reports/TASK-ASC-005-report.md` 明确记录 scope 检查失败，但对应 `pipeline-state.json` 仍将整个 pipeline 标记为 `completed`；
- `.spec/bailian-embedding-provider/scripts/check-task-scope.sh` 只展示 diff 并要求人工比较，没有执行失败关闭。

因此需要统一 Agent 控制面，同时强化权威 Harness 的状态真实性、范围校验和可审计证据。

## 2. Goals

1. 建立与具体模型供应商无关的单一 Agent 治理体系。
2. 确保 Codex、Claude Code 和未来其他 Coding Agent 都使用相同的 `.agents` 工作流和 `.spec` 功能契约。
3. 将 Claude Code 配置收敛为权威流程的兼容适配层，不再复制另一套任务、Review 或状态规则。
4. 将 `docs/agent-harness/` 和可执行型 `.ai-guides/` 流程转为只读历史或参考资料，禁止继续作为新任务的执行入口。
5. 为遗留 pending 任务建立可追踪的迁移清单，防止任务静默丢失或被重复实现。
6. 为 `task-scope.json`、`pipeline-state.json`、Review verdict 和测试证据定义统一、可验证的契约。
7. 确保 scope、Harness、测试或 Review 失败时采用失败关闭，禁止普通 `completed` 状态掩盖失败或未批准例外。
8. 保留现有历史文档、Review 记录和完成报告，不破坏审计价值。

## 3. Non-Goals

1. 不实现 `docs/agent-harness/tasks/` 中任何 pending 业务功能。
2. 不实现或修复 `.ai-guides/` 中描述的业务需求。
3. 不修改前端、Go 服务、数据库、Proto、部署或运行时业务配置。
4. 不删除历史任务、Review、ADR、执行日志、`.ai-guides` 文档或已完成 `.spec` 功能目录。
5. 不批量重写已完成 feature 的 SPEC、SDD、TASKS 或报告。
6. 不进行 Codex 与 Claude 模型能力或质量排名。
7. 不引入新的 Go、npm、Python 或系统依赖。
8. 不在本功能中建立新的 CI/CD 平台或远程任务服务。
9. 不自动迁移仍不明确是否活跃的遗留任务。

## 4. User-Facing Behavior

### 4.1 新功能开发

用户无论通过 Codex 还是 Claude Code 发起非平凡功能开发，Agent 都应：

1. 读取 `AGENTS.md`；
2. 选择 `.agents/skills/spec-harness/SKILL.md`；
3. 在 `.spec/<feature-name>/` 创建或读取功能契约；
4. 按 `draft-spec-sdd`、`prepare-harness`、`implement-task`、`self-review`、`fix-check-failures` 的权威语义执行；
5. 仅在用户明确调用 pipeline 时使用 `.agents/skills/harness-pipeline/SKILL.md`。

### 4.2 Claude Code 兼容

Claude Code 应具有仓库根目录入口，并明确：

- 导入或遵循 `AGENTS.md`；
- `.agents/skills/` 是权威工作流定义；
- `.spec/` 是唯一可执行功能来源；
- `.claude/` 中保留的命令、Skill 和 Agent 只是适配器；
- 不得继续从 `docs/agent-harness/tasks/` 或 `.ai-guides/<phase>/tasks.md` 直接实施代码。

### 4.3 遗留任务调用

当用户引用 P0/P1/P2/P3 遗留任务或 `.ai-guides` phase 时，Agent 不得直接执行。Agent必须：

1. 识别其为遗留输入；
2. 查找是否已有对应 `.spec/<feature-name>/`；
3. 若没有，则停止实施并要求先迁移为新的 SPEC/SDD/Harness；
4. 不得自动假设旧文档与当前代码仍一致。

### 4.4 状态与失败展示

用户应能从 pipeline state、TASK report 和机器证据中明确区分：

- 尚未开始；
- 正在实现；
- 正在 Review 或修复；
- 因范围、测试、Review 或确认门禁被阻塞；
- 全部检查通过并完成；
- 经人工明确批准后带例外完成。

## 5. Functional Requirements

### FR-001：权威来源声明

仓库必须明确声明 `AGENTS.md`、两个 `.agents` Skill 和 `.spec/<feature-name>/` 是 Agent 开发流程的唯一权威来源。任何适配器和参考文档不得覆盖这些规则。

### FR-002：Claude Code 根入口

仓库必须提供 Claude Code 可发现的根目录项目指令入口。该入口必须复用 `AGENTS.md`，并只包含 Claude Code 特有的适配说明，不得复制完整仓库规则。

### FR-003：Claude 适配器收敛

仍需保留的 `.claude/skills/`、`.claude/agents/` 和 `.claude/commands/` 必须映射到以下权威模式：

| Claude 角色或命令 | 权威模式 |
| --- | --- |
| Developer / run-agent-task | `spec-harness implement-task` |
| Reviewer / review-agent-task | `spec-harness self-review` |
| Fixer / fix-agent-task | `spec-harness fix-check-failures` |
| Batch coordinator | `harness-pipeline` |
| Phase workflow | 先迁移到 `.spec`；未迁移时停止 |

适配器不得继续维护独立的任务目录、状态文件、修复轮数、Review verdict、固定集成分支或完成定义。

### FR-004：遗留体系冻结

`docs/agent-harness/` 和 `.ai-guides/` 必须有清晰的 Legacy / Reference 标识。标识必须说明：

- 内容保留用于历史、架构背景和审计；
- 不再作为新代码任务的直接执行输入；
- pending 工作必须迁移到新的 `.spec/<feature-name>/` 后才能继续；
- 历史状态不得被批量改写成新状态。

### FR-005：遗留迁移清单

本功能必须产出可审计的遗留清单，至少覆盖：

- `docs/agent-harness/EXECUTION_LOG.md` 中 11 个 pending 任务；
- `.ai-guides/` 中包含 `constitution.md`、`spec.md`、`plan.md`、`tasks.md` 的 phase 目录；
- `.spec/semantic-retrieval-score-fixes/` 等未满足当前 Harness 前置条件的功能目录。

每个条目必须记录来源、当前状态、是否已有等价 `.spec`、建议迁移方式以及是否需要人工确认。

### FR-006：遗留任务迁移门禁

遗留任务在迁移时必须重新核对当前代码，不得机械复制旧任务文件。迁移至少需要：

1. 新的 feature name；
2. 当前代码分析；
3. SPEC 和 SDD；
4. 用户确认；
5. `prepare-harness` 生成的新 TASKS、acceptance 和 scope；
6. 重复实现与现有功能冲突检查。

### FR-007：Harness 结构校验

权威流程必须能够在实施前验证 feature 目录是否满足当前契约，包括：

- SPEC 和 SDD 文件名正确；
- `TASKS.md`、`AGENT_RULES.md`、`task-scope.json` 存在；
- 对应 acceptance、prompts、scripts 和 reports 目录存在；
- TASK ID 在所有相关文件中一致；
- `task-scope.json` 使用受支持的 schema；
- pipeline 运行时存在合法 `pipeline-state.json`。

缺失或旧式结构必须明确失败，不能退化成人工目测后继续。

### FR-008：确定性的 TASK 范围校验

scope 检查必须：

1. 接收明确 TASK ID；
2. 使用明确的 TASK 起始基线或等价的任务级 change set；
3. 覆盖已跟踪、已暂存和未跟踪文件；
4. 同时检查 `allowedFiles` 和 `forbiddenFiles`；
5. 对任何越界文件返回非零退出码；
6. 输出实际文件、匹配规则和失败原因；
7. 避免把前序 TASK 的已确认变更误判为当前 TASK 变更。

仅打印 diff、要求 Agent 人工比较的脚本不满足要求。

### FR-009：统一 Review 协议

权威 Review 输出必须遵循 `spec-harness self-review` 的 finding 格式，并以以下之一结束：

```text
verdict: 通过
```

或：

```text
verdict: 不通过
```

Reviewer 必须只读。平台支持独立 Agent 或新上下文时，应优先采用独立 Review；无法独立 Review 时，报告必须明确这是同一 Agent 的 self-review。

### FR-010：机器可读执行证据

每个 TASK 必须能够生成与 Markdown 报告对应的机器可读证据，至少包含：

- feature 和 TASK ID；
- TASK 起始与结束 Git 标识；
- 实际修改文件；
- scope 检查结果；
- 执行的检查命令、退出码和时间信息；
- Review verdict；
- 人工确认或批准例外；
- 未执行检查及原因。

Markdown 报告不得把失败命令描述为通过，也不得仅依赖自然语言声明完成。

### FR-011：状态真实性

pipeline 只有在以下条件全部满足时才能标记普通 `completed`：

- 所有必需 TASK 均完成；
- 所有 scope 检查通过；
- 所有必需测试通过；
- 所有 Review verdict 为通过；
- 所有需要人工确认的 TASK 已有明确确认；
- `failed_tasks` 和未批准例外为空。

若存在经过批准的例外，必须使用区别于普通完成的状态并记录批准信息。存在未批准失败时必须为 blocked 或失败状态。

### FR-012：历史兼容

本功能不得修改既有已完成任务的业务代码、历史 Review 结论或 commit 记录。历史文档中的旧术语可保留，但必须通过顶层标识阻止其继续作为权威执行规则。

### FR-013：适配器与权威流程一致性检查

必须提供可重复执行的检查，验证：

- Claude 可执行入口没有继续引用旧任务执行路径；
- Claude verdict 和模式能够映射到权威协议；
- 新功能只从 `.spec` 读取；
- 适配器没有重新定义权威 Skill 已定义的核心流程；
- Legacy 标识存在且内容明确。

## 6. Non-Functional Requirements

### 6.1 可维护性

- 核心流程规则只能在 `.agents/skills/` 中定义一次。
- 适配器应保持短小，只做入口发现、参数映射和能力差异说明。
- 新增 schema 和状态字段必须有文档和校验逻辑。

### 6.2 可审计性

- 状态变化必须能追溯到 TASK、Git 标识、检查结果和 Review verdict。
- 人工批准必须记录批准对象、原因和时间。
- 历史资料必须保持可读。

### 6.3 可移植性

- 不依赖绝对用户目录。
- 不依赖仅在单一模型供应商中存在的隐藏状态。
- Shell 脚本应兼容本项目现有 macOS/zsh 环境，并避免无必要的 GNU-only 语法。
- 优先复用仓库已有 Bash、Git 和 Node 运行时，不增加依赖。

### 6.4 确定性

- 同一 feature、TASK、Git 基线和工作区状态应得到相同的 scope 判断。
- 缺失文件、未知 schema、未知 TASK 或无效状态必须明确失败。

### 6.5 性能

- Harness 结构与静态一致性检查应在本地快速完成。
- 不应为了治理检查默认运行所有业务模块的全量测试；业务测试仍按 TASK 范围选择。

## 7. Compatibility Requirements

1. 现有 `spec-harness` 六个 mode 名称保持兼容。
2. 现有 `harness-pipeline` 入口和 `feature_name`、`start_task`、`max_review_rounds` 参数保持兼容，除非后续 TASK 明确批准变更。
3. 已满足当前结构的 `.spec` feature 应继续可读；新增严格校验不得静默改写其文件。
4. 不符合新结构的 feature 应被报告为需要迁移或修复，而不是自动推断旧 schema。
5. Claude 现有常用命令名可通过薄适配器暂时保留，但其执行语义必须切换到 `.agents/.spec`。
6. `docs/agent-harness/`、`.ai-guides/` 和 `.claude/agent-memory/` 历史内容保持可访问。
7. 不强制修改业务开发分支或当前用户工作区；分支与 worktree 行为必须服从当前任务和运行环境。

## 8. Observability and Debug Requirements

1. Harness preflight 必须输出 feature、TASK、权威 Skill、schema 版本和 Git 基线。
2. scope 检查必须列出检测到的全部变更文件及其 allow/forbid 判断。
3. pipeline 每次状态迁移必须记录当前 phase、TASK、Review 轮次和阻塞原因。
4. 适配器拒绝遗留任务时必须指出原始路径和需要创建的 `.spec` 入口。
5. 一致性检查必须输出所有旧执行引用，而不是只返回笼统失败。
6. 日志和报告不得包含密钥、Cookie、本地凭证或敏感候选人数据。

## 9. Error Handling and Fallback Requirements

1. 未找到根规则、Skill 或 feature 文件时立即停止，不猜测替代来源。
2. 遇到未知 `task-scope.json` 或 `pipeline-state.json` schema 时失败并给出迁移建议。
3. scope、测试或 Review 失败时不得推进到下一 TASK。
4. Claude 适配器无法调用独立 Agent 时，可退化为权威 `self-review`，但必须披露该降级。
5. 遗留任务没有等价 `.spec` 时，只允许创建 SPEC/SDD 迁移草案，不允许直接实施。
6. 遗留资料与当前代码冲突时，以当前代码和新 SPEC/SDD 为准，并将冲突列为待确认项。
7. 不允许通过删除失败证据、放宽断言或扩大 scope 来规避检查。

## 10. Security and Safety Requirements

1. 不得扩大 Claude Code、Codex、Shell、MCP 或本地文件访问权限。
2. 不得把 `.claude/settings.local.json` 等本地权限或凭证配置纳入共享权威规则。
3. 不得把绝对用户路径写入新的共享 Agent 配置、Skill 或报告模板。
4. 不得提交 API Key、Token、Cookie、邮箱凭证或其他秘密。
5. 历史 Agent memory 只可作为非权威参考，不得在未验证当前代码的情况下驱动实施。
6. 任何删除历史文件、修改全局配置或扩大工具权限的 TASK 都必须设置人工确认。

## 11. Acceptance Criteria

### AC-001：唯一权威来源

仓库入口文档明确声明 `.agents/.spec` 权威关系，Codex 与 Claude Code 的新功能开发都落到同一 feature contract。

### AC-002：Claude 入口可发现

仓库根目录存在 Claude Code 项目入口，并复用 `AGENTS.md`；入口不复制完整 Harness 规则。

### AC-003：旧执行引用清零

所有仍可执行的 Claude Skill、Agent 和 command 不再把 `docs/agent-harness/tasks/`、`EXECUTION_LOG.md` 或 `.ai-guides/<phase>/tasks.md` 作为代码实施的权威输入。Legacy 提示和迁移检测中的引用不计为违规。

### AC-004：Legacy 标识与历史保留

`docs/agent-harness/` 和 `.ai-guides/` 均有清晰 Legacy / Reference 标识；既有历史任务、Review、ADR 和报告未被删除或批量改写。

### AC-005：迁移清单完整

迁移清单至少包含旧执行日志中的 11 个 pending 任务、可执行型 `.ai-guides` phase，以及不满足当前 Harness 前置条件的 `.spec` feature，并标明迁移状态和人工决策点。

### AC-006：结构校验失败关闭

对缺少 scripts、pipeline state 或使用旧式扁平 `task-scope.json` 的 feature 运行 preflight 时返回非零退出码，并指出具体缺失项。

### AC-007：scope 检查失败关闭

测试场景中加入一个允许文件和一个禁止或越界文件时，scope 检查必须列出越界文件并返回非零退出码；仅包含允许文件时返回零。

### AC-008：TASK 级基线

连续执行两个 TASK 时，第二个 TASK 的 scope 判断不会把第一个已完成 TASK 的变更误算为自身变更，同时仍能发现第二个 TASK 新增的 staged、unstaged 和 untracked 越界文件。

### AC-009：状态真实性

任一必需 scope、测试或 Review 失败时，pipeline 无法写入普通 `completed`。存在批准例外时，状态与证据能明确区别于无例外完成。

### AC-010：Review 协议统一

Claude 适配器和 Codex 使用相同 finding 结构与最终 verdict；Reviewer 不修改代码，无法独立 Review 时有明确披露。

### AC-011：机器证据可验证

代表性 TASK 的机器证据包含 Git 基线、文件清单、命令退出码、scope 结果和 Review verdict，并与 Markdown 报告一致。

### AC-012：无业务与依赖变更

最终变更不包含前端、Go 业务代码、数据库、Proto、package manifest、lockfile、go.mod、go.sum、部署或 CI/CD 修改。

### AC-013：兼容性检查

现有已完成 `.spec` feature 和历史资料保持可读；不合规的 pending feature 被明确阻止而非被静默升级或误判完成。

## 12. Out of Scope

- 将旧 P0/P1/P2/P3 任务真正实施或逐项迁移成业务 feature；
- 修复 `semantic-retrieval-score-fixes` 的业务实现；
- 重构招聘系统业务架构；
- 修改模型、Prompt、MCP、RAG、Skill 或 Memory 产品功能；
- 新增 GitHub Action、远程 Agent 平台或集中式数据库；
- 自动提交、推送、创建 PR 或合并分支；
- 清理用户本地 Claude 权限配置；
- 删除 `.claude/agent-memory/` 或历史 Review。

## 13. Assumptions Requiring Confirmation

1. 假设根目录 `CLAUDE.md` 采用导入 `AGENTS.md` 的薄入口，并保留 `.claude/CLAUDE.md` 作为适配说明或将其内容迁移后标记为非权威。
2. 假设 Claude 现有常用 Skill 名称暂时保留为兼容适配器，而不是立即删除。
3. 假设 `docs/agent-harness/` 中 13 个已通过或合并任务只保留历史，不迁移到 `.spec`。
4. 假设 11 个 pending 旧任务先冻结并登记，只有用户决定继续某项时才按当前代码重新生成 `.spec`，不在本功能中批量迁移。
5. 假设 `.ai-guides/` 默认作为历史、设计和参考资料；仍然活跃的 phase 需要用户逐项确认后迁移。
6. 假设 `.spec/semantic-retrieval-score-fixes/` 由单独的 Harness 修复流程处理，本功能只要求 validator 能识别并阻止其直接执行。
7. 假设新增机器证据文件可以位于 `.spec/<feature>/reports/`，不需要外部数据库。
8. 假设已批准例外需要区别于普通完成，但最终状态名称仍需用户确认。

## 14. Open Questions

1. 11 个 pending 旧任务应按需迁移，还是需要一次性生成迁移后的 `.spec` 草案？
2. `.ai-guides/candidate-match-evaluation-algorithm`、`.ai-guides/hr-ai-context-usage` 等未见 delivery report 的 phase 是否仍为活跃工作？
3. `semantic-retrieval-score-fixes` 的 Harness 修复是否应成为本 feature 的一个 TASK，还是另建 `semantic-retrieval-score-fixes-harness-repair`？
4. 批准例外后的 pipeline 状态应命名为 `completed_with_exceptions`、`accepted_with_exceptions`，还是直接保持 blocked 直到全部消除？
5. 是否要求每个实现 TASK 强制使用独立 worktree，还是只要求存在可确定的 TASK 基线并允许当前 Codex 工作区模式？
6. `.claude/commands/run-phase.md` 和 phase agents 应保留为“迁移提示器”，还是直接标记 deprecated 并停止注册？
7. `.claude/agent-memory/` 中的历史项目知识是否继续供 Claude 参考，还是只保留审计用途并禁止自动加载？
