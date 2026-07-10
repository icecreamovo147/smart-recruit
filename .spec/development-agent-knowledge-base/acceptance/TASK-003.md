# Acceptance - TASK-003

## TASK Summary

创建首批系统架构、Agent Runtime、语义召回和核心领域知识，并建立对应路由。

## SPEC References

- FR-003 至 FR-005
- FR-013
- AC-002、AC-003、AC-009、AC-013、AC-014

## SDD References

- Section 1, Existing Architecture
- Section 3, Proposed Design
- Section 11.3, Content Acceptance
- Section 13, TASK-003

## Acceptance Criteria

- 创建 SPEC/TASKS 指定的 7 篇 architecture/domain 文档。
- 每篇文档有唯一 ID、active 状态、稳定 owner、tags、applies_to、source_refs、last_verified 和 review_after。
- 所有关键结论已对照当前代码、README、有效 SPEC 或测试核验。
- 不把历史 `.ai-guides` 或过时 SPEC 直接提升为事实。
- INDEX 与 manifest 能将 Agent Skill、Memory、Embedding 和系统模块任务路由到相关知识。
- 文档聚焦设计原因、边界、入口、影响和陷阱，不复制大段代码。

## Required Checks

- `node .knowledge/scripts/knowledge-validator.test.mjs`
- `node .knowledge/scripts/validate-knowledge.mjs --root .`
- `node .knowledge/scripts/check-references.mjs --root .`
- 对代表性路径运行 `detect-impact` 的测试/fixture。
- `git diff --check`
- Harness scope 与 agent check。

## Manual Verification, if needed

- Reviewer 逐篇抽查 `source_refs` 与当前实现一致。
- 确认 owner taxonomy 已获确认或在报告中记录假设。

## Out-of-Scope

- Runbook 和 Pitfall。
- 修改任何被文档描述的业务代码。
- AGENTS、Harness 和 CI。
