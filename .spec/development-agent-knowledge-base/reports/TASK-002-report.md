# TASK Report - TASK-002

## 1. TASK ID

TASK-002 — 实现知识校验与影响检测工具

## 2. Modified File List

- `.knowledge/scripts/validate-knowledge.mjs`
- `.knowledge/scripts/check-references.mjs`
- `.knowledge/scripts/detect-impact.mjs`
- `.knowledge/scripts/generate-catalog.mjs`
- `.knowledge/scripts/knowledge-validator.test.mjs`
- `.spec/development-agent-knowledge-base/pipeline-state.json`
- `.spec/development-agent-knowledge-base/reports/TASK-002-report.md`
- `.spec/development-agent-knowledge-base/reports/TASK-002-evidence.json`

## 3. Change Summary by File

- `validate-knowledge.mjs`: parses the documented YAML subset, validates formal knowledge metadata, IDs, paths, dates, sources, lifecycle, ADR links, secrets, case conflicts, and manifest routes.
- `check-references.mjs`: checks INDEX and manifest document references, with explicit `--strict-routes` for final completeness.
- `detect-impact.mjs`: requires a reliable Git tree, includes tracked/untracked files, applies deterministic routes, and reports coverage gaps.
- `generate-catalog.mjs`: deterministically generates ignored Markdown/JSON catalogs.
- `knowledge-validator.test.mjs`: provides 17 dependency-free tests covering parsing ambiguity, required manifest sections, route/trigger item contracts, schema types, paths, globbing, references, lifecycle, ADR reciprocity, case conflicts, catalog CLI, global triggers, coverage gaps, invalid-manifest rejection, and Git impact behavior.

## 4. Scope Check Result

Passed against base tree `2702392758b6b948c716624780fda63a7e3f9942`; all changed files are allowed.

## 5. SPEC Comparison Result

符合 FR-009 至 FR-011：无新增依赖，支持文本/JSON 输出、可靠 baseline、tracked/untracked、跨平台路径、稳定排序和 coverage gap。

## 6. SDD Comparison Result

CLI、退出码、路径/glob 规则、错误处理和测试策略与 SDD Sections 5、6、9、11 一致。

## 7. Acceptance Comparison Result

全部 17 个自动化测试和 repository validation/reference checks 通过。后续 TASK 尚未创建的 routed documents 以明确 warning 呈现；最终审计使用 `--strict-routes`，届时 INDEX 和 route 缺失都将失败。

## 8. Test Commands and Results

| Command | Result |
|---|---|
| `node --check .knowledge/scripts/*.mjs` | PASS |
| `node .knowledge/scripts/knowledge-validator.test.mjs` | PASS (17) |
| `detect-impact` TASK-002 false-new-module regression check | PASS |
| `node .knowledge/scripts/validate-knowledge.mjs --root .` | PASS with expected bootstrap warnings |
| `node .knowledge/scripts/check-references.mjs --root .` | PASS with expected bootstrap warnings |
| `bash .../check-task-scope.sh TASK-002` | PASS |
| `bash .../agent-check.sh` | PASS |
| `git diff --check` | PASS |

## 9. Repair Summary

The first test run exposed that a case-insensitive macOS filesystem cannot create both `Case.md` and `case.md` in one fixture. The case-conflict algorithm was extracted into a pure path function and tested directly with cross-platform path inputs. The complete suite then passed.

Independent review round 1 found incomplete global-trigger mapping, an invalid `review_required` result, incomplete schema/ADR checks, missing CLI/strict/date/glob/device-independent behavior, and overstated test coverage. The repair:

- maps deterministically detectable changed-file categories only to triggers declared in manifest;
- emits only allowed knowledge-impact results;
- validates metadata types, uniqueness, manifest keys/policies, strict dates, restricted globs, and reciprocal ADR links;
- makes strict mode fail missing/escaping INDEX targets;
- adds catalog JSON output and usage exit code 2;
- removes absolute repository roots from JSON output;
- expands the suite from 10 to 15 tests and corrects this report.

Independent review round 2 found a no-output `git cat-file -e` bug, trigger-only `none` results, remaining manifest fail-open cases, missing pre-impact semantic validation, and stale evidence metadata. The second repair:

- distinguishes successful no-output Git existence checks from failures;
- emits `candidate_required` for trigger-only impact and deterministically handles every declared global trigger;
- rejects missing/unknown/duplicate manifest sections and route keys;
- runs full knowledge semantic validation before impact detection;
- adds regression tests for existing/new top-level directories, trigger-only changes, required manifest sections, and invalid manifest rejection;
- reruns every check and records the final tree/timestamps below.

Independent review round 3 found missing route/global-trigger element type, format, and uniqueness checks plus incorrect detect-impact CLI classification of semantic validation failures. The final scoped repair validates those list contracts, returns exit 1 for invalid knowledge contracts and exit 2 only for CLI usage, and adds direct CLI regression coverage. A subsequent agent-check run exposed inherited `TASK_BASE_TREE` in the usage test; the test now clears that environment variable before asserting exit 2. The final complete rerun passed.

## 10. Knowledge Impact

- Result: `none`
- Reason: This TASK implements knowledge tooling and does not change any routed business surface or existing formal knowledge document.
- Coverage gap: false.

## 11. Risks

- The YAML parser intentionally supports only the documented subset; unsupported syntax fails closed.
- Non-strict reference mode is required during staged content creation; TASK-008 must run strict route checking.

## 12. Follow-up Items

- TASK-003/004 must create all routed document IDs so strict reference validation passes.

## 13. Whether the Next TASK Can Start

Yes. Final self-review verdict: 通过.
