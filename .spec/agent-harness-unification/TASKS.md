# TASKS - agent-harness-unification

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
| --- | --- | --- | --- | --- |
| TASK-AHU-001 | 固化唯一权威来源与 Canonical Harness 契约 | pending | `AGENTS.md`、`spec-harness` 规则 | AC-001、AC-006、AC-010、AC-011 |
| TASK-AHU-002 | 实现共享结构、Scope 与 Evidence 校验器 | pending | `spec-harness/scripts` | AC-006、AC-007、AC-008、AC-011 |
| TASK-AHU-003 | 强化 Pipeline 状态与完成不变量 | pending | `harness-pipeline` 规则与脚本 | AC-009、AC-011、AC-013 |
| TASK-AHU-004 | 收敛 Claude 根入口与单 TASK 适配器 | pending | 根 `CLAUDE.md`、Developer/Reviewer/Fixer 适配器 | AC-001、AC-002、AC-003、AC-010 |
| TASK-AHU-005 | 收敛 Claude Coordinator 与 Phase 迁移门禁 | pending | batch coordinator、run-phase、phase agents | AC-003、AC-010、AC-013 |
| TASK-AHU-006 | 冻结 Legacy 入口并生成迁移清单 | pending | Legacy banner、inventory | AC-004、AC-005、AC-013 |
| TASK-AHU-007 | 执行跨控制面最终一致性审计 | pending | 本 feature 的检查、证据与汇总 | AC-001 至 AC-013 |

## Confirmed Design Decisions

本 Harness 基于用户执行 `prepare-harness` 的确认，采用以下默认决策：

1. 旧执行日志中的 11 个 pending 任务冻结并按需迁移，不批量生成业务 SPEC。
2. `.ai-guides/` 默认作为历史和参考；未完成 phase 不自动视为 active。
3. `.spec/semantic-retrieval-score-fixes/` 不在本 feature 中修复，只由 validator 识别并阻止直接执行，后续另建修复 feature。
4. 允许 `completed_with_exceptions`，但每个例外必须有人工批准记录；未批准例外保持 blocked。
5. 不强制所有 Agent 使用 worktree，但每个 TASK 必须具有可靠、可审计的 Git 基线。
6. `.claude/commands/run-phase.md` 暂时保留为迁移门禁，不再直接实施 `.ai-guides` phase。
7. `.claude/agent-memory/` 保留为非权威历史上下文，不删除、不自动作为当前实现依据。
8. `task-scope.json.tasks[*].status` 暂保留为生成时初始值，运行时状态以 `pipeline-state.json` 为准。
9. 提供只读全仓 `.spec` audit 能力，用于识别 current、legacy-compatible 和 unsupported feature。

## TASK-AHU-001 - 固化唯一权威来源与 Canonical Harness 契约

### Goal

在仓库级规则和 `spec-harness` 中明确唯一权威关系、schema v1、机器证据、Review 降级披露、Legacy 迁移门禁和运行时状态来源，为后续校验器与适配器提供稳定契约。

### Scope

- 在 `AGENTS.md` 中明确 `.agents/skills/spec-harness`、`.agents/skills/harness-pipeline` 和 `.spec/<feature>` 的权威关系。
- 更新 `spec-harness` 的文档契约，使其定义 schemaVersion、evidence、preflight、legacy-compatible/unsupported 分类和状态真实性要求。
- 保持六个 mode 名称和既有 staged flow 兼容。
- 不实现共享脚本；脚本实现属于 TASK-AHU-002。

### Allowed Files

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/agent-harness-unification/**`

### Forbidden Files

- `.agents/skills/harness-pipeline/**`
- `.claude/**`
- `docs/agent-harness/**`
- `.ai-guides/**`
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- 已确认的 SPEC 和 SDD
- 无前置 implementation TASK

### Acceptance Criteria

- `AGENTS.md` 明确唯一权威来源且不复制完整 Skill 内容。
- `spec-harness` 定义 schema v1、preflight、机器 evidence 和失败关闭要求。
- 保留 `init-feature`、`draft-spec-sdd`、`prepare-harness`、`implement-task`、`self-review`、`fix-check-failures` 六个 mode。
- 明确独立 Reviewer 优先、self-review 降级必须披露。
- 明确 `pipeline-state.json` 是运行时状态源，`task-scope.status` 不得与其竞争。
- 不新增依赖，不修改业务文件。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-001`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- 检查六个 mode 名称仍存在。
- 检查新增规则不含用户绝对路径。

### Risks

- 共享 Skill 修改影响所有未来 feature。
- 规则写得过细可能与脚本实现重复。
- 需要人工确认后才能实施。

### Notes

- `requiresHumanConfirmation: true`
- 只定义契约，不实现 TASK-AHU-002/003 的脚本逻辑。

## TASK-AHU-002 - 实现共享结构、Scope 与 Evidence 校验器

### Goal

在 `spec-harness` 下实现无新依赖的共享只读 validator、确定性 scope checker 和 evidence 校验能力，使新 feature 可以使用统一实现，旧 feature 可以被分类而不被自动改写。

### Scope

- 新增 `.agents/skills/spec-harness/scripts/` 下的共享脚本。
- 固定提供 `validate-feature.mjs`、`check-task-scope.mjs`、`validate-evidence.mjs`、`audit-specs.mjs` 和 `validator.test.mjs`，避免后续适配器猜测脚本名。
- 验证 feature 结构、TASK ID 对齐、task-scope schema 和 acceptance/report 引用。
- 支持 current、legacy-compatible、unsupported 三类结果。
- 实现 TASK 级 Git 基线、tracked/staged/unstaged/untracked 文件合并和 allow/forbid pattern 匹配。
- 验证 evidence 必需字段和检查退出码语义。
- 提供只读 audit 模式扫描 `.spec/*`，不得修改被扫描 feature。

### Allowed Files

- `.agents/skills/spec-harness/scripts/**`
- `.spec/agent-harness-unification/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.agents/skills/harness-pipeline/**`
- `.claude/**`
- `docs/agent-harness/**`
- `.ai-guides/**`
- 任何其他 `.spec/<feature>` 内容
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- TASK-AHU-001

### Acceptance Criteria

- current schema 返回成功。
- 无 schemaVersion 但符合 `feature_name + tasks` 的结构返回 legacy-compatible 提示。
- `.spec/semantic-retrieval-score-fixes` 的扁平 scope 或缺失 Harness 被识别为 unsupported，并返回非零退出码。
- unknown TASK、缺失 acceptance、无效 pattern 或无可靠基线失败关闭。
- scope checker 覆盖 committed、staged、unstaged 和 untracked 变更。
- TASK 级基线避免前序 TASK 污染。
- forbidden/out-of-scope 文件被列出并返回非零退出码。
- audit 只读且不改写现有 feature。
- evidence validator 能识别失败命令被伪装成成功的冲突。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-002`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- `node .agents/skills/spec-harness/scripts/validator.test.mjs`
- `node .agents/skills/spec-harness/scripts/audit-specs.mjs --root .spec`
- 使用受控临时 fixture 覆盖 current、legacy-compatible、unsupported、unknown TASK、pattern 和多种 Git change 类型。
- 运行全仓只读 `.spec` audit 并确认没有文件被修改。

### Risks

- Git 基线在不同 Agent 环境中行为不同。
- Glob 语义不一致会产生漏报或误报。
- Validator 过严可能阻止 legacy-compatible feature。
- 需要人工确认后才能实施。

### Notes

- `requiresHumanConfirmation: true`
- 不修改任何历史 `.spec` 数据。

## TASK-AHU-003 - 强化 Pipeline 状态与完成不变量

### Goal

更新 `harness-pipeline`，使 pipeline state、Review/Fix 循环和最终状态由机器 evidence 驱动，禁止 scope/test/review/confirmation 失败时写入普通 `completed`。

### Scope

- 更新 `harness-pipeline` 状态 schema 和迁移规则。
- 新增或更新共享 pipeline state 校验脚本。
- 固定提供 `validate-pipeline-state.mjs` 和 `pipeline-state.test.mjs`。
- 定义 `pending`、`in_progress`、`blocked`、`completed`、`completed_with_exceptions`。
- 强制 evidence、scope、checks、review 和 human confirmation 不变量。
- 修正 `skip_human_confirm`：跳过必需 TASK 不得当作完成。
- 保持 `feature_name`、`start_task` 和 `max_review_rounds` 调用兼容。

### Allowed Files

- `.agents/skills/harness-pipeline/SKILL.md`
- `.agents/skills/harness-pipeline/scripts/**`
- `.spec/agent-harness-unification/**`

### Forbidden Files

- `AGENTS.md`
- `.agents/skills/spec-harness/**`
- `.claude/**`
- `docs/agent-harness/**`
- `.ai-guides/**`
- 任何其他 `.spec/<feature>` 内容
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- TASK-AHU-001
- TASK-AHU-002

### Acceptance Criteria

- scope failed + report exists 不能写 `completed`。
- test/check 失败不能写 `completed`。
- Review 不通过不能写 `completed`。
- 需要确认但未确认不能写 `completed`。
- approved exception 必须包含批准对象、批准人、批准时间和原因，并使用 `completed_with_exceptions`。
- `failed_tasks` 或 `blocked_tasks` 非空时不能写 `completed`。
- 缺少 TASK evidence 时不能加入 completed_tasks。
- `skip_human_confirm=true` 产生 blocked/partial 结果，不伪装成完成。
- 旧 state 不被自动静默改写。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-003`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- `node .agents/skills/harness-pipeline/scripts/pipeline-state.test.mjs`
- fixture 覆盖所有状态不变量和 approved exception。

### Risks

- 状态规则变化会影响所有 pipeline feature。
- 与旧 state 的兼容分类需要明确。
- 需要人工确认后才能实施。

### Notes

- `requiresHumanConfirmation: true`
- 不回写历史 pipeline state。

## TASK-AHU-004 - 收敛 Claude 根入口与单 TASK 适配器

### Goal

建立 Claude Code 可发现的根入口，并把 Developer、Reviewer、Fixer 相关 Skill/Agent 收敛成 canonical mode 的薄适配器。

### Scope

- 新增根 `CLAUDE.md`，复用 `AGENTS.md` 并指向权威 `.agents/.spec`。
- 将 `.claude/CLAUDE.md` 改为非权威适配说明。
- 迁移 run/review/fix skills 和 task developer/reviewer/fixer agents。
- 删除适配器中的旧任务目录、固定集成分支、旧 verdict、旧状态和绝对用户路径依赖。
- 保留常用适配器名称。

### Allowed Files

- `CLAUDE.md`
- `.claude/CLAUDE.md`
- `.claude/skills/run-agent-task/SKILL.md`
- `.claude/skills/review-agent-task/SKILL.md`
- `.claude/skills/fix-agent-task/SKILL.md`
- `.claude/agents/task-developer.md`
- `.claude/agents/task-reviewer.md`
- `.claude/agents/task-fixer.md`
- `.spec/agent-harness-unification/**`

### Forbidden Files

- `.claude/settings.local.json`
- `.claude/agent-memory/**`
- `.claude/skills/batch-agent-coordinator/**`
- `.claude/commands/**`
- `.claude/agents/phase-*.md`
- `.agents/**`
- `docs/agent-harness/**`
- `.ai-guides/**`
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- TASK-AHU-001
- TASK-AHU-002
- TASK-AHU-003

### Acceptance Criteria

- 根 `CLAUDE.md` 存在并复用 `AGENTS.md`，不复制完整规则。
- Developer 只映射 `implement-task`。
- Reviewer 只读并映射 `self-review`，最终 verdict 使用权威格式。
- Fixer 只映射 `fix-check-failures`。
- 适配器不再把 `docs/agent-harness`、`EXECUTION_LOG.md` 或 `integration/agent-platform` 声明为权威。
- 适配器不包含绝对用户路径。
- 无独立 subagent 时明确披露 self-review 降级。
- 不修改 Claude 本地权限和 memory 历史。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-004`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- 静态检查根入口、mode mapping、只读边界和旧权威引用。
- 如本地 Claude Code 可用，运行不写文件的发现/解析 smoke check；不可用时记录原因。

### Risks

- Claude Code 版本对根入口和 Skill 发现行为可能不同。
- 过度精简可能丢失有用的 provider-specific 行为。
- 需要人工确认后才能实施。

### Notes

- `requiresHumanConfirmation: true`
- `.claude/agent-memory` 保留但不作为权威。

## TASK-AHU-005 - 收敛 Claude Coordinator 与 Phase 迁移门禁

### Goal

将 Claude batch coordinator 映射到 `harness-pipeline`，并把旧 phase command/agents 转为 `.ai-guides` 到 `.spec` 的迁移门禁。

### Scope

- 精简 batch coordinator，使其不复制 TASK 循环、状态、修复轮数或 merge 行为。
- 统一输入为 canonical Feature/TASK/mode。
- 保留 `run-phase` 名称，但禁止直接实施 `.ai-guides` phase。
- phase agents 仅在已有显式等价 `.spec` 时使用 canonical mode；否则停止并建议 `draft-spec-sdd`。

### Allowed Files

- `.claude/skills/batch-agent-coordinator/SKILL.md`
- `.claude/commands/run-phase.md`
- `.claude/agents/phase-implementer.md`
- `.claude/agents/phase-reviewer.md`
- `.spec/agent-harness-unification/**`

### Forbidden Files

- `.claude/settings.local.json`
- `.claude/agent-memory/**`
- `.claude/skills/run-agent-task/**`
- `.claude/skills/review-agent-task/**`
- `.claude/skills/fix-agent-task/**`
- `.claude/agents/task-*.md`
- `.agents/**`
- `docs/agent-harness/**`
- `.ai-guides/**`
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- TASK-AHU-003
- TASK-AHU-004

### Acceptance Criteria

- batch coordinator 只适配 `harness-pipeline`，不重定义循环和状态。
- 不再固定 `integration/agent-platform`、两轮 Fix 或 squash merge。
- run-phase 发现 `.ai-guides` 输入时不实施代码。
- 没有显式 `.spec` 映射时输出 legacy path、权威入口和推荐 mode。
- 有显式有效 `.spec` 映射时只调用 canonical mode。
- phase reviewer 使用权威 verdict 或明确映射，不产生第三套状态协议。
- 不修改 `.ai-guides` 内容。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-005`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- 静态检查旧 branch/merge/state 规则已移除。
- 模拟无映射和有映射 phase 输入，验证迁移门禁。

### Risks

- 依赖旧 `/run-phase` 的用户需要适应迁移提示。
- Claude command 的参数解析需保持清晰。
- 需要人工确认后才能实施。

### Notes

- `requiresHumanConfirmation: true`
- 不自动创建或改写 `.spec`。

## TASK-AHU-006 - 冻结 Legacy 入口并生成迁移清单

### Goal

在不删除或批量改写历史资料的前提下，为 `docs/agent-harness` 和 `.ai-guides` 添加明确 Legacy/Reference 标识，并登记所有遗留 pending 与不合规 canonical feature。

### Scope

- 新增 `docs/agent-harness/README.md`。
- 仅在 `docs/agent-harness/00-HARNESS.md` 顶部添加非破坏性 Legacy banner。
- 新增 `.ai-guides/README.md`，区分参考指南、历史 delivery 和旧 phase contract。
- 创建 `.spec/agent-harness-unification/reports/legacy-inventory.md`。
- 清单覆盖 11 个旧 pending 任务、所有可执行型 `.ai-guides` phase 和不合规 pending `.spec`。

### Allowed Files

- `docs/agent-harness/README.md`
- `docs/agent-harness/00-HARNESS.md`
- `.ai-guides/README.md`
- `.spec/agent-harness-unification/**`

### Forbidden Files

- `docs/agent-harness/tasks/**`
- `docs/agent-harness/reviews/**`
- `docs/agent-harness/decisions/**`
- `docs/agent-harness/EXECUTION_LOG.md`
- `.ai-guides/*/**`
- `.ai-guides/*.md`（`README.md` 除外）
- `.claude/**`
- `.agents/**`
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- TASK-AHU-001
- TASK-AHU-004
- TASK-AHU-005

### Acceptance Criteria

- 两个 Legacy 入口明确说明“冻结不等于取消”。
- 历史已完成任务、Review、ADR、delivery report 未删除或改写。
- 清单包含旧执行日志中 P1-008 至 P3-001 的 11 个 pending 任务。
- 清单覆盖含 constitution/spec/plan/tasks 的 `.ai-guides` phase，并标记是否存在 delivery/acceptance。
- 清单包含 `.spec/semantic-retrieval-score-fixes`，决策为 separate-repair。
- 每项记录 source、legacy ID、historical status、current code verified、equivalent spec、migration decision 和 notes。
- 不把任何遗留任务自动标记为 active、done 或 cancelled。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-006`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- 对比历史文件清单和 diff，确认除允许入口外无历史内容修改。
- 脚本统计 pending 条目和 phase 目录，并与 inventory 对齐。

### Risks

- 遗漏 pending 条目会导致后续任务不可发现。
- 将历史资料描述为取消会造成错误产品决策。
- 需要人工确认后才能实施。

### Notes

- `requiresHumanConfirmation: true`
- Inventory 只登记，不分析或实施业务功能。

## TASK-AHU-007 - 执行跨控制面最终一致性审计

### Goal

以只读方式验证所有 SPEC acceptance、Claude 适配器、Legacy 标识、shared validator、pipeline state 不变量和历史兼容性，并生成最终审计证据。

### Scope

- 运行 feature Harness 和共享只读 audit。
- 验证所有可执行入口只指向 `.agents/.spec`。
- 验证 Legacy inventory 完整性。
- 验证无业务、依赖、历史内容越界修改。
- 生成 TASK-AHU-007 report/evidence 和 pipeline summary 草案。
- 若发现共享文件问题，只报告并阻塞，不在本 TASK 跨范围修复。

### Allowed Files

- `.spec/agent-harness-unification/**`

### Forbidden Files

- `AGENTS.md`
- `CLAUDE.md`
- `.agents/**`
- `.claude/**`
- `docs/agent-harness/**`
- `.ai-guides/**`
- 所有业务代码、manifest、lockfile、Go module、部署和 CI 文件

### Dependencies

- TASK-AHU-001 至 TASK-AHU-006

### Acceptance Criteria

- `FINAL_AUDIT=1` 的 feature agent-check 通过。
- 所有 AC-001 至 AC-013 均有检查证据或明确阻塞原因。
- 可执行 Claude 入口不存在旧权威路径、旧固定 branch、旧 verdict 或独立状态源。
- current/legacy-compatible/unsupported `.spec` 分类符合实际，且没有被 audit 修改。
- 普通 completed 状态不接受失败 evidence。
- Legacy inventory 与实际 11 个 pending 条目和 phase 目录一致。
- 最终 diff 不含业务、manifest、lockfile、Go module、部署、CI 或未允许历史文件。

### Required Tests

- `git diff --name-only`
- `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-007`
- `bash .spec/agent-harness-unification/scripts/agent-check.sh`
- `FINAL_AUDIT=1 bash .spec/agent-harness-unification/scripts/agent-check.sh`
- 运行 shared `.spec` audit、pipeline invariant tests 和 adapter static checks。

### Risks

- Final audit 可能发现只能在早期共享 TASK 范围内修复的问题。
- 如发现问题，应 verdict 不通过并回到对应 TASK，不得扩大 TASK-AHU-007 scope。

### Notes

- `requiresHumanConfirmation: false`
- 本 TASK 只写本 feature 的 report、evidence 和 summary。
