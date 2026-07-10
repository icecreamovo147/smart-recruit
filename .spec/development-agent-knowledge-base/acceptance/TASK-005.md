# Acceptance - TASK-005

## TASK Summary

在 `AGENTS.md` 中接入统一的知识读取、维护、影响检查、scope 和安全协议。

## SPEC References

- FR-002、FR-005 至 FR-008
- FR-012
- AC-003、AC-005、AC-006、AC-011、AC-014

## SDD References

- Section 3.3, Trust Zones
- Section 6.1 and 6.2, Agent Workflows
- Section 8, Compatibility Strategy
- Section 13, TASK-005

## Acceptance Criteria

- 实施前已记录明确人工确认。
- `AGENTS.md` 声明 `.knowledge` 是 canonical 控制面下游。
- 非简单 TASK 强制进行知识影响检查，但不要求每次都修改知识。
- 明确固定 verdict、L1/L2/L3、Inbox、冲突和 scope 外债务处理。
- 普通知识不能覆盖 AGENTS、当前 SPEC、代码或测试。
- Provider adapter 仍通过 canonical 入口使用同一知识库，无规则副本。
- 不修改业务代码、共享 Harness 或 CI。

## Required Checks

- `git diff --check`
- 知识工具测试、结构校验和引用检查。
- `rg -n "\.knowledge|AGENTS\.md|spec-harness" CLAUDE.md .claude/CLAUDE.md`
- Harness scope 与 agent check。

## Manual Verification, if needed

- Reviewer 检查 AGENTS 与 `.knowledge/README.md` 权威顺序一致且负担可控。

## Out-of-Scope

- 修改 Provider adapters。
- 修改 spec-harness 或 harness-pipeline。
- CI 和业务代码。
