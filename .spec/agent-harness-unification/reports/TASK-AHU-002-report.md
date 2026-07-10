# TASK Report - TASK-AHU-002

## 1. TASK ID

TASK-AHU-002 - 实现共享结构、Scope 与 Evidence 校验器

## 2. Modified File List

- `.agents/skills/spec-harness/scripts/validate-feature.mjs`
- `.agents/skills/spec-harness/scripts/check-task-scope.mjs`
- `.agents/skills/spec-harness/scripts/validate-evidence.mjs`
- `.agents/skills/spec-harness/scripts/audit-specs.mjs`
- `.agents/skills/spec-harness/scripts/validator.test.mjs`
- `.spec/agent-harness-unification/reports/TASK-AHU-002-report.md`
- `.spec/agent-harness-unification/reports/TASK-AHU-002-evidence.json`
- `.spec/agent-harness-unification/pipeline-state.json`

The scope checker also sees prior untracked `.spec/agent-harness-unification/**` Harness files from the feature baseline; they are within this TASK's allowed feature evidence scope.

## 3. Change Summary by File

- `.agents/skills/spec-harness/scripts/validate-feature.mjs`: added read-only feature classification for `current`, `legacy-compatible`, and `unsupported`; validates required Harness artifacts, TASK alignment, schema shape, acceptance/report references, pattern syntax, and optional pipeline state presence.
- `.agents/skills/spec-harness/scripts/check-task-scope.mjs`: added deterministic TASK scope checking from a required base tree; merges tracked, staged, unstaged, and untracked changes; reports allowed, forbidden, and out-of-scope matches.
- `.agents/skills/spec-harness/scripts/validate-evidence.mjs`: added machine-readable evidence validation for required fields, command exit code/status consistency, scope result consistency, review verdict, confirmation, and exceptions.
- `.agents/skills/spec-harness/scripts/audit-specs.mjs`: added read-only `.spec` scanner that classifies feature directories without rewriting them.
- `.agents/skills/spec-harness/scripts/validator.test.mjs`: added controlled temporary Git fixtures for current, legacy-compatible, unsupported, unknown TASK, missing baseline, allowed, forbidden, out-of-scope, staged, unstaged, untracked, and evidence contradiction behavior.
- `.spec/agent-harness-unification/reports/TASK-AHU-002-report.md`: records implementation, checks, and review status for this TASK.
- `.spec/agent-harness-unification/reports/TASK-AHU-002-evidence.json`: records machine-readable evidence for checks and review.
- `.spec/agent-harness-unification/pipeline-state.json`: advances TASK-AHU-002 from implementation to review evidence.

## 4. Scope Check Result

PASS.

Command:

```bash
bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-002
```

Result:

- Exit code: 0
- Classification: `current`
- All detected TASK-local files matched either `.agents/skills/spec-harness/scripts/**` or `.spec/agent-harness-unification/**`.
- No forbidden or out-of-scope files were reported.

## 5. SPEC Comparison Result

PASS.

- FR-007 is covered by `validate-feature.mjs`.
- FR-008 is covered by `check-task-scope.mjs`.
- FR-010 is covered by `validate-evidence.mjs`.
- FR-013 is supported by shared audit output from `audit-specs.mjs`.

## 6. SDD Comparison Result

PASS.

- SDD 3.6 is reflected in schema v1 and legacy-compatible classification.
- SDD 3.8 is reflected in required `base_tree` scope checks.
- SDD 3.9 is reflected in evidence validation.
- SDD 3.10 is reflected in fixed shared script names.
- SDD 11.2-11.4 are covered by temporary Git fixtures and `.spec` audit.

## 7. Acceptance Comparison Result

PASS.

- current, legacy-compatible, and unsupported classifications are deterministic.
- unsupported and malformed Harness data return non-zero from feature validation.
- unknown TASK, invalid patterns, and missing baseline fail closed.
- allowed, forbidden, out-of-scope, staged, unstaged, and untracked changes are covered in `validator.test.mjs`.
- audit mode is read-only and reports unsupported legacy features without modifying them.
- evidence validator detects command status contradictions.
- Required script files are present.
- No dependencies, shared Skill docs, or business code were modified by this TASK.

## 8. Test Commands and Results

| Command | Exit Code | Result |
| --- | ---: | --- |
| `git diff --name-only` | 0 | Listed existing feature diff and new TASK-AHU-002 scripts. |
| `bash .spec/agent-harness-unification/scripts/check-task-scope.sh TASK-AHU-002` | 0 | PASS |
| `bash .spec/agent-harness-unification/scripts/agent-check.sh` | 0 | PASS |
| `node .agents/skills/spec-harness/scripts/validator.test.mjs` | 0 | PASS |
| `node .agents/skills/spec-harness/scripts/audit-specs.mjs --root .spec` | 0 | PASS; reported current, legacy-compatible, and unsupported features without rewriting them. |

## 9. Risks

- Glob support is intentionally small and fails closed on unsupported character class syntax.
- Existing unsupported `.spec` directories remain unsupported; this TASK only detects and reports them.
- TASK-local baseline quality depends on the pipeline recording a reliable `base_tree`.

## 10. Follow-up Items

- TASK-AHU-003 should wire pipeline completion state to evidence validation invariants.
- Later final audit should verify adapters and legacy inventory use the shared scripts consistently.

## 11. Whether the Next TASK Can Start

Yes. Self-review completed with `verdict: 通过`.

## Self-Review

## Findings

### Critical

None.

### High

None.

### Medium

None.

### Low

None.

## Required Fixes

| ID | Severity | File | Problem | Required Fix |
| --- | --- | --- | --- | --- |
| - | - | - | No required fixes. | - |

## Verdict

verdict: 通过

## Repair Summary

### Failed Check

TASK-AHU-003 scope validation initially reported TASK-AHU-002 `spec-harness/scripts` files as forbidden current-task changes.

### Root Cause

`check-task-scope.mjs` used `git diff <base-tree>` plus untracked-file inspection. When a synthetic TASK baseline tree included files that were still untracked in the real Git index, `git diff <base-tree>` reported those paths even when their working-tree blob matched the baseline blob.

### Files Changed

- `.agents/skills/spec-harness/scripts/check-task-scope.mjs`
- `.agents/skills/spec-harness/scripts/validator.test.mjs`

### Fix Summary

- If an untracked file exists in the TASK baseline tree and its working-tree blob matches, the scope checker now removes that path from the changed set.
- Added a fixture where a synthetic base tree contains an otherwise untracked file with identical content.

### Re-run Commands

```bash
node .agents/skills/spec-harness/scripts/validator.test.mjs
node .agents/skills/spec-harness/scripts/audit-specs.mjs --root .spec
```

### Re-run Results

- `validator.test`: PASS
- `.spec` audit: PASS

### Remaining Risks

None known for this repair.
