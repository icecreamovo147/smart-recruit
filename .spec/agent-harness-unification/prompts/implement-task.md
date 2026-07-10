# Prompt - implement-task

Use `spec-harness`.

```text
Mode: implement-task
Feature: agent-harness-unification
Task: <TASK-ID>
```

执行前必须完整读取：

1. `.spec/agent-harness-unification/agent-harness-unification-SPEC.md`
2. `.spec/agent-harness-unification/agent-harness-unification-SDD.md`
3. `.spec/agent-harness-unification/TASKS.md`
4. `.spec/agent-harness-unification/AGENT_RULES.md`
5. `.spec/agent-harness-unification/task-scope.json`
6. `.spec/agent-harness-unification/acceptance/<TASK-ID>.md`
7. 本 prompt

执行要求：

- 只执行指定 TASK，不得提前执行后续 TASK。
- 若 `requiresHumanConfirmation` 为 true，先确认当前对话中已有针对该 TASK 的明确授权；没有则停止。
- 开始修改前记录可靠 `base_sha`，并导出 `TASK_BASE_SHA` 供 scope checker 使用。
- 只修改当前 TASK `allowedFiles`。
- 不修改 SPEC 或 SDD。
- 不修改业务代码、依赖、部署、CI、本地权限、Agent memory 或未允许历史文件。
- 不自动 commit、push、merge 或创建 PR。
- 共享 Skill 或 Claude 入口的修改必须保持向后兼容，并避免复制权威规则。
- 所有失败命令保留真实退出码，不得把失败写成成功。

完成实现后运行：

```bash
git diff --name-only
TASK_BASE_SHA=<task-start-sha> bash .spec/agent-harness-unification/scripts/check-task-scope.sh <TASK-ID>
bash .spec/agent-harness-unification/scripts/agent-check.sh
```

再运行 acceptance 文件中的 TASK-specific checks。

最后创建或更新：

```text
.spec/agent-harness-unification/reports/<TASK-ID>-report.md
.spec/agent-harness-unification/reports/<TASK-ID>-evidence.json
```

Report 和 evidence 必须一致。完成后停止，不进入下一 TASK。

Hard Stop：需要扩大 scope、修改共享但未确认的文件、无法获得可靠基线、SPEC/SDD/TASKS 冲突、检查无法安全判断或需要业务变更时，立即停止并报告。
