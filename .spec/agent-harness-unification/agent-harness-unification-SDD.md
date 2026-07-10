# Agent Harness Unification SDD

## 1. Existing Architecture Summary

### 1.1 权威控制面

当前仓库权威 Agent 流程由四层组成：

```text
AGENTS.md
  └─ 仓库结构、编码规范、测试规则、安全规则、SPEC+SDD+Harness 总约束

.agents/skills/spec-harness/SKILL.md
  └─ init-feature / draft-spec-sdd / prepare-harness
     / implement-task / self-review / fix-check-failures

.agents/skills/harness-pipeline/SKILL.md
  └─ 多 TASK 串行编排、状态恢复、Review/Fix 循环、人工门禁

.spec/<feature-name>/
  └─ SPEC、SDD、TASKS、AGENT_RULES、task-scope、acceptance、
     prompts、scripts、reports、pipeline-state
```

根 `AGENTS.md` 已规定所有非平凡功能必须使用 `.spec/<feature-name>/`。用户进一步确认 `.agents` 下的 spec+harness 是权威来源。

### 1.2 Claude 旧任务控制面

`.claude/CLAUDE.md`、`.claude/skills/batch-agent-coordinator/`、`.claude/skills/run-agent-task/` 及 task developer/reviewer/fixer agents 使用另一套架构：

```text
docs/agent-harness/tasks/P*-*.md
  + docs/agent-harness/00..06 文档
  + docs/agent-harness/EXECUTION_LOG.md
  + integration/agent-platform
  + agent/<task-id>-<task-name>
  + PASS / NEEDS_FIX / BLOCKED
```

该体系要求独立 task branch/worktree、最多两轮 Fix，并由 coordinator 直接 squash merge。`EXECUTION_LOG.md` 记录了 13 个历史完成或合并任务和 11 个 pending 任务。

仓库根目录当前没有 `CLAUDE.md`；只有 `.claude/CLAUDE.md`。现有 Claude Skill 多次要求读取 `CLAUDE.md`，但没有统一解析到 `.claude/CLAUDE.md` 的仓库内契约。

### 1.3 Claude phase 控制面

`.claude/commands/run-phase.md`、`phase-implementer.md` 和 `phase-reviewer.md` 使用第三种执行布局：

```text
.ai-guides/<phase-slug>/
  constitution.md
  spec.md
  plan.md
  tasks.md
  delivery-report.md
  acceptance-review.md
```

该流程使用 `codex/<phase-slug>` 分支、最多五轮实现/审查循环和 `DECISION: PASS | NEEDS_WORK` 协议。`.ai-guides/` 同时还包含大量设计指南、计划、审查报告和历史材料，不能整体视为同一种状态。

### 1.4 当前 `.spec` 数据形态

现有 `.spec` 中多数 feature 使用：

```json
{
  "feature_name": "...",
  "tasks": {
    "TASK-...": {
      "allowedFiles": [],
      "forbiddenFiles": [],
      "requiresHumanConfirmation": false
    }
  }
}
```

但仍存在不一致：

- `.spec/semantic-retrieval-score-fixes/task-scope.json` 使用 `{feature, allowedFiles}` 扁平结构；
- 该 feature 缺少 `scripts/` 和 `pipeline-state.json`；
- 部分较早 scope script 只输出变更，不执行规则匹配；
- 部分 pipeline report 记录 scope 失败，但 pipeline state 仍为 `completed`；
- `task-scope.json` 内 TASK status 可一直保持 `pending`，同时 pipeline state 已将 TASK 列为完成，存在双重状态漂移。

## 2. Problem Analysis

### 2.1 多控制面冲突

当前不同 Agent 可能依据启动入口读取不同事实源。同一个需求可能落入 `.spec`、`docs/agent-harness/tasks` 或 `.ai-guides/<phase>`，从而产生不同任务 ID、范围、Review 协议和完成条件。

### 2.2 规则复制导致漂移

Claude Skill、Agent prompt、`CLAUDE.md` 和旧 Harness 文档重复声明分支、测试、Review 和停止条件。当权威 Skill 更新后，复制内容不会自动同步。目前已出现 2、3、5 轮修复上限以及三套 verdict 的直接冲突。

### 2.3 状态由自然语言主导

当前 Markdown report 可以说明 scope 检查失败，但 pipeline state 仍由编排 Agent 写成 `completed`。这说明测试证据、scope 结果、Review verdict 和最终状态之间没有可执行的不变量。

### 2.4 范围基线不确定

现有 scope script 常读取整个 working tree 的 unstaged、staged 和 untracked 变更。若多个 TASK 在同一工作区串行完成且没有确定基线，前序 TASK 变更会污染后续 TASK 判断；若脚本因此被降级为人工目测，又失去失败关闭能力。

### 2.5 历史和待办混合

`docs/agent-harness/` 和 `.ai-guides/` 同时承载历史记录、仍 pending 的任务和架构参考。简单删除会损失审计价值；继续直接执行则会绕过权威流程。

### 2.6 Review 独立性不稳定

权威模式名称是 `self-review`，Claude 旧体系则有独立 reviewer agent。统一后需要保持一个协议，同时允许有能力的平台使用独立上下文，不能让“是否独立”改变 verdict 或状态语义。

## 3. Proposed Design

### 3.1 分层控制面

统一后采用以下层次：

```text
L0  Repository Constitution
    AGENTS.md

L1  Canonical Workflow
    .agents/skills/spec-harness/SKILL.md
    .agents/skills/harness-pipeline/SKILL.md

L2  Feature Contract and Runtime State
    .spec/<feature-name>/...

L3  Provider Adapters
    CLAUDE.md
    .claude/skills/**
    .claude/agents/**
    .claude/commands/**

L4  Legacy and Reference Material
    docs/agent-harness/**
    .ai-guides/**
    .claude/agent-memory/**
```

依赖只能自下而上引用：L3 可以引用 L0-L2，L4 不得反向成为 L1-L3 的可执行权威来源。

### 3.2 根入口设计

新增根目录 `CLAUDE.md`，内容保持最小化：

1. 导入或明确要求读取 `AGENTS.md`；
2. 声明 `.agents/skills/spec-harness/SKILL.md` 和 `harness-pipeline/SKILL.md` 是工作流权威；
3. 声明 `.spec/<feature-name>/` 是唯一可执行功能来源；
4. 指向 Claude 专属适配器；
5. 明确 `docs/agent-harness` 和 `.ai-guides` 只作历史/参考。

不在根 `CLAUDE.md` 复制测试命令、分支规则、完成定义或 Hard Stop 列表。

`.claude/CLAUDE.md` 有两种可接受实现：

- 保留文件，但改成指向根 `CLAUDE.md` 和权威 Skill 的短适配说明；
- 若 Claude Code 当前版本不加载该路径，则保留 Legacy banner，不再作为规则入口。

最终选择需在实施 TASK 前依据本地 Claude Code 行为确认。

### 3.3 Claude Skill 和 Agent 适配

现有名称可暂时保留以兼容用户习惯，但内部流程必须变薄：

#### run-agent-task / task-developer

- 必须接收 `Feature` 和 `Task`；
- 读取权威 `spec-harness`；
- 只执行 `implement-task`；
- 不自行创建第二套 task branch、commit 或状态文件；
- 不写 Claude 私有完成状态。

#### review-agent-task / task-reviewer

- 读取同一 feature/TASK 的 SPEC、SDD、TASKS、scope、acceptance 和 report；
- 只读执行 `self-review` 协议；
- 输出权威 findings 和 `verdict: 通过|不通过`；
- 若通过 Claude subagent 实现独立 Review，在 evidence 中记录 reviewer 类型；
- 不再写 `docs/agent-harness/reviews/`。

#### fix-agent-task / task-fixer

- 只执行 `fix-check-failures`；
- 修复输入来自权威 self-review finding、Harness 或测试失败；
- 只更新当前 TASK report 和 evidence。

#### batch-agent-coordinator

- 只做 `harness-pipeline` 的 Claude 调用适配；
- 不复制 TASK 循环、Review 上限、merge 规则和状态迁移；
- 不直接执行 squash merge；
- 需要多 Agent 时由适配器把权威 mode 交给对应 subagent，但状态仍由 canonical pipeline 管理。

#### run-phase / phase agents

- 默认变为迁移门禁；
- 接收到 `.ai-guides` phase 时，查找显式关联的 `.spec` feature；
- 找不到时停止并提示运行 `spec-harness draft-spec-sdd`；
- 不自动把旧 spec/plan/tasks 当作已确认的新 SPEC/SDD；
- 是否继续注册该 command 由 Open Question 决定。

### 3.4 Legacy 标识设计

采用顶层标识，避免批量改写历史文件：

- `docs/agent-harness/README.md` 或等价入口：说明历史范围、已完成任务、pending 冻结规则和迁移入口；
- `.ai-guides/README.md`：区分参考指南、历史 delivery 和旧 phase contract，禁止直接实施；
- 必要时在 `docs/agent-harness/00-HARNESS.md` 顶部增加短 Legacy banner，但不重写正文；
- `.claude/agent-memory/` 不作为权威来源，适配器必须要求验证当前文件与代码。

Legacy 标识需要使用清晰文本，不能只依赖目录命名或隐含约定。

### 3.5 遗留迁移清单设计

迁移清单建议生成在：

```text
.spec/agent-harness-unification/reports/legacy-inventory.md
```

每个条目至少包含：

| 字段 | 含义 |
| --- | --- |
| Source | 原始任务、phase 或 feature 路径 |
| Legacy ID | P1-008、phase slug 等原编号 |
| Historical Status | 原状态，不改写 |
| Current Code Verified | 是否已核对当前代码 |
| Equivalent Spec | 是否已有等价 `.spec` |
| Migration Decision | archive / migrate-on-demand / separate-repair / needs-confirmation |
| Notes | 重复实现、依赖和风险 |

本清单只登记，不自动生成业务 feature。

### 3.6 Canonical Harness schema

为未来生成物引入显式 schema 版本。第一版严格 schema 使用：

```json
{
  "schemaVersion": 1,
  "feature_name": "feature-name",
  "tasks": {
    "TASK-001": {
      "title": "...",
      "status": "pending",
      "allowedFiles": [],
      "forbiddenFiles": [],
      "requiresHumanConfirmation": false,
      "allowedActions": ["modify", "create", "test"],
      "acceptance": ".spec/.../acceptance/TASK-001.md",
      "report": ".spec/.../reports/TASK-001-report.md",
      "notes": []
    }
  }
}
```

兼容策略：

- 无 `schemaVersion` 但符合当前 `feature_name + tasks` 结构的 feature 可识别为 legacy-compatible，并给出升级提示；
- 扁平 `{feature, allowedFiles}` 结构不允许进入 pipeline；
- validator 不自动改写任何现有 feature；
- TASK status 的唯一运行时来源应逐步收敛到 `pipeline-state.json`，`task-scope.json.status` 仅作为生成时初始值，避免双写；最终是否移除该字段需在 prepare-harness 前确认。

### 3.7 Pipeline state 设计

建议状态结构：

```json
{
  "schemaVersion": 1,
  "feature_name": "feature-name",
  "status": "in_progress",
  "current_task": "TASK-001",
  "current_phase": "review",
  "completed_tasks": [],
  "blocked_tasks": [],
  "failed_tasks": [],
  "review_round": 1,
  "task_runs": {
    "TASK-001": {
      "base_sha": "...",
      "head_sha": "...",
      "scope_status": "passed",
      "checks_status": "passed",
      "review_verdict": "通过",
      "human_confirmation": null,
      "evidence": ".spec/.../reports/TASK-001-evidence.json"
    }
  }
}
```

Pipeline status 候选值：

- `pending`
- `in_progress`
- `blocked`
- `completed`
- `completed_with_exceptions`（名称待确认）

状态不变量：

1. `completed` 要求全部 TASK 的 scope、checks、review 和 confirmation 通过；
2. `failed_tasks` 或 `blocked_tasks` 非空时不能写 `completed`；
3. `completed_with_exceptions` 必须为每个例外记录人工批准；
4. 缺少 evidence 的 TASK 不能进入 completed_tasks；
5. pipeline state 更新必须在对应检查之后发生。

### 3.8 TASK 基线与 scope 检查

任务开始时记录 `base_sha`。change set 按以下优先级计算：

1. 已提交变更：`base_sha..HEAD`；
2. unstaged：working tree 相对 index；
3. staged：index 相对 HEAD；
4. untracked：`git ls-files --others --exclude-standard`。

合并后去重，再与当前 TASK allow/forbid patterns 比较。

为避免前序 TASK 污染：

- 每个 TASK 在开始时刷新自己的 `base_sha`；
- 前序 TASK 必须完成并形成可识别 HEAD 或等价 checkpoint；
- 如果当前运行环境不允许 commit，pipeline 必须保存已确认的前序文件集合并从当前 TASK change set 排除，且该降级策略必须写入 evidence；
- 无法获得可靠基线时 Hard Stop，不允许退化成人工目测后继续。

Pattern matcher 必须统一定义 `*`、`**` 和精确路径语义，禁止每个 feature 生成不同实现。

### 3.9 机器证据设计

每个 TASK 新增：

```text
.spec/<feature>/reports/<TASK-ID>-evidence.json
```

建议结构：

```json
{
  "schemaVersion": 1,
  "feature_name": "...",
  "task_id": "TASK-001",
  "base_sha": "...",
  "head_sha": "...",
  "changed_files": [],
  "scope": {
    "status": "passed",
    "out_of_scope": [],
    "forbidden": []
  },
  "checks": [
    {
      "command": "...",
      "exit_code": 0,
      "started_at": "RFC3339",
      "duration_ms": 0,
      "status": "passed"
    }
  ],
  "review": {
    "reviewer_type": "independent-agent",
    "verdict": "通过",
    "round": 1
  },
  "human_confirmation": null,
  "exceptions": []
}
```

Markdown TASK report 继续服务人类阅读，但关键状态必须从 evidence 或同一次检查结果生成，不能手工把失败改写为通过。

### 3.10 共享 validator 与脚本生成

当前每个 feature 都复制 `check-task-scope.sh`，容易产生实现漂移。推荐分两步收敛：

1. 在权威 `.agents/skills/spec-harness/` 下提供共享 validator/checker 脚本；
2. 新 feature 的 `scripts/check-task-scope.sh` 和 `agent-check.sh` 作为稳定入口，只传递 feature/TASK 参数或运行 TASK 特定检查。

共享 checker 修改属于共享工作流变更，实施 TASK 必须设置 `requiresHumanConfirmation: true`。

不引入新依赖，JSON 处理优先使用现有 Node 运行时；Shell 入口保留，以兼容 `AGENTS.md` 中的既有命令。

## 4. Data Structure Changes

本功能不修改业务数据库或业务数据结构，只修改 Agent 治理文件结构。

计划新增或规范化：

1. `task-scope.json.schemaVersion`；
2. `pipeline-state.json.schemaVersion`、`current_phase`、`task_runs`；
3. TASK 运行状态枚举和 pipeline 状态不变量；
4. `<TASK-ID>-evidence.json`；
5. `legacy-inventory.md`。

历史 feature 不自动重写。Validator 负责分类：current、legacy-compatible、unsupported。

## 5. API and Interface Changes

### 5.1 不涉及的 API

- 无 HTTP、gRPC、SSE 或数据库 API 变化；
- 无前端 route、component 或 public type 变化；
- 无 Go public API 变化。

### 5.2 Agent 工作流接口

Claude 适配器的逻辑输入统一为：

```text
Mode: <canonical-mode>
Feature: <feature-name>
Task: <TASK-ID, if required>
```

Legacy P 编号或 phase slug 不再直接映射到实现操作，只映射到迁移提示或明确关联的 `.spec` feature。

### 5.3 CLI/脚本接口

保留：

```bash
bash .spec/<feature>/scripts/check-task-scope.sh <TASK-ID>
bash .spec/<feature>/scripts/agent-check.sh
```

可新增只读 preflight/validation 入口，但不得要求修改 `package.json`。

## 6. Algorithm or Workflow Changes

### 6.1 新功能流程

```text
用户需求
  → 根 Agent 指令发现
  → spec-harness draft-spec-sdd 或 init-feature
  → 用户确认 SPEC/SDD
  → prepare-harness
  → preflight schema/structure validation
  → implement-task
  → scope + checks + evidence
  → independent review when available / disclosed self-review fallback
  → fix-check-failures loop
  → truthful pipeline state
  → 人工合入
```

### 6.2 遗留调用流程

```text
Legacy task/phase input
  → classify source
  → lookup explicit equivalent .spec
  ├─ found and valid → use canonical feature
  └─ missing/invalid → stop implementation
                     → draft new SPEC/SDD migration
                     → user confirmation
```

### 6.3 Pipeline 完成判定

伪代码：

```text
if any required task is blocked or failed:
    status = blocked
else if any required evidence is missing:
    status = blocked
else if any scope/check/review/confirmation is not passed:
    status = blocked
else if approved exceptions exist:
    status = completed_with_exceptions
else:
    status = completed
```

`skip_human_confirm=true` 不得把被跳过的必需 TASK 当作完成；pipeline 应保持 blocked/partial 语义，具体状态名称需与权威 Skill 一并确认。

## 7. Configuration Design

本功能不增加业务运行时配置。

Agent 配置原则：

- 权威规则存在仓库文件中；
- Claude provider-specific 配置仅负责入口和适配；
- 不修改 `.claude/settings.local.json`；
- 不新增全局 Codex 或 Claude 用户配置；
- 不硬编码用户绝对路径；
- 不要求启用新的 MCP server、Hook 或外部服务。

## 8. Compatibility Strategy

### 8.1 现有 `.spec`

- 完成的 feature 保持只读兼容；
- 无 schemaVersion 但结构符合 `feature_name + tasks` 的 feature 由 validator 标记 legacy-compatible；
- unsupported feature 被阻止并给出修复路径；
- 不批量修改历史 report 或 pipeline state。

### 8.2 Claude 旧命令

- 优先保留常用名称作为薄适配器，降低使用中断；
- 适配器接收到旧参数时只做迁移提示；
- 不再维护 `EXECUTION_LOG.md` 或旧 review 文件。

### 8.3 历史文档

- `docs/agent-harness` 保留 13 个完成/合并任务历史；
- 11 个 pending 条目保持原始状态并在迁移清单中登记；
- `.ai-guides` 保留设计和交付证据；
- 新 Legacy banner 不改变历史内容含义。

### 8.4 分支与工作区

本功能不把旧 `integration/agent-platform` 规则提升为全仓库权威。TASK 隔离通过可靠 Git 基线、当前环境的 branch/worktree 能力和 task scope 共同保证。是否强制每 TASK 一个 worktree 作为待确认策略保留。

## 9. Error Handling and Fallback Design

| 情况 | 行为 |
| --- | --- |
| 根 `AGENTS.md` 或权威 Skill 缺失 | Hard Stop，报告缺失路径 |
| feature 文件不完整 | Preflight 非零退出并列出缺失文件 |
| unsupported task-scope schema | 阻止执行，提示重新 prepare-harness 或单独修复 |
| unknown TASK ID | 非零退出，不选择相似 TASK |
| 无可靠 TASK Git 基线 | Hard Stop，不降级为人工目测 |
| out-of-scope/forbidden file | scope 失败，pipeline blocked |
| required check 失败 | evidence 记录退出码，pipeline blocked |
| Review 不通过 | 进入受限 fix；超过上限 blocked |
| 需要人工确认但未确认 | 停止，不标记完成 |
| Claude 无 subagent | 使用 self-review，并记录 reviewer_type |
| Legacy 任务无 `.spec` | 只允许迁移到 draft-spec-sdd |
| 历史文档与当前代码冲突 | 当前代码优先，冲突写入新 SPEC open question |

## 10. Observability and Debug Output Design

### 10.1 Preflight 输出

```text
feature: agent-harness-unification
task: TASK-...
canonical_skill: .agents/skills/spec-harness/SKILL.md
scope_schema: current | legacy-compatible | unsupported
pipeline_schema: current | missing | unsupported
base_sha: ...
result: PASS | FAIL
```

### 10.2 Scope 输出

每个文件输出分类：allowed、forbidden、out-of-scope，并显示命中的 pattern。失败时返回非零退出码。

### 10.3 Pipeline 输出

状态更新至少包含：TASK、phase、review round、evidence path、阻塞原因。最终 summary 从 state/evidence 汇总，不能只依据 Agent 自述。

### 10.4 适配器诊断

Claude 拒绝旧入口时输出：

- 发现的 legacy path；
- 权威入口；
- 是否存在等价 `.spec`；
- 推荐的下一 canonical mode。

## 11. Testing Strategy

### 11.1 文档和结构检查

- 验证 SPEC/SDD 必需章节存在；
- 验证根 `CLAUDE.md` 能定位 `AGENTS.md` 和权威 Skill；
- 验证 Legacy banner 存在；
- 验证没有业务文件进入 diff。

### 11.2 静态引用检查

使用 `rg` 检查可执行 Claude 配置：

- 不得继续声明 `docs/agent-harness` 为 authoritative；
- 不得直接从 `.ai-guides/*/tasks.md` 实施；
- 不得硬编码 `integration/agent-platform` 作为统一流程前提；
- 必须引用 `.agents/skills` 和 `.spec`。

Legacy banner、历史 memory 和归档文档中的旧文本允许存在，应通过路径白名单区分。

### 11.3 JSON schema/semantic checks

使用仓库已有 Node 运行时验证：

- 有效 current schema；
- legacy-compatible schema；
- unsupported 扁平 schema；
- unknown TASK；
- 无效 pipeline 状态组合；
- `completed` 与失败 evidence 冲突。

### 11.4 Scope checker tests

在受控临时 fixture 或本 feature 的测试脚本中覆盖：

1. allowed tracked file；
2. forbidden tracked file；
3. out-of-scope untracked file；
4. staged 越界文件；
5. 前一 TASK 已完成变更不污染后一 TASK；
6. 无 base_sha；
7. `*`、`**` 和精确路径 pattern。

测试不得修改或清理用户现有工作树。

### 11.5 Pipeline invariant tests

覆盖：

- scope failed + report exists 不能 completed；
- tests failed 不能 completed；
- review 不通过不能 completed；
- human confirmation missing 不能 completed；
- approved exception 使用区别状态；
- 全部通过可 completed。

### 11.6 Adapter smoke tests

对每个 Claude 适配入口验证 mode mapping 和只读/可写边界。测试不要求实际调用外部模型，可通过静态 prompt contract 和模拟输入完成。

### 11.7 不需要的业务测试

由于不修改业务代码，本功能不默认运行全部 Go/Vue 测试。若某 TASK 实际触碰共享脚本之外的业务文件，应视为 scope 违规而不是扩大测试范围。

## 12. Migration Risks

1. **pending 遗漏**：Legacy banner 生效后，11 个旧 pending 任务可能被误认为取消；迁移清单必须明确冻结不等于取消。
2. **重复实现**：旧任务可能已由其他 `.spec` 或后续代码部分实现；迁移前必须检查当前代码。
3. **Claude 入口兼容**：不同 Claude Code 版本对 `CLAUDE.md`、Skill 和 command 的发现规则可能不同，实施时需本地 smoke check。
4. **共享 Skill 影响面**：修改 `.agents/skills/spec-harness` 或 `harness-pipeline` 会影响全部未来 feature，必须拆成独立高风险 TASK 并人工确认。
5. **历史 feature 误阻断**：严格 validator 可能阻止旧但仍可用的 feature；需要 current / legacy-compatible / unsupported 分类。
6. **Git 基线差异**：Codex desktop、Claude Code worktree 和普通本地分支行为不同；不能假设所有环境都会为每 TASK commit。
7. **状态迁移复杂度**：task-scope status 与 pipeline state 双写可能导致旧数据冲突；validator 需要明确优先级。
8. **过度治理**：机器证据和 schema 若设计过重，会增加简单任务成本；实现应保持生成和消费自动化。
9. **历史敏感信息**：Legacy 文档和 memory 可能含绝对路径或旧环境信息；本功能不批量清理，但新适配器不能复制这些内容。

## 13. Implementation Boundaries

### 13.1 允许进入后续 TASK 规划的路径

- `AGENTS.md`
- `CLAUDE.md`
- `.agents/skills/spec-harness/**`
- `.agents/skills/harness-pipeline/**`
- `.claude/CLAUDE.md`
- `.claude/skills/**`
- `.claude/agents/**`
- `.claude/commands/**`
- `docs/agent-harness/README.md`
- `docs/agent-harness/00-HARNESS.md` 的顶层 Legacy 标识
- `.ai-guides/README.md`
- `.spec/agent-harness-unification/**`

其中 `AGENTS.md`、`.agents/skills/**`、共享 Claude 入口属于全局/共享控制面，相关 TASK 必须 `requiresHumanConfirmation: true`。

### 13.2 禁止修改

- `hr-frontend/**`
- `user-frontend/**`
- `interviewer-frontend/**`
- `logic-grpc-service/**`
- `web-gin-service/**`
- `db.sql`
- `deploy/**`
- `docker/**`
- `.github/**`
- `package.json`
- `pnpm-lock.yaml`
- `pnpm-workspace.yaml`
- `**/go.mod`
- `**/go.sum`
- 既有历史 task、review、ADR、delivery report 和已完成 `.spec` 内容，除非单独 TASK 明确只添加非破坏性标识并经确认。

### 13.3 TASK 拆分原则

后续 `prepare-harness` 应至少把以下高耦合事项拆开：

1. 权威 schema、validator 和 pipeline invariant；
2. Claude 根入口与 provider adapter；
3. Legacy 标识与迁移清单；
4. 兼容性/回归检查；
5. 不合规 pending `.spec` 的处理决定。

不得在同一个 TASK 中同时修改共享 Skill、所有 Claude adapter 和全部 Legacy 文档。

## 14. Alternatives Considered

### 14.1 保留三套体系并用文档说明优先级

拒绝。规则仍会复制和漂移，Agent 在局部上下文中仍可能只读取非权威入口。

### 14.2 立即删除 `.claude`、`docs/agent-harness` 和 `.ai-guides`

拒绝。会破坏 Claude Code 使用体验并丢失历史、ADR、Review 和交付证据。

### 14.3 让 `.claude` 体系保持权威，Codex 适配它

拒绝。与用户明确确认及根 `AGENTS.md` 的 `.spec` 工作流冲突。

### 14.4 用符号链接让 Claude Skill 直接指向 `.agents` Skill

暂不采用为默认方案。符号链接简洁，但跨平台、Git 配置和工具发现兼容性不稳定。优先使用短 wrapper，并通过检查防止 wrapper 复制核心规则。

### 14.5 批量自动转换所有旧任务

拒绝。旧任务可能已过期、部分实现或与当前代码冲突，机械转换会把陈旧假设重新提升为权威需求。

### 14.6 继续使用 Markdown 报告而不增加机器证据

拒绝。现有记录已证明自然语言报告与 pipeline state 可以相互矛盾，无法可靠执行失败关闭。

### 14.7 强制每个 TASK 都使用独立 worktree

暂不作为已确认设计。Worktree 隔离强，但不同 Agent 产品的运行环境和当前 Codex desktop 工作区能力不同。设计先要求可靠 TASK 基线；是否进一步强制 worktree 保留为用户决策。

## 15. Assumptions Requiring Confirmation

1. 根 `CLAUDE.md` 使用 `AGENTS.md` 导入或强引用方式，并作为 Claude Code 的唯一项目级起点。
2. 保留 `.claude/skills` 常用名称作为薄 wrapper，以减少使用迁移成本。
3. `docs/agent-harness` 的 completed/merged 历史不迁移，pending 条目冻结并按需迁移。
4. `.ai-guides` 默认只作历史与参考；未完成 phase 不自动视为 active。
5. 新 schema 使用 `schemaVersion: 1`，无版本但结构正确的 feature 归类为 legacy-compatible。
6. 机器 evidence 使用 JSON 并存放于 feature reports，不引入数据库。
7. 共享 scope validator 放在 `.agents/skills/spec-harness/` 内，feature 脚本作为稳定 wrapper。
8. 独立 Reviewer 是优先策略，不是所有环境的硬依赖；降级为 self-review 时必须披露。
9. `completed_with_exceptions` 作为候选状态，最终名称和允许条件需用户确认。

## 16. Open Questions

1. 旧执行日志中的 11 个 pending 任务采用按需迁移还是一次性迁移草案？
2. 哪些 `.ai-guides` phase 仍然活跃，需要纳入迁移清单的优先队列？
3. `semantic-retrieval-score-fixes` 的 Harness 缺口由本 feature 修复还是另立 feature？
4. Pipeline 是否允许 `completed_with_exceptions`；若允许，批准人、批准时间和原因采用什么字段？
5. 是否移除 `task-scope.json.tasks[*].status`，只保留 `pipeline-state.json` 作为运行时状态源？
6. 是否强制每个 implementation TASK 使用独立 worktree/commit，还是保留可靠基线的多环境实现？
7. `.claude/commands/run-phase.md` 是保留为迁移门禁，还是直接 deprecated？
8. `.claude/agent-memory` 是保留为非权威上下文，还是禁止新 Agent 自动读取？
9. 共享 validator 是否需要同时提供单独的只读 audit 命令，用于扫描所有 `.spec` feature 的合规状态？
