# Acceptance - TASK-006

## TASK Summary

Connect Agent, Prompt, Skill, Agent Skill, and capability governance data to runtime execution.

## SPEC References

- FR-006 Agent, Prompt, Skill, and Capability runtime integration
- Compatibility Requirements 1, 2, 7
- Acceptance Criteria 10

## SDD References

- Proposed Design 3.6
- Configuration Design
- Algorithm or Workflow Changes 6.1, 6.2
- Testing Strategy

## Acceptance Criteria

- Runtime loads default enabled AgentConfig for `hr_recruiting_agent`.
- Prompt template and instruction changes affect subsequent runtime requests.
- Capability bindings and selected `skill_capability_keys` restrict available tools and skills.
- Manual Agent Skill IDs and rule/semantic selected skills are handled consistently.
- Skill confirmation metadata is persisted where existing schema supports it.

## Required Checks

- Targeted AI Agent service tests for runtime config, prompt rendering/application, capability filtering, and Agent Skill selection/confirmation.
- Gateway or frontend tests if payload mapping changes.
- `GOWORK=off go test ./...` from `smart-recruit-ai-agent-service`.
- `pnpm --filter hr-frontend typecheck` if HR frontend is touched.
- `git diff --name-only`.
- `bash .spec/ai-agent-runtime-recovery/scripts/check-task-scope.sh TASK-006`.
- `bash .spec/ai-agent-runtime-recovery/scripts/agent-check.sh`.

## Manual Verification, if needed

Admin UI live verification may be skipped if the local stack is unavailable; document test substitute.

## Out-of-Scope

- MCP live execution.
- Embedding provider implementation.
- Auth/RBAC changes.
