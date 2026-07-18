# Implement TASK Prompt

Use spec-harness.

Mode: implement-task
Feature: recruiting-intelligence-runtime-parity
Task: <TASK-ID>

Read `AGENTS.md`, the complete spec-harness skill, this feature's SPEC, SDD, TASKS, AGENT_RULES, task-scope, current acceptance contract, `.knowledge/README.md`, manifest, index, and routed active knowledge. Confirm the root-owned `base_sha`/`base_tree` and any required user grant in pipeline state. Inspect current and `dev` reference code, implement exactly this TASK within allowed files, run every acceptance and Harness check, and stop on any hard-stop condition.

Create `reports/<TASK-ID>-report.md` and `reports/<TASK-ID>-evidence.json` with truthful command results, scope, contract comparison, knowledge impact, privacy result, risks, and next-TASK readiness. Do not claim skipped or failing checks passed.
