---
name: spec-harness
description: Use this skill to create SPEC + SDD documents from a feature prompt, generate TASK + Harness files, and then verify, execute, review, and repair one TASK at a time under .spec/<feature-name>.
---

# spec-harness

## Canonical Authority

For repositories that adopt this workflow, the canonical control plane is:

1. the nearest applicable `AGENTS.md` for durable repository rules;
2. this `spec-harness` skill for feature and single-TASK lifecycle semantics;
3. `harness-pipeline` only when the user explicitly requests serial multi-TASK orchestration;
4. `.spec/<feature-name>/` for executable feature requirements, scope, acceptance, runtime evidence, and reports.

Provider-specific prompts, skills, agents, commands, memories, historical plans, and execution logs may adapt to or inform this workflow, but they must not redefine its TASK source, review verdict, runtime state, or completion semantics.

Use this skill for feature-level SPEC + SDD + Harness workflows under:

```text
.spec/<feature-name>/
```

Each feature must have its own isolated directory.

The required SPEC and SDD file names are:

```text
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
```

After SPEC and SDD are created, this skill can generate the Harness files:

```text
.spec/<feature-name>/TASKS.md
.spec/<feature-name>/AGENT_RULES.md
.spec/<feature-name>/task-scope.json
.spec/<feature-name>/acceptance/
.spec/<feature-name>/prompts/
.spec/<feature-name>/scripts/
.spec/<feature-name>/reports/
```

This skill supports six modes:

* `init-feature`
* `draft-spec-sdd`
* `prepare-harness`
* `implement-task`
* `self-review`
* `fix-check-failures`

## Mode Summary

| Mode                 | Purpose                                                                                      |
| -------------------- | -------------------------------------------------------------------------------------------- |
| `init-feature`       | From a user-provided feature prompt, create SPEC + SDD, then generate TASKS + Harness files. |
| `draft-spec-sdd`     | Only create SPEC + SDD from a feature prompt and current repository context.                 |
| `prepare-harness`    | Generate TASKS + Harness files from existing SPEC + SDD.                                     |
| `implement-task`     | Execute exactly one specified TASK.                                                          |
| `self-review`        | Review the current TASK diff without modifying code.                                         |
| `fix-check-failures` | Fix only failures from the previous Harness, test, or self-review check.                     |

## Common Rules

* Work from the current repository files only.
* Every feature must live under `.spec/<feature-name>/`.
* `feature_name` should use lowercase kebab-case, for example:

  * `skill-memory-ranking`
  * `voice-input-hotkey`
  * `e2e-cli-coverage`
* Do not place feature SPEC, SDD, TASKS, acceptance files, prompts, scripts, or reports outside `.spec/<feature-name>/`.
* Do not modify business code during:

  * `init-feature`
  * `draft-spec-sdd`
  * `prepare-harness`
  * `self-review`
* Business code may only be modified during:

  * `implement-task`
  * `fix-check-failures`
* During implementation or repair, modify only files allowed by the current TASK scope.
* Do not execute more than one TASK at a time.
* Do not continue to the next TASK without user confirmation, unless execution is explicitly delegated to a separate pipeline orchestrator such as `harness-pipeline` and the current TASK does not require human confirmation.
* Do not silently invent missing requirements.
* Missing, unclear, or conflicting requirements must be written as:

  * `Open Questions`
  * `Assumptions Requiring Confirmation`
  * or `Hard Stop` findings
* Do not modify existing SPEC or SDD files unless:

  * the current mode explicitly allows creating them and they do not already exist, or
  * the user explicitly asks to revise or overwrite them.
* If existing SPEC or SDD files are present and the user asks to create new ones, stop and ask for confirmation before overwriting.
* If a required behavior is not present in the user prompt, SPEC, SDD, TASKS, or acceptance criteria, do not implement it.
* If repository inspection contradicts the user prompt, stop and report the conflict.

## Required Prompt Inputs

The caller should provide:

```text
Mode: <mode-name>
Feature: <feature-name>
Task: <TASK-ID, only required for TASK modes>
Feature Prompt: <required for init-feature or draft-spec-sdd>
```

Example:

```text
Use spec-harness.

Mode: init-feature
Feature: skill-memory-ranking

Feature Prompt:
Refactor Skill / Memory recall ranking from raw additive score to a production-grade hybrid ranking design...
```

## Feature Directory Contract

For a feature named:

```text
skill-memory-ranking
```

the directory structure must be:

```text
.spec/skill-memory-ranking/
  skill-memory-ranking-SPEC.md
  skill-memory-ranking-SDD.md
  TASKS.md
  AGENT_RULES.md
  task-scope.json
  acceptance/
    TASK-001.md
    TASK-002.md
  prompts/
    implement-task.md
    self-review.md
    fix-check-failures.md
  scripts/
    check-task-scope.sh
    agent-check.sh
  reports/
    TASK-001-report.md
```

## Canonical Harness Contract

### Schema version

New Harness packages must use `schemaVersion: 1` in `task-scope.json`. The canonical top-level shape is:

```json
{
  "schemaVersion": 1,
  "feature_name": "<feature-name>",
  "tasks": {}
}
```

Classify an inspected feature without rewriting it:

* `current`: `schemaVersion: 1`, `feature_name`, `tasks`, and all required Harness artifacts are present and internally consistent.
* `legacy-compatible`: `schemaVersion` is absent, but the feature uses the `feature_name + tasks` shape and can be read safely. Report the missing version; do not silently rewrite it.
* `unsupported`: the scope is flat or ambiguous, required Harness artifacts are missing, TASK IDs disagree, or the schema cannot be interpreted safely. Implementation and pipeline execution must stop.

Completed historical features may remain readable in their original format. Pending or resumed work must pass preflight before implementation.

### Runtime state ownership

`pipeline-state.json` is the runtime status source for TASK and pipeline execution. `task-scope.json.tasks[*].status` is only the generated initial state and must not override runtime evidence or be independently advanced.

Runtime state must never report ordinary completion when required scope checks, validation commands, review, evidence, or human confirmation failed or are missing. Pipeline-specific transitions and exception semantics are owned by `harness-pipeline`.

### TASK evidence

Every implemented TASK must create or update:

```text
.spec/<feature-name>/reports/<TASK-ID>-evidence.json
```

Evidence must be machine-readable and include at least:

* `schemaVersion`, `feature_name`, and `task_id`;
* TASK start and end Git identifiers;
* the actual changed-file list;
* scope status, including forbidden and out-of-scope files;
* validation commands with real exit codes and timing information;
* review type, round, and verdict;
* required human confirmation or approved exception metadata;
* skipped checks and their reasons.

Markdown reports and evidence must agree. A failed command, failed scope check, missing confirmation, or failed review must not be described as passing or complete.

### Review independence

Use an independent read-only Agent or fresh context for `self-review` when the current environment supports it. If only the implementing Agent can review, continue with the canonical `self-review` format and disclose `reviewer_type: self-review` in the report and evidence. The verdict syntax does not change.

### Legacy migration gate

Historical task files, phase plans, execution logs, and agent memories are reference material, not executable feature contracts. If a requested legacy item has no explicit valid `.spec/<feature-name>/` equivalent, stop implementation and route it through `draft-spec-sdd` with current repository inspection. Never mechanically promote stale requirements or silently infer a mapping.

---

# Mode: init-feature

## Purpose

Create the full SPEC + SDD + Harness package for one feature from a user-provided feature prompt.

This mode performs both:

```text
draft-spec-sdd
prepare-harness
```

in one controlled flow.

## Must Read

* The user-provided feature prompt
* Relevant repository files needed to understand:

  * existing architecture
  * current implementation
  * existing APIs
  * data structures
  * configuration
  * tests
  * coding conventions
  * compatibility constraints

## Allowed to Create

```text
.spec/<feature-name>/
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
.spec/<feature-name>/TASKS.md
.spec/<feature-name>/AGENT_RULES.md
.spec/<feature-name>/task-scope.json
.spec/<feature-name>/acceptance/
.spec/<feature-name>/prompts/
.spec/<feature-name>/scripts/
.spec/<feature-name>/reports/
```

## Forbidden

* Do not modify business code.
* Do not implement functionality.
* Do not modify package manifests or lockfiles.
* Do not modify shared modules.
* Do not modify shared types.
* Do not modify global configuration.
* Do not overwrite existing SPEC or SDD files without explicit user confirmation.
* Do not invent requirements that are not present in the feature prompt or current repository.

## Execution Steps

1. Validate `feature_name`.
2. Create `.spec/<feature-name>/` if it does not exist.
3. Inspect relevant repository files.
4. Extract requirements from the feature prompt.
5. Identify existing architecture and affected code areas.
6. Create:

```text
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
```

7. Validate that SPEC and SDD do not conflict.
8. Generate:

```text
TASKS.md
AGENT_RULES.md
task-scope.json
acceptance/
prompts/
scripts/
reports/
```

9. Output created files, risks, assumptions, open questions, and first TASK instructions.

## Output Must Include

* Created files and directories
* Repository areas inspected
* Requirements extracted from the feature prompt
* SPEC summary
* SDD summary
* Generated TASK list
* Any SPEC / SDD conflicts
* Any assumptions requiring confirmation
* Any Hard Stop findings
* Instructions for running the first TASK

---

# Mode: draft-spec-sdd

## Purpose

Create initial SPEC and SDD documents for one feature from a user-provided feature prompt and the current repository context.

Use this mode when the user wants to review SPEC and SDD before generating TASKS and Harness files.

## Must Read

* The user-provided feature prompt
* Relevant current repository files needed to understand:

  * existing architecture
  * current implementation
  * existing APIs
  * data structures
  * configuration
  * tests
  * compatibility constraints

## Allowed to Create

```text
.spec/<feature-name>/
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
```

## Forbidden

* Do not modify business code.
* Do not create TASKS or Harness files in this mode.
* Do not implement functionality.
* Do not silently resolve unclear requirements.
* Do not invent requirements that are not present in the user prompt or repository.
* Do not modify existing SPEC or SDD files unless the user explicitly asks for revision or overwrite.

## SPEC Requirements

The SPEC document must include:

```markdown
# <Feature Name> SPEC

## 1. Background

## 2. Goals

## 3. Non-Goals

## 4. User-Facing Behavior

## 5. Functional Requirements

## 6. Non-Functional Requirements

## 7. Compatibility Requirements

## 8. Observability and Debug Requirements

## 9. Error Handling and Fallback Requirements

## 10. Security and Safety Requirements

## 11. Acceptance Criteria

## 12. Out of Scope

## 13. Assumptions Requiring Confirmation

## 14. Open Questions
```

### SPEC Rules

* Requirements must be traceable to the feature prompt or existing repository behavior.
* If the feature prompt is unclear, record the uncertainty instead of inventing behavior.
* Acceptance criteria must be testable.
* Non-goals must explicitly prevent scope creep.
* Compatibility requirements must identify existing APIs, CLI behavior, UI behavior, or tests that should not break.

## SDD Requirements

The SDD document must include:

```markdown
# <Feature Name> SDD

## 1. Existing Architecture Summary

## 2. Problem Analysis

## 3. Proposed Design

## 4. Data Structure Changes

## 5. API and Interface Changes

## 6. Algorithm or Workflow Changes

## 7. Configuration Design

## 8. Compatibility Strategy

## 9. Error Handling and Fallback Design

## 10. Observability and Debug Output Design

## 11. Testing Strategy

## 12. Migration Risks

## 13. Implementation Boundaries

## 14. Alternatives Considered

## 15. Assumptions Requiring Confirmation

## 16. Open Questions
```

### SDD Rules

* The SDD must be grounded in current repository structure.
* Do not propose architecture that conflicts with existing project conventions.
* Do not introduce new dependencies unless explicitly required by the user.
* If public API behavior may change, mark it as a Hard Stop unless explicitly allowed.
* Implementation boundaries must be specific enough to guide TASK generation.
* Testing strategy must map to acceptance criteria.

## Output Must Include

* Created SPEC file path
* Created SDD file path
* Requirements extracted from the prompt
* Existing code areas inspected
* Open questions
* Assumptions requiring confirmation
* Compatibility risks
* Whether the feature is ready for `prepare-harness`

---

# Mode: prepare-harness

## Purpose

Generate TASKS and Harness files for one feature from existing SPEC and SDD.

Use this after SPEC and SDD already exist.

## Must Read

```text
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
```

## Allowed to Create

```text
.spec/<feature-name>/TASKS.md
.spec/<feature-name>/AGENT_RULES.md
.spec/<feature-name>/task-scope.json
.spec/<feature-name>/acceptance/
.spec/<feature-name>/prompts/
.spec/<feature-name>/scripts/
.spec/<feature-name>/reports/
```

## Forbidden

* Do not modify business code.
* Do not modify existing SPEC or SDD.
* Do not implement functionality.
* Do not invent TASK requirements not supported by SPEC or SDD.
* Do not create tasks that require behavior missing from SPEC or SDD.
* Do not silently resolve SPEC / SDD conflicts.

## TASKS.md Requirements

`TASKS.md` must include:

```markdown
# TASKS - <feature-name>

## Task Overview

| TASK | Title | Status | Scope | Acceptance |
|------|-------|--------|-------|------------|

## TASK-001 - <title>

### Goal

### Scope

### Allowed Files

### Forbidden Files

### Dependencies

### Acceptance Criteria

### Required Tests

### Risks

### Notes
```

Each TASK must be:

* independently reviewable
* small enough for one Agent execution cycle
* traceable to SPEC and SDD
* limited by explicit file scope
* ordered by dependency
* not mixed with unrelated refactors

## AGENT_RULES.md Requirements

`AGENT_RULES.md` must define:

* feature-specific implementation rules
* scope boundaries
* forbidden changes
* testing requirements
* compatibility rules
* logging / debug rules
* report requirements
* Hard Stop conditions

## task-scope.json Requirements

`task-scope.json` must define each TASK scope.

Suggested schema:

```json
{
  "schemaVersion": 1,
  "feature_name": "<feature-name>",
  "tasks": {
    "TASK-001": {
      "title": "Short task title",
      "status": "pending",
      "allowedFiles": [],
      "forbiddenFiles": [],
      "requiresHumanConfirmation": false,
      "allowedActions": [
        "modify",
        "create",
        "test"
      ],
      "acceptance": ".spec/<feature-name>/acceptance/TASK-001.md",
      "report": ".spec/<feature-name>/reports/TASK-001-report.md",
      "notes": []
    }
  }
}
```

Rules:

* New Harness packages must set `schemaVersion: 1`.
* `allowedFiles` must be explicit when possible.
* If exact files are unknown, use the narrowest safe directory pattern.
* Shared modules, shared types, package manifests, lockfiles, global configuration, and public APIs should default to forbidden unless explicitly required and confirmed.
* Set `requiresHumanConfirmation: true` for high-risk TASKs.
* Treat `tasks[*].status` as generated initial metadata; runtime status belongs to `pipeline-state.json`.

## acceptance/ Requirements

Create one acceptance file per TASK:

```text
.spec/<feature-name>/acceptance/<TASK-ID>.md
```

Each acceptance file must include:

```markdown
# Acceptance - <TASK-ID>

## TASK Summary

## SPEC References

## SDD References

## Acceptance Criteria

## Required Checks

## Manual Verification, if needed

## Out-of-Scope
```

## prompts/ Requirements

Create prompt templates:

```text
.spec/<feature-name>/prompts/implement-task.md
.spec/<feature-name>/prompts/self-review.md
.spec/<feature-name>/prompts/fix-check-failures.md
```

Each prompt must instruct the Agent to:

* use `spec-harness`
* specify current mode
* specify current feature
* specify current TASK
* read required files
* obey task scope
* stop on Hard Stop conditions

## scripts/ Requirements

Create:

```text
.spec/<feature-name>/scripts/check-task-scope.sh
.spec/<feature-name>/scripts/agent-check.sh
```

### check-task-scope.sh

This script should verify that changed files are allowed by `task-scope.json` for the specified TASK.

Required behavior:

```bash
bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>
```

It should:

* read changed files from `git diff --name-only`
* compare them against the TASK `allowedFiles`
* fail if changed files exceed scope
* print all out-of-scope files
* fail if TASK-ID does not exist

### agent-check.sh

This script should run project-appropriate validation checks.

It may include existing commands such as:

* unit tests
* type checks
* lint checks
* formatting checks
* targeted tests

Rules:

* Prefer existing project scripts.
* Do not invent unavailable scripts.
* If no safe command is known, create a script that clearly reports which checks must be filled in.
* Do not modify `package.json` just to add scripts.

## reports/ Requirements

Create the directory:

```text
.spec/<feature-name>/reports/
```

Add a `.gitkeep` file if needed.

## Output Must Include

* Created files and directories
* TASK list
* Acceptance files generated
* Scope rules generated
* Scripts generated
* Any detected SPEC / SDD conflicts
* Any assumptions requiring user confirmation
* Instructions for running the first TASK

---

# Mode: implement-task

## Purpose

Execute exactly one specified TASK.

## Must Read

```text
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
.spec/<feature-name>/TASKS.md
.spec/<feature-name>/AGENT_RULES.md
.spec/<feature-name>/task-scope.json
.spec/<feature-name>/acceptance/<TASK-ID>.md
.spec/<feature-name>/prompts/implement-task.md
```

## Preflight

Before modifying files:

1. Classify the feature as `current`, `legacy-compatible`, or `unsupported` using the Canonical Harness Contract.
2. Verify the requested TASK exists consistently in `TASKS.md`, `task-scope.json`, and its acceptance file.
3. Verify required prompts, scripts, and report paths exist.
4. Establish a reliable TASK-level Git baseline that can distinguish this TASK from earlier TASK changes.
5. Check `requiresHumanConfirmation` and record explicit confirmation before editing when required.
6. Stop on unsupported schema, missing artifacts, unreliable baseline, unrelated dirty changes, or conflicting requirements. Do not downgrade to manual visual comparison.

## Restrictions

* Execute only the specified TASK.
* Do not execute later TASKs.
* Do not modify files outside the TASK scope.
* Do not modify SPEC or SDD.
* Do not modify unrelated tests.
* Do not refactor unrelated code.
* Do not add new functionality beyond the current TASK.
* If the TASK requires a wider scope, stop and request confirmation.
* If required behavior is not present in SPEC, SDD, TASKS, or acceptance, stop and report the gap.
* Run Harness checks after implementation.
* Generate or update a TASK completion report.
* Stop after the report.

## Required Checks After Implementation

Run:

```bash
git diff --name-only
bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>
bash .spec/<feature-name>/scripts/agent-check.sh
```

If additional TASK-specific checks are listed in the acceptance file, run them too.

## Completion Report

Create or update:

```text
.spec/<feature-name>/reports/<TASK-ID>-report.md
```

Create or update the matching machine-readable evidence file:

```text
.spec/<feature-name>/reports/<TASK-ID>-evidence.json
```

The report must include:

```markdown
# TASK Report - <TASK-ID>

## 1. TASK ID

## 2. Modified File List

## 3. Change Summary by File

## 4. Scope Check Result

## 5. SPEC Comparison Result

## 6. SDD Comparison Result

## 7. Acceptance Comparison Result

## 8. Test Commands and Results

## 9. Risks

## 10. Follow-up Items

## 11. Whether the Next TASK Can Start
```

## Output Must Include

* TASK ID
* Modified file list
* Summary of changes
* Scope check result
* Test results
* Report path
* Evidence path
* Reviewer type or planned review type
* Whether the next TASK can start

---

# Mode: self-review

## Purpose

Review the current TASK diff without modifying code.

## Must Read

```text
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
.spec/<feature-name>/TASKS.md
.spec/<feature-name>/AGENT_RULES.md
.spec/<feature-name>/task-scope.json
.spec/<feature-name>/acceptance/<TASK-ID>.md
.spec/<feature-name>/reports/<TASK-ID>-report.md
```

## Review Against

* SPEC
* SDD
* TASKS
* `task-scope.json`
* acceptance file
* `AGENT_RULES.md`
* current `git diff`
* test results in the TASK report

## Default Behavior

* Do not modify code.
* Do not fix issues in this mode.
* Prefer an independent read-only Agent or fresh context when available.
* If review is performed by the implementing Agent, disclose `reviewer_type: self-review`.
* Report findings ordered by severity.
* Identify:

  * scope violations
  * missing tests
  * incomplete acceptance criteria
  * behavior not covered by SPEC or SDD
  * compatibility risks
  * unsafe assumptions
  * unnecessary refactors
  * hidden public API changes
  * insufficient error handling
  * insufficient observability or debug output

## Verdict

The review must end with exactly one verdict:

```text
verdict: 通过
```

or

```text
verdict: 不通过
```

If not passing, provide a concrete issue list.

## Finding Format

Use this format:

```markdown
## Findings

### Critical

### High

### Medium

### Low

## Required Fixes

| ID | Severity | File | Problem | Required Fix |
|----|----------|------|---------|--------------|

## Verdict

verdict: 通过
```

or:

```markdown
## Verdict

verdict: 不通过
```

---

# Mode: fix-check-failures

## Purpose

Fix only failures from the previous Harness, test, or self-review check.

## Must Read

```text
.spec/<feature-name>/<feature-name>-SPEC.md
.spec/<feature-name>/<feature-name>-SDD.md
.spec/<feature-name>/TASKS.md
.spec/<feature-name>/AGENT_RULES.md
.spec/<feature-name>/task-scope.json
.spec/<feature-name>/acceptance/<TASK-ID>.md
.spec/<feature-name>/reports/<TASK-ID>-report.md
```

Also read the previous failed output from:

* Harness checks
* test commands
* self-review findings
* user-provided failure logs

## Restrictions

* Fix only the failed items.
* Do not expand TASK scope.
* Do not add new functionality.
* Do not refactor unrelated code.
* Do not modify SPEC or SDD.
* Do not modify unrelated tests.
* If the fix requires a wider scope, stop and request confirmation.
* If two consecutive repair attempts fail on the same issue, stop and explain the root cause.

## Required After Fixes

Re-run:

```bash
git diff --name-only
bash .spec/<feature-name>/scripts/check-task-scope.sh <TASK-ID>
bash .spec/<feature-name>/scripts/agent-check.sh
```

Also re-run the originally failed check.

## Report Update

Update or create:

```text
.spec/<feature-name>/reports/<TASK-ID>-report.md
```

Add:

```markdown
## Repair Summary

### Failed Check

### Root Cause

### Files Changed

### Fix Summary

### Re-run Commands

### Re-run Results

### Remaining Risks
```

## Output Must Include

* Failed items fixed
* Files changed
* Commands re-run
* Results
* Remaining risks
* Whether another self-review is needed

---

# Hard Stop Conditions

Stop and request confirmation if any of the following occur:

* Need to modify `package.json`.
* Need to modify a lockfile.
* Need to add, remove, or upgrade dependencies.
* Need to modify a shared module.
* Need to modify shared types.
* Need to modify global configuration.
* Need to change public API behavior.
* Need to change database schema or migration files.
* Need to change authentication, authorization, permission, or security behavior.
* Need to change CI/CD configuration.
* Need to modify files outside the current TASK scope.
* TASK scope is unclear.
* Feature prompt is too vague to produce SPEC or SDD.
* Existing code structure cannot be located.
* SPEC, SDD, TASKS, and acceptance criteria conflict.
* Required behavior is not present in the feature prompt, SPEC, or SDD.
* Existing SPEC or SDD files would be overwritten without explicit confirmation.
* Harness scripts cannot safely determine task scope.
* The repository has unrelated dirty changes and the current mode may modify files.
* The feature is classified as `unsupported` for implementation or pipeline execution.
* A reliable TASK-level Git baseline cannot be established.
* Required TASK evidence cannot be produced without falsifying or dropping failed checks.

---

# Recommended Usage Flow

## Full initialization from feature prompt

Use when starting a new feature:

```text
Use spec-harness.

Mode: init-feature
Feature: <feature-name>

Feature Prompt:
<describe the feature requirements here>
```

This creates:

```text
SPEC + SDD + TASKS + Harness
```

## Staged initialization

Use when the user wants to review SPEC and SDD first:

```text
Use spec-harness.

Mode: draft-spec-sdd
Feature: <feature-name>

Feature Prompt:
<describe the feature requirements here>
```

Then after user confirmation:

```text
Use spec-harness.

Mode: prepare-harness
Feature: <feature-name>
```

## Execute one TASK

```text
Use spec-harness.

Mode: implement-task
Feature: <feature-name>
Task: TASK-001
```

## Review one TASK

```text
Use spec-harness.

Mode: self-review
Feature: <feature-name>
Task: TASK-001
```

## Fix failed checks

```text
Use spec-harness.

Mode: fix-check-failures
Feature: <feature-name>
Task: TASK-001
```

---

# Final Output Rules

Every mode must end with a concise summary.

The summary must include:

* Mode executed
* Feature name
* TASK ID, if applicable
* Files created or modified
* Checks run, if applicable
* Risks
* Hard Stop findings, if any
* Recommended next command

Do not continue into another mode unless the user explicitly requested that mode or the current mode is `init-feature`.
