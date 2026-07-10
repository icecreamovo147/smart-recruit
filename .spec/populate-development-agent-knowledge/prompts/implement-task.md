Use spec-harness.

Mode: implement-task
Feature: populate-development-agent-knowledge
Task: <TASK-ID>

Read the feature SPEC, SDD, TASKS, AGENT_RULES, task-scope, acceptance file, `.knowledge/README.md`, `.knowledge/INDEX.md`, and `.knowledge/manifest.yaml`.

Execute exactly this TASK. Modify only files allowed by `task-scope.json`. Do not modify business code, package manifests, lockfiles, proto/generated files, migrations, CI config, or historical feature contracts. Verify knowledge claims against current source references. Put uncertain or policy-level conclusions in Inbox or report follow-up.

After implementation, run the required checks and create both the Markdown report and machine-readable evidence with `knowledgeImpact`.
