#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { validatePipelineState } from "./validate-pipeline-state.mjs";

function write(file, content) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, content);
}

function makeEvidence(overrides = {}) {
  return {
    schemaVersion: 1,
    feature_name: "feature",
    task_id: "TASK-001",
    base_sha: "base",
    head_sha: "head",
    changed_files: ["file.txt"],
    scope: { status: "passed", out_of_scope: [], forbidden: [] },
    checks: [{ command: "test", exit_code: 0, started_at: "2026-07-10T00:00:00Z", duration_ms: 1, status: "passed" }],
    review: { reviewer_type: "self-review", round: 1, verdict: "通过" },
    human_confirmation: { required: true, confirmed: true, confirmed_at: "2026-07-10T00:00:00Z" },
    exceptions: [],
    ...overrides,
  };
}

function makeFeature(root, state, evidence = makeEvidence()) {
  const dir = path.join(root, ".spec", "feature");
  write(path.join(dir, "task-scope.json"), `${JSON.stringify({
    schemaVersion: 1,
    feature_name: "feature",
    tasks: {
      "TASK-001": {
        title: "One",
        status: "pending",
        allowedFiles: ["file.txt"],
        forbiddenFiles: [],
        requiresHumanConfirmation: true,
        acceptance: ".spec/feature/acceptance/TASK-001.md",
        report: ".spec/feature/reports/TASK-001-report.md",
      },
    },
  }, null, 2)}\n`);
  write(path.join(dir, "reports/TASK-001-evidence.json"), `${JSON.stringify(evidence, null, 2)}\n`);
  write(path.join(dir, "pipeline-state.json"), `${JSON.stringify(state, null, 2)}\n`);
  return dir;
}

function validState(overrides = {}) {
  return {
    schemaVersion: 1,
    feature_name: "feature",
    current_task: "TASK-001",
    current_phase: "completed",
    completed_tasks: ["TASK-001"],
    failed_tasks: [],
    blocked_tasks: [],
    review_round: 0,
    status: "completed",
    task_runs: {
      "TASK-001": {
        base_sha: "base",
        head_sha: "head",
        scope_status: "passed",
        checks_status: "passed",
        review_verdict: "通过",
        human_confirmation: { confirmed: true, confirmed_at: "2026-07-10T00:00:00Z" },
        evidence: ".spec/feature/reports/TASK-001-evidence.json",
      },
    },
    ...overrides,
  };
}

function assertInvalid(stateOverride, evidenceOverride, expected) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-test-"));
  try {
    const dir = makeFeature(root, validState(stateOverride), makeEvidence(evidenceOverride));
    const result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes(expected)), `${expected} not found in ${result.issues.join("; ")}`);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

const root = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-test-"));
try {
  const dir = makeFeature(root, validState());
  assert.equal(validatePipelineState(dir).valid, true);
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}

assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], scope_status: "failed" } } }, {}, "scope_status must be passed");
assertInvalid({}, { scope: { status: "failed", out_of_scope: ["x"], forbidden: [] } }, "evidence scope.status must be passed");
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], checks_status: "failed" } } }, {}, "checks_status must be passed");
assertInvalid({}, { checks: [{ command: "test", exit_code: 1, started_at: "2026-07-10T00:00:00Z", duration_ms: 1, status: "failed" }] }, "review contains failed checks");
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], review_verdict: "不通过" } } }, {}, "review_verdict must be 通过");
assertInvalid({}, { review: { reviewer_type: "self-review", round: 1, verdict: "不通过" } }, "review verdict must be 通过");
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], human_confirmation: { confirmed: false } } } }, {}, "requires human confirmation");
assertInvalid({}, { human_confirmation: { required: true, confirmed: false } }, "human confirmation is missing");
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], evidence: null } } }, {}, "no evidence path");
assertInvalid({ failed_tasks: ["TASK-001"] }, {}, "failed_tasks");
assertInvalid({ blocked_tasks: ["TASK-001"] }, {}, "blocked_tasks");
assertInvalid({ status: "completed_with_exceptions", approved_exceptions: [{}] }, {}, "approved_exceptions[0]");
assertInvalid({ status: "completed", skip_human_confirm: true, skipped_human_confirmation_tasks: ["TASK-001"] }, {}, "skip_human_confirm=true");

{
  const rootWithException = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-test-"));
  try {
    const state = validState({
      status: "completed_with_exceptions",
      approved_exceptions: [{
        approved_object: "TASK-001 check exception",
        approved_by: "user",
        approved_at: "2026-07-10T00:00:00Z",
        reason: "explicit approval fixture",
      }],
    });
    const dir = makeFeature(rootWithException, state);
    assert.equal(validatePipelineState(dir).valid, true);
  } finally {
    fs.rmSync(rootWithException, { recursive: true, force: true });
  }
}

console.log("pipeline-state.test: PASS");
