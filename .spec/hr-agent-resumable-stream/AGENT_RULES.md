# HR Agent Resumable Stream Agent Rules

## Mode

This feature was initialized with `spec-harness` init-feature mode. Do not implement business code until a specific TASK is selected and confirmed according to `TASKS.md`.

## Source Of Truth

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SPEC.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SDD.md`
- `.spec/hr-agent-resumable-stream/TASKS.md`
- `.spec/hr-agent-resumable-stream/task-scope.json`
- The selected TASK acceptance file

## Execution Rules

- Execute only one TASK at a time.
- Read the SPEC, SDD, TASKS, task scope, and selected acceptance file before editing.
- Do not modify files outside the selected TASK allowed scope.
- Do not refactor unrelated code.
- Do not delete existing tests.
- Do not modify public API behavior, proto contracts, database schema, shared types, global config, package manifests, or lockfiles unless the selected TASK explicitly allows it and the user has confirmed it when required.
- Treat browser refresh and SSE disconnect as subscription lifecycle only; they must not imply backend cancellation.
- Treat explicit cancel as a command.
- Keep legacy chat stream compatibility until a later confirmed TASK removes it.
- Keep logic-grpc as the owner of run lifecycle and durable state.
- Keep web-gin gateway transport-oriented.
- Keep the frontend reducer framework-independent and local to HR Agent runtime unless a later confirmed TASK changes that decision.

## Required Checks After Each TASK

Run or explicitly explain why unable to run:

- `git diff --name-only`
- `bash .spec/hr-agent-resumable-stream/scripts/check-task-scope.sh <TASK-ID>`
- `bash .spec/hr-agent-resumable-stream/scripts/agent-check.sh`

## TASK Report Requirements

Each TASK report under `.spec/hr-agent-resumable-stream/reports/` must include:

- TASK ID
- Modified file list
- Change summary for each file
- Whether changes exceed task scope
- SPEC comparison result
- SDD comparison result
- Acceptance comparison result
- Test commands and results
- Knowledge impact result
- Risks
- Whether the next TASK can start

## Hard Stops

Stop and ask the user before implementation when:

- The selected TASK has `requiresHumanConfirmation: true`.
- A required change touches files outside the selected TASK scope.
- A required change touches public API, proto, database schema, shared types, global config, package manifests, or lockfiles and the TASK does not explicitly allow that change.
- The implementation would change product behavior beyond the SPEC or SDD.
- A migration, rollback, or generated proto update cannot be validated.
