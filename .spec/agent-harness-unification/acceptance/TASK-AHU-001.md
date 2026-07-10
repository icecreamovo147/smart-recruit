# Acceptance - TASK-AHU-001

## TASK Summary

固化 `.agents/.spec` 唯一权威关系，并在仓库规则与 `spec-harness` 中定义 canonical schema、evidence、Review 和状态来源契约。

## SPEC References

- Goals 1、2、6、7
- FR-001、FR-007、FR-009、FR-010、FR-011
- AC-001、AC-006、AC-010、AC-011、AC-012

## SDD References

- 3.1 分层控制面
- 3.6 Canonical Harness schema
- 3.7 Pipeline state 设计
- 3.9 机器证据设计
- 13.1 允许进入后续 TASK 规划的路径

## Acceptance Criteria

- [ ] `AGENTS.md` 明确 `AGENTS.md → .agents/skills → .spec` 的权威链路。
- [ ] `spec-harness` 保留六个既有 mode 名称和 staged workflow。
- [ ] `spec-harness` 定义 schemaVersion、current/legacy-compatible/unsupported 分类。
- [ ] `spec-harness` 定义 TASK evidence 的必需字段和失败真实性。
- [ ] `spec-harness` 明确 preflight 失败关闭。
- [ ] 独立 Review 优先，self-review 降级必须披露。
- [ ] `pipeline-state.json` 被定义为运行时状态源。
- [ ] 未实现 TASK-AHU-002/003 的脚本或 pipeline 逻辑。
- [ ] 无绝对用户路径、无业务或依赖变更。

## Required Checks

```bash
git diff --name-only
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-001
bash .spec/agent-harness-unification/scripts/agent-check.sh
rg -n "init-feature|draft-spec-sdd|prepare-harness|implement-task|self-review|fix-check-failures" .agents/skills/spec-harness/SKILL.md
```

## Manual Verification, if needed

- 人工确认新增权威声明没有把 provider-specific 细节提升为仓库级规则。
- 人工确认 shared Skill 改动符合当前 Codex 与 Claude 使用预期。

## Out-of-Scope

- 共享 validator/checker 实现；
- `harness-pipeline` 状态逻辑；
- Claude adapter 和 Legacy 文档；
- 业务代码和依赖。
