# AGENT RULES - agent-skill-selection-confirmation

## Feature-Specific Rules

- Implement only the current TASK.
- Do not change ranking weights, embedding generation, or Agent Skill admin behavior unless a TASK explicitly allows it.
- Preserve existing manual `agent_skill_ids` behavior.
- Keep ADK and legacy runtime behavior consistent.
- Do not expose Agent Skill body markdown or hidden prompt content to the frontend confirmation UI.
- Do not add dependencies.

## Scope Boundaries

- Backend work must stay in AI service orchestration, Agent Skill selection policy, stream payload mapping, and trace recording.
- Frontend work must stay in HR AI chat, chat components, AI API/types, and trace display.
- Proto/public API changes require explicit user confirmation before implementation.

## Forbidden Changes

- Package manifests and lockfiles.
- Database schema or migrations.
- Authentication, authorization, permissions, or rate limit policy.
- Candidate-side chat.
- Unrelated Agent Skill management pages.
- Opportunistic refactors.

## Testing Requirements

- Run the TASK-specific required checks.
- Always run `git diff --name-only`.
- Always run `bash .spec/agent-skill-selection-confirmation/scripts/check-task-scope.sh <TASK-ID>`.
- Always run `bash .spec/agent-skill-selection-confirmation/scripts/agent-check.sh`.

## Compatibility Rules

- Existing chat stream behavior must continue for requests with zero or one Skill candidate.
- Existing manually selected Skill IDs must execute without confirmation.
- Existing sessions/messages must remain readable.
- Proto copies must stay synchronized if proto is changed.

## Logging and Debug Rules

- Log confirmation-required decisions with metadata only.
- Do not log raw resume text, full user message content beyond existing logging norms, or Skill body content.
- Trace should distinguish recommended candidates from confirmed execution where applicable.

## Report Requirements

Each TASK report must include:

- TASK ID
- Modified file list
- Change summary by file
- Scope check result
- SPEC/SDD/acceptance comparison
- Test commands and results
- Risks
- Whether the next TASK can start

## Hard Stop Conditions

Stop and ask for confirmation if:

- A TASK needs files outside its allowed scope.
- A proto/public API change is required but not explicitly confirmed.
- A new request field is required to avoid repeated confirmation and the TASK did not allow proto/API changes.
- A database migration is needed.
- A package manifest or lockfile change is needed.
- Existing unrelated dirty changes prevent reliable scope validation.
