# Agent Rules - agent-harness-unification

## 1. Authority

本 feature 的执行权威按以下优先级排列：

1. 用户当前明确指令；
2. 根 `AGENTS.md`；
3. `.agents/skills/spec-harness/SKILL.md`；
4. `.agents/skills/harness-pipeline/SKILL.md`（仅 pipeline 调用）；
5. 本 feature 的 SPEC、SDD、TASKS、task-scope、acceptance 和本文件。

`docs/agent-harness/`、`.ai-guides/`、`.claude/agent-memory/` 不是实现权威。

## 2. Confirmed Decisions

- 遗留 pending 任务按需迁移，不批量实施。
- `.ai-guides` 默认历史/参考。
- `semantic-retrieval-score-fixes` 另立 Harness 修复 feature。
- `completed_with_exceptions` 只允许携带明确人工批准。
- 不强制 worktree，但 TASK 必须有可靠 Git 基线。
- `run-phase` 保留为迁移门禁。
- Claude memory 保留但非权威。
- `pipeline-state.json` 是运行时状态源。

## 3. Execution Rules

1. 每次只执行一个 TASK。
2. 执行前必须读取 SPEC、SDD、TASKS、AGENT_RULES、task-scope、对应 acceptance 和 prompt。
3. `requiresHumanConfirmation: true` 的 TASK 未获得用户明确确认时不得修改文件。
4. 不得修改当前 TASK `allowedFiles` 之外的文件。
5. 不得修改本 feature 的 SPEC 或 SDD。
6. 不得把后续 TASK 内容提前合并到当前 TASK。
7. 不得顺手清理历史文件、Agent memory、本地配置或业务代码。
8. 不得新增依赖、修改 manifest、lockfile、Go module、部署或 CI。
9. 不得自动 commit、push、merge 或创建 PR。
10. Fix 模式只能修复上一轮明确 finding，不得扩大 scope。

## 4. Git Baseline Rules

1. 每个 implementation TASK 开始前记录 `base_sha`。
2. 前一 TASK 的变更必须已形成可识别 checkpoint，或由 canonical pipeline 记录等价基线。
3. scope 检查必须覆盖 committed、staged、unstaged 和 untracked 变更。
4. 无可靠基线时 Hard Stop；不得退化为人工目测后继续。
5. 不得通过清理、覆盖或丢弃用户变更来获得干净基线。

## 5. State and Evidence Rules

1. 每个 TASK 必须创建或更新 Markdown report。
2. 实现后的 TASK 必须生成机器可读 evidence，记录 Git 标识、变更文件、scope、checks、Review 和 confirmation。
3. 失败的命令必须保留真实退出码。
4. scope/check/review/confirmation 任一必需条件失败时，不得标记普通 completed。
5. `completed_with_exceptions` 必须记录批准对象、批准人、批准时间和原因。
6. `task-scope.json.tasks[*].status` 只表示生成时初始状态；运行时以 `pipeline-state.json` 为准。

## 6. Review Rules

1. Reviewer 只读，不得修改任何文件。
2. 优先使用独立 Agent 或新上下文；无法使用时必须在报告中披露 self-review。
3. Review 必须对照 SPEC、SDD、TASKS、scope、acceptance、report、evidence 和实际 diff。
4. 最终 verdict 只能是 `verdict: 通过` 或 `verdict: 不通过`。
5. 未验证的 Agent 自述不能作为通过证据。

## 7. Legacy Safety Rules

1. 不得删除或批量改写 `docs/agent-harness`、`.ai-guides`、`.claude/agent-memory` 历史。
2. 冻结不等于取消；pending 状态必须保留原始含义。
3. 遗留任务没有新的 `.spec` 时不得直接实施。
4. 迁移清单只登记，不得自动创建业务任务或声称当前代码已验证。
5. 历史绝对路径、旧分支和旧结论不得复制到新权威配置。

## 8. Testing Rules

每个 TASK 完成后必须运行：

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh <TASK-ID>
bash .spec/agent-harness-unification/scripts/agent-check.sh
```

并运行 acceptance 中的 TASK-specific checks。

本 feature 不修改业务代码，因此不默认运行 Go/Vue 全量测试。若出现业务文件 diff，应判定 scope 失败，而不是扩大测试范围。

## 9. Hard Stop Conditions

出现以下任一情况必须停止并请求确认：

- 需要修改 SPEC 或 SDD；
- 需要修改 TASK scope 外文件；
- 需要删除历史文件；
- 需要修改 `.claude/settings.local.json`、全局 Codex/Claude 配置或权限；
- 需要修改业务代码、数据库、Proto、manifest、lockfile、Go module、部署或 CI；
- 无法获得可靠 Git 基线；
- 发现 SPEC、SDD、TASKS 或 acceptance 冲突；
- 共享 Skill 修改超出当前已确认 TASK；
- 需要自动迁移不明确是否活跃的 legacy 任务；
- 检查脚本无法安全判断 scope；
- 同一问题连续两次修复失败。

## 10. Report Requirements

TASK report 必须包含：

- TASK ID 和标题；
- 实际修改文件；
- 每个文件的变更摘要；
- scope 检查结果；
- SPEC/SDD/acceptance 对照；
- 执行命令和真实结果；
- evidence 路径；
- Review 类型与 verdict；
- 风险和遗留项；
- 是否允许进入下一 TASK。

超范围或失败时必须明确写为失败/阻塞，不得用自然语言淡化。
