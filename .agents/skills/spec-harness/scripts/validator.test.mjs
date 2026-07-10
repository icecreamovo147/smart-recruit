#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { validateFeature } from "./validate-feature.mjs";
import { checkTaskScope } from "./check-task-scope.mjs";
import { validateEvidence } from "./validate-evidence.mjs";

function write(file, content = "\n") {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
}

function git(root, args, env = {}) {
  return execFileSync("git", ["-C", root, ...args], { encoding: "utf8", env: { ...process.env, ...env } }).trim();
}

function createFeature(root, name, scope) {
  const dir = path.join(root, ".spec", name);
  write(path.join(dir, `${name}-SPEC.md`), "# SPEC\n");
  write(path.join(dir, `${name}-SDD.md`), "# SDD\n");
  write(path.join(dir, "TASKS.md"), "# TASKS\n\n## TASK-001 - Test\n");
  write(path.join(dir, "AGENT_RULES.md"), "# Rules\n");
  write(path.join(dir, "task-scope.json"), `${JSON.stringify(scope, null, 2)}\n`);
  write(path.join(dir, "acceptance/TASK-001.md"), "# Acceptance\n");
  write(path.join(dir, "prompts/implement-task.md"));
  write(path.join(dir, "prompts/self-review.md"));
  write(path.join(dir, "prompts/fix-check-failures.md"));
  write(path.join(dir, "scripts/check-task-scope.sh"));
  write(path.join(dir, "scripts/agent-check.sh"));
  fs.mkdirSync(path.join(dir, "reports"), { recursive: true });
  return dir;
}

const temp = fs.mkdtempSync(path.join(os.tmpdir(), "spec-harness-validator-"));
try {
  git(temp, ["init", "-q"]);
  git(temp, ["config", "user.email", "validator@example.invalid"]);
  git(temp, ["config", "user.name", "Validator Test"]);
  const task = {
    title: "Test",
    status: "pending",
    allowedFiles: ["allowed.txt", "committed.txt", ".spec/current/**"],
    forbiddenFiles: ["forbidden.txt"],
    requiresHumanConfirmation: false,
    allowedActions: ["modify", "create", "test"],
    acceptance: ".spec/current/acceptance/TASK-001.md",
    report: ".spec/current/reports/TASK-001-report.md",
    notes: [],
  };
  const currentDir = createFeature(temp, "current", { schemaVersion: 1, feature_name: "current", tasks: { "TASK-001": task } });
  const legacyTask = { ...task, acceptance: ".spec/legacy/acceptance/TASK-001.md", report: ".spec/legacy/reports/TASK-001-report.md", allowedFiles: ["allowed.txt", ".spec/legacy/**"] };
  const legacyDir = createFeature(temp, "legacy", { feature_name: "legacy", tasks: { "TASK-001": legacyTask } });
  const unsupportedDir = createFeature(temp, "unsupported", { feature: "unsupported", allowedFiles: ["allowed.txt"] });
  const invalidPatternTask = { ...task, acceptance: ".spec/invalid-pattern/acceptance/TASK-001.md", report: ".spec/invalid-pattern/reports/TASK-001-report.md", allowedFiles: ["bad[pattern"] };
  const invalidPatternDir = createFeature(temp, "invalid-pattern", { schemaVersion: 1, feature_name: "invalid-pattern", tasks: { "TASK-001": invalidPatternTask } });
  const missingAcceptanceTask = { ...task, acceptance: ".spec/missing-acceptance/acceptance/TASK-001.md", report: ".spec/missing-acceptance/reports/TASK-001-report.md", allowedFiles: [".spec/missing-acceptance/**"] };
  const missingAcceptanceDir = createFeature(temp, "missing-acceptance", { schemaVersion: 1, feature_name: "missing-acceptance", tasks: { "TASK-001": missingAcceptanceTask } });
  fs.rmSync(path.join(missingAcceptanceDir, "acceptance/TASK-001.md"));
  write(path.join(temp, "allowed.txt"), "base\n");
  write(path.join(temp, "committed.txt"), "base\n");
  write(path.join(temp, "forbidden.txt"), "base\n");
  git(temp, ["add", "-A"]);
  git(temp, ["commit", "-qm", "baseline"]);
  const baseTree = git(temp, ["rev-parse", "HEAD^{tree}"]);

  assert.equal(validateFeature(currentDir).classification, "current");
  assert.equal(validateFeature(legacyDir).classification, "legacy-compatible");
  assert.equal(validateFeature(unsupportedDir).classification, "unsupported");
  assert.equal(validateFeature(invalidPatternDir).classification, "unsupported");
  assert(validateFeature(invalidPatternDir).issues.some((issue) => issue.includes("invalid glob")));
  assert.equal(validateFeature(missingAcceptanceDir).classification, "unsupported");
  assert(validateFeature(missingAcceptanceDir).issues.some((issue) => issue.includes("missing acceptance file")));

  write(path.join(temp, "virtual-baseline.txt"), "same in synthetic baseline\n");
  const syntheticIndex = path.join(os.tmpdir(), `synthetic-${process.pid}-${Date.now()}.index`);
  git(temp, ["read-tree", "HEAD"], { GIT_INDEX_FILE: syntheticIndex });
  git(temp, ["add", "virtual-baseline.txt"], { GIT_INDEX_FILE: syntheticIndex });
  const syntheticBaseTree = git(temp, ["write-tree"], { GIT_INDEX_FILE: syntheticIndex });
  let result = checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-001", baseTree: syntheticBaseTree });
  assert.equal(result.failed, false);
  assert(!result.rows.some((row) => row.file === "virtual-baseline.txt"));
  fs.rmSync(syntheticIndex);
  fs.rmSync(path.join(temp, "virtual-baseline.txt"));

  write(path.join(temp, "committed.txt"), "committed\n");
  git(temp, ["add", "committed.txt"]);
  git(temp, ["commit", "-qm", "committed change after task baseline"]);
  result = checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-001", baseTree });
  assert.equal(result.failed, false);
  assert(result.rows.some((row) => row.file === "committed.txt" && row.result === "ALLOWED"));

  write(path.join(temp, "allowed.txt"), "changed\n");
  result = checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-001", baseTree });
  assert.equal(result.failed, false);
  assert.deepEqual(result.rows.map((row) => row.file), ["allowed.txt", "committed.txt"]);

  write(path.join(temp, "new.txt"), "untracked\n");
  result = checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-001", baseTree });
  assert.equal(result.failed, true);
  assert(result.rows.some((row) => row.file === "new.txt" && row.result === "OUT_OF_SCOPE"));

  fs.rmSync(path.join(temp, "new.txt"));
  write(path.join(temp, "forbidden.txt"), "changed\n");
  git(temp, ["add", "forbidden.txt"]);
  result = checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-001", baseTree });
  assert(result.rows.some((row) => row.file === "forbidden.txt" && row.result === "FORBIDDEN"));
  assert.throws(() => checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-UNKNOWN", baseTree }), /unknown TASK-ID/);
  assert.throws(() => checkTaskScope({ root: temp, featureDir: currentDir, taskId: "TASK-001", baseTree: null }), /reliable/);

  const evidence = {
    schemaVersion: 1,
    feature_name: "current",
    task_id: "TASK-001",
    base_sha: "a",
    head_sha: "b",
    changed_files: ["allowed.txt"],
    scope: { status: "passed", out_of_scope: [], forbidden: [] },
    checks: [{ command: "test", exit_code: 0, started_at: new Date().toISOString(), duration_ms: 1, status: "passed" }],
    review: { reviewer_type: "self-review", verdict: "通过", round: 1 },
    human_confirmation: { confirmed: true },
    exceptions: [],
  };
  assert.equal(validateEvidence(evidence).valid, true);
  evidence.checks[0].exit_code = 1;
  assert.equal(validateEvidence(evidence).valid, false);

  console.log("validator.test: PASS");
} finally {
  fs.rmSync(temp, { recursive: true, force: true });
}
