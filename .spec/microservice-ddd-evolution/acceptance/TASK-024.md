# Acceptance - TASK-024

## TASK Summary

迁移 AI Agent MCP/skill/embedding/intelligence。

## SPEC References

- FR-014
- SSR-005
- ARC-015

## SDD References

- 3.4 Per-Service Migration Pattern
- 9. Error Handling and Fallback Design

## Acceptance Criteria

- MCP/Skill/Embedding/RecruitingIntelligence/CandidateMatch 本地化。
- 私网限制、allowlist、credential encryption、fallback、semantic retrieval 语义兼容。
- 覆盖 MCP policy、skill version、embedding config、candidate match 测试。

## Required Checks

- AI Agent targeted tests
- `go test ./...` in `smart-recruit-ai-agent-service`
- scope check
- agent-check

## Manual Verification, if needed

本 TASK 需要人工确认。

## Out-of-Scope

schema/proto/security 行为变化。
