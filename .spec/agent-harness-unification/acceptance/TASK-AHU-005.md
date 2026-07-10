# Acceptance - TASK-AHU-005

## TASK Summary

将 Claude batch coordinator 收敛为 `harness-pipeline` 适配器，并将 run-phase/phase agents 改成 legacy-to-spec 迁移门禁。

## SPEC References

- FR-003、FR-006、FR-013
- AC-003、AC-010、AC-013
- Error Handling Requirements 4、5、6

## SDD References

- 3.3 batch-agent-coordinator
- 3.3 run-phase / phase agents
- 6.2 遗留调用流程
- 8.2 Claude 旧命令

## Acceptance Criteria

- [ ] batch coordinator 只引用 `harness-pipeline`，不复制循环、状态和修复上限。
- [ ] 不再硬编码 integration branch、squash merge 或两轮 Fix。
- [ ] run-phase 保留名称但不直接实施 `.ai-guides`。
- [ ] 无显式有效 `.spec` 映射时停止并输出迁移指引。
- [ ] 有有效映射时只调用 canonical mode。
- [ ] phase reviewer 不产生第三套 verdict/状态协议。
- [ ] `.ai-guides`、task adapters、Agent memory 和本地权限未修改。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-005
bash .spec/agent-harness-unification/scripts/agent-check.sh
rg -n "harness-pipeline|spec-harness|\.spec/" .claude/skills/batch-agent-coordinator/SKILL.md .claude/commands/run-phase.md .claude/agents/phase-*.md
```

模拟检查：

- legacy phase 无 mapping → 阻止实施并建议 `draft-spec-sdd`；
- valid `.spec` mapping → 调用 canonical mode；
- invalid `.spec` mapping → preflight 失败。

## Manual Verification, if needed

- 核对保留 `/run-phase` 不会误导用户认为旧 phase 可直接执行。

## Out-of-Scope

- 自动创建或迁移业务 `.spec`；
- 修改 `.ai-guides`；
- 修改单 TASK adapters；
- Git merge/push 自动化。
