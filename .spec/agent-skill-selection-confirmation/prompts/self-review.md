# Self Review Prompt

Use spec-harness.

Mode: self-review
Feature: agent-skill-selection-confirmation
Task: <TASK-ID>

Review against:

- SPEC
- SDD
- TASKS
- AGENT_RULES
- task-scope.json
- acceptance file
- current git diff
- TASK report

Do not modify files. Report findings by severity and end with exactly one verdict:

`verdict: 通过`

or

`verdict: 不通过`
