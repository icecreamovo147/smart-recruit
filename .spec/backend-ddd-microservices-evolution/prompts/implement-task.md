# Implement TASK Prompt - backend-ddd-microservices-evolution

Use spec-harness.

Mode: implement-task
Feature: backend-ddd-microservices-evolution
Task: <TASK-ID>

## Required Reads

- `.spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SPEC.md`
- `.spec/backend-ddd-microservices-evolution/backend-ddd-microservices-evolution-SDD.md`
- `.spec/backend-ddd-microservices-evolution/TASKS.md`
- `.spec/backend-ddd-microservices-evolution/AGENT_RULES.md`
- `.spec/backend-ddd-microservices-evolution/task-scope.json`
- `.spec/backend-ddd-microservices-evolution/acceptance/<TASK-ID>.md`
- `.spec/backend-ddd-microservices-evolution/prompts/implement-task.md`
- `AGENTS.md`
- `.knowledge/README.md`, `.knowledge/manifest.yaml`, and only relevant active knowledge files

## Instructions

1. Classify the feature as current before implementation.
2. Establish a reliable TASK base tree and export `TASK_BASE_TREE` before scope checks.
3. If `requiresHumanConfirmation` is true, stop until explicit user confirmation is recorded.
4. Modify only files allowed by `task-scope.json` for the current TASK.
5. Stop on Hard Stop conditions from `AGENT_RULES.md` or `spec-harness`.
6. Run required checks from the acceptance file.
7. Create both report and evidence files for the TASK.
8. Do not continue to the next TASK.
