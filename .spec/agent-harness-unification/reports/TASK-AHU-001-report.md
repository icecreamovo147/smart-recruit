# TASK Report - TASK-AHU-001

## 1. TASK ID

TASK-AHU-001 — 固化唯一权威来源与 Canonical Harness 契约

## 2. Modified File List

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/agent-harness-unification/pipeline-state.json`
- `.spec/agent-harness-unification/reports/TASK-AHU-001-report.md`
- `.spec/agent-harness-unification/reports/TASK-AHU-001-evidence.json`

说明：本 feature 的 SPEC/SDD/Harness 文件在 TASK 开始前已作为相关 untracked 准备产物存在，因此 Git change set 会同时列出这些文件；本 TASK 没有修改 SPEC 或 SDD。

## 3. Change Summary by File

- `AGENTS.md`：新增唯一权威链路，明确仓库规则、两个 canonical Skill、`.spec` feature contract 和 runtime state 的职责边界。
- `.agents/skills/spec-harness/SKILL.md`：新增 canonical authority、schema v1、feature 分类、runtime state ownership、TASK evidence、Review 降级披露和 Legacy migration gate；补充 implement-task preflight 和 evidence 输出要求。
- `pipeline-state.json`：记录 TASK 基线、人工确认和当前 implement phase。
- TASK report/evidence：记录真实修改、检查结果、风险和 Review 状态。

## 4. Scope Check Result

通过。

```bash
TASK_BASE_SHA=7587e238855bb8159377f99cb1f53b9358ae67b2 \
  bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-001
```

Scope checker 将 `AGENTS.md`、`spec-harness/SKILL.md` 和当前 feature 目录全部判定为 allowed；没有 forbidden 或 out-of-scope 文件。

## 5. SPEC Comparison Result

符合 FR-001、FR-007、FR-009、FR-010、FR-011。没有实施共享 validator、pipeline 状态脚本、Claude adapter 或 Legacy banner。

## 6. SDD Comparison Result

符合 SDD 3.1、3.6、3.7、3.9 和 13.1。本 TASK 只定义 canonical contract；脚本实现仍归 TASK-AHU-002/003。

## 7. Acceptance Comparison Result

- 权威链路：通过。
- 六个 mode 保持：通过。
- schemaVersion/current/legacy-compatible/unsupported：通过。
- TASK evidence 和失败真实性：通过。
- preflight 失败关闭：通过。
- Review 降级披露：通过。
- runtime state ownership：通过。
- 无绝对路径、业务或依赖变更：通过。

## 8. Test Commands and Results

- `git diff --name-only`：通过；仅列出 `AGENTS.md` 和 `spec-harness/SKILL.md` 两个 tracked 修改。
- `check-task-scope.sh TASK-AHU-001`：通过。
- `agent-check.sh`：通过，治理边界内共 23 个最终 change-set 文件。
- 六个 mode 名称检查：通过。
- `rg '/Users/' AGENTS.md .agents/skills/spec-harness/SKILL.md`：零命中，通过。
- Go/Vue 测试：未运行；本 TASK 不修改业务代码。

## 9. Risks

- `spec-harness` 是共享 Skill，后续 feature 都会读取新增契约。
- 当前准备产物未形成 Git checkpoint；进入 TASK-AHU-002 前需要可靠 checkpoint 或 canonical 等价基线，否则前序 tracked 变更会污染下一 TASK scope。
- 本轮 Review 由同一 Agent 在独立只读阶段执行，必须披露为 `reviewer_type: self-review`。

## 10. Follow-up Items

- 运行只读 self-review 并把 verdict 写入 evidence/state。
- 在 TASK-AHU-002 前解决前序 TASK checkpoint 问题。

## 11. Whether the Next TASK Can Start

只有在本 TASK Review 通过、state/evidence 更新完成，并建立可区分前序改动的可靠 checkpoint 后，TASK-AHU-002 才能开始。

## 12. Self-Review Result

- reviewer_type: `self-review`
- review_round: 1
- Critical findings: 无
- High findings: 无
- Medium findings: 无
- Low findings: 无

```text
verdict: 通过
```
