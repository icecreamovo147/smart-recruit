# Prompt - fix-check-failures

Use `spec-harness`.

```text
Mode: fix-check-failures
Feature: agent-harness-unification
Task: <TASK-ID>
```

修复前完整读取：

1. SPEC、SDD、TASKS、AGENT_RULES
2. `task-scope.json`
3. 对应 acceptance
4. TASK report 和 evidence
5. 上一轮 Harness/test/self-review 的具体失败输出

只允许修复上一轮明确列出的失败：

- 不新增功能；
- 不扩大 TASK scope；
- 不修改 SPEC/SDD；
- 不修改与 finding 无关的共享文件；
- 不通过放宽规则、删除失败证据或改写历史来规避检查；
- 如果修复需要其他 TASK 的文件，停止并返回 owning TASK；
- 同一问题连续两次失败时停止并说明根因。

修复后重新运行：

```bash
git diff --name-only
TASK_BASE_SHA=<original-task-start-sha> bash .spec/agent-harness-unification/scripts/check-task-scope.sh <TASK-ID>
bash .spec/agent-harness-unification/scripts/agent-check.sh
```

并重新运行原失败命令。更新 TASK report 的 Repair Summary 和 evidence，保留原失败与重跑结果。完成后停止，等待下一轮 self-review。
