# Fix Check Failures Prompt

Use `spec-harness`.

Mode: `fix-check-failures`

Feature: `hr-agent-resumable-stream`

Task: `<TASK-ID>`

Read:

- `AGENTS.md`
- `.agents/skills/spec-harness/SKILL.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SPEC.md`
- `.spec/hr-agent-resumable-stream/hr-agent-resumable-stream-SDD.md`
- `.spec/hr-agent-resumable-stream/TASKS.md`
- `.spec/hr-agent-resumable-stream/AGENT_RULES.md`
- `.spec/hr-agent-resumable-stream/task-scope.json`
- `.spec/hr-agent-resumable-stream/acceptance/<TASK-ID>.md`
- `.spec/hr-agent-resumable-stream/reports/<TASK-ID>-report.md`
- The failing Harness, test, or self-review output

Fix only the reported failures. Do not expand TASK scope, add unrelated behavior, modify SPEC/SDD, or move to another TASK. Re-run required checks and update the report and evidence JSON.
