# Self Review Prompt

Use spec-harness.

Mode: self-review
Feature: recruiting-intelligence-runtime-parity
Task: <TASK-ID>

Act as a fresh read-only Reviewer. Read the repository rules, complete skill, feature contracts, task scope/acceptance, pipeline baseline, implementation diff, tests, report, evidence, and routed knowledge. Verify behavior against current and `dev` references, independently rerun or inspect required checks, check privacy and out-of-scope changes, and report actionable findings with file/line/severity.

End with exactly one of:

`verdict: 通过`

`verdict: 不通过`

Do not edit any file and do not accept missing, stale, fabricated, or contradictory evidence.
