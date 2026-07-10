Use spec-harness.

Mode: implement-task
Feature: development-agent-knowledge-base
Task: <TASK-ID>

Read completely:

- `.spec/development-agent-knowledge-base/development-agent-knowledge-base-SPEC.md`
- `.spec/development-agent-knowledge-base/development-agent-knowledge-base-SDD.md`
- `.spec/development-agent-knowledge-base/TASKS.md`
- `.spec/development-agent-knowledge-base/AGENT_RULES.md`
- `.spec/development-agent-knowledge-base/task-scope.json`
- `.spec/development-agent-knowledge-base/acceptance/<TASK-ID>.md`
- this prompt

Classify the feature, establish a reliable TASK base tree, check `requiresHumanConfirmation`, and stop on every Hard Stop. Implement exactly one TASK and modify only allowed files. Do not modify SPEC or SDD. After implementation run all required checks and create/update the TASK report and evidence. Do not continue to the next TASK.
