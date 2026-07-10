# Implement TASK Prompt

Use `spec-harness`.

Mode: `implement-task`

Feature: `hr-agent-resumable-stream`

Task: `<TASK-ID>`

Before editing, read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SPEC.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SDD.md`
- `.spec/hr-agent-resumable-stream/TASKS.md`
- `.spec/hr-agent-resumable-stream/AGENT_RULES.md`
- `.spec/hr-agent-resumable-stream/task-scope.json`
- `.spec/hr-agent-resumable-stream/acceptance/<TASK-ID>.md`
- Relevant active `.knowledge/` documents declared by the TASK scope

Rules:

- Execute only `<TASK-ID>`.
- Establish a reliable TASK baseline before editing.
- Stop if the feature validates as unsupported.
- Stop if `<TASK-ID>` requires human confirmation and confirmation is not explicit.
- Modify only files allowed by the selected TASK scope.
- Do not modify SPEC, SDD, TASKS, or acceptance files during implementation.
- Do not expand public API, schema, proto, shared types, global config, manifests, or lockfiles unless explicitly allowed and confirmed.
- After implementation, run required checks and create both Markdown report and evidence JSON.
