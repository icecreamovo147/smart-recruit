#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { validatePipelineState } from "./validate-pipeline-state.mjs";
import { hashJson } from "../../spec-harness/scripts/contract-utils.mjs";
import { PARITY_DIMENSIONS } from "../../spec-harness/scripts/validate-parity.mjs";

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

function makeFeature(root, state, evidence = makeEvidence(), taskOverrides = {}) {
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
        ...(taskOverrides["TASK-001"] || {}),
      },
      ...(taskOverrides.extraTasks || {}),
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

function makeV2Feature(root, overrides = {}) {
  const dir = path.join(root, ".spec", "feature-v2");
  const requiredCheck = { id: "CHECK-001", kind: "unit", blocking: true, covers: ["FR-001"], successCriteria: "unit behavior passes", command: "node test.mjs" };
  const task = {
    title: "One",
    lifecycle: "completed",
    allowedFiles: ["file.txt"],
    forbiddenFiles: [],
    allowedActions: ["modify", "create", "test"],
    requiresHumanConfirmation: false,
    requirements: ["FR-001"],
    behaviorSurfaces: ["FLOW-001"],
    dependencies: [],
    destructiveActions: [],
    requiredChecks: [requiredCheck],
    reviewPolicy: "self_allowed",
    acceptance: ".spec/feature-v2/acceptance/TASK-001.md",
    report: ".spec/feature-v2/reports/TASK-001-report.md",
  };
  const traceability = {
    schemaVersion: 2,
    feature_name: "feature-v2",
    contractRevision: 1,
    requirements: [{
      id: "FR-001",
      mandatory: true,
      status: "verified",
      design_refs: ["SDD#design"],
      task_ids: ["TASK-001"],
      acceptance_ids: ["ACASE-001"],
      check_ids: ["CHECK-001"],
      evidence_refs: ["reports/TASK-001-evidence.json#CHECK-001"],
    }],
  };
  const taskHash = hashJson(task);
  const traceabilityHash = hashJson(traceability);
  const evidence = {
    schemaVersion: 2,
    feature_name: "feature-v2",
    task_id: "TASK-001",
    contract_revision: 1,
    task_definition_hash: taskHash,
    traceability_hash: traceabilityHash,
    base_sha: "base",
    head_sha: "head",
    changed_files: ["file.txt"],
    scope: { status: "passed", out_of_scope: [], forbidden: [] },
    checks: [{ id: "CHECK-001", kind: "unit", covers: ["FR-001"], command: "node test.mjs", exit_code: 0, started_at: "2026-07-10T00:00:00Z", duration_ms: 1, status: "passed" }],
    skipped_checks: [],
    coverage_claims: [{ id: "FR-001", status: "verified" }],
    review: { reviewer_type: "self-review", outcome: "pass", round: 1, verdict: "通过" },
    human_confirmation: { required: false, confirmed: false },
    exceptions: [],
    ...(overrides.evidence || {}),
  };
  const state = {
    schemaVersion: 2,
    feature_name: "feature-v2",
    contract_revision: 1,
    current_task: "TASK-001",
    current_phase: "completed",
    completion_level: "behavior_verified",
    completed_tasks: ["TASK-001"],
    failed_tasks: [],
    blocked_tasks: [],
    skipped_human_confirmation_tasks: [],
    approved_exceptions: [],
    open_change_requests: [],
    needs_revalidation_tasks: [],
    review_round: 0,
    status: "completed",
    task_runs: {
      "TASK-001": {
        base_sha: "base",
        head_sha: "head",
        contract_revision: 1,
        task_definition_hash: taskHash,
        traceability_hash: traceabilityHash,
        scope_status: "passed",
        checks_status: "passed",
        review_outcome: "pass",
        review_verdict: "通过",
        evidence: ".spec/feature-v2/reports/TASK-001-evidence.json",
      },
    },
    ...(overrides.state || {}),
  };
  write(path.join(dir, "contract.json"), `${JSON.stringify({
    schemaVersion: 2,
    feature_name: "feature-v2",
    profile: "feature_delivery",
    contractRevision: 1,
    planningStatus: "approved",
    requiredCompletionLevel: "behavior_verified",
    assumptions: [],
    reviewPolicy: {
      plan: "self_allowed",
      highRiskTask: "independent_required",
      amendment: "independent_required",
      deleteSource: "independent_and_human",
    },
  }, null, 2)}\n`);
  write(path.join(dir, "traceability.json"), `${JSON.stringify(traceability, null, 2)}\n`);
  write(path.join(dir, "task-scope.json"), `${JSON.stringify({ schemaVersion: 2, feature_name: "feature-v2", contractRevision: 1, tasks: { "TASK-001": task } }, null, 2)}\n`);
  write(path.join(dir, "reports/TASK-001-evidence.json"), `${JSON.stringify(evidence, null, 2)}\n`);
  write(path.join(dir, "pipeline-state.json"), `${JSON.stringify(state, null, 2)}\n`);
  return dir;
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
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], human_confirmation: { confirmed: true } } } }, {}, "no confirmed_at");
assertInvalid({}, { human_confirmation: { required: true, confirmed: false } }, "human confirmation is missing");
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], evidence: null } } }, {}, "no evidence path");
assertInvalid({ failed_tasks: ["TASK-001"] }, {}, "failed_tasks");
assertInvalid({ blocked_tasks: ["TASK-001"] }, {}, "blocked_tasks");
assertInvalid({ completed_tasks: ["TASK-001", "TASK-001"] }, {}, "completed_tasks contains duplicate task");
assertInvalid({ failed_tasks: ["TASK-999"] }, {}, "failed_tasks contains unknown task");
assertInvalid({ status: "completed_with_exceptions", approved_exceptions: [{}] }, {}, "approved_exceptions[0]");
assertInvalid({
  status: "completed_with_exceptions",
  approved_exceptions: [{
    approved_object: "generic exception",
    approved_by: "user",
    approved_at: "2026-07-10T00:00:00Z",
    reason: "explicit approval fixture",
  }],
}, {}, "must include task_id or task_ids");
assertInvalid({ status: "completed", skip_human_confirm: true, skipped_human_confirmation_tasks: ["TASK-001"] }, {}, "skip_human_confirm=true");
assertInvalid({}, { task_id: "TASK-999" }, "evidence task_id does not match");
assertInvalid({}, { feature_name: "other-feature" }, "evidence feature_name does not match");
assertInvalid({ task_runs: { "TASK-001": { ...validState().task_runs["TASK-001"], base_sha: "state-base" } } }, { base_sha: "evidence-base" }, "base_sha does not match");

{
  const rootWithKnowledge = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-test-"));
  try {
    const dir = makeFeature(rootWithKnowledge, validState(), makeEvidence(), {
      "TASK-001": { requiredKnowledgeImpact: true },
    });
    const result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("knowledgeImpact must be an object")), result.issues.join("; "));
  } finally {
    fs.rmSync(rootWithKnowledge, { recursive: true, force: true });
  }
}

{
  const rootWithException = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-test-"));
  try {
    const state = validState({
      status: "completed_with_exceptions",
      approved_exceptions: [{
        task_id: "TASK-001",
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

{
  const rootWithUncoveredTask = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-test-"));
  try {
    const state = validState({
      status: "completed_with_exceptions",
      completed_tasks: ["TASK-001"],
      approved_exceptions: [{
        task_id: "TASK-001",
        approved_object: "TASK-001 exception",
        approved_by: "user",
        approved_at: "2026-07-10T00:00:00Z",
        reason: "explicit approval fixture",
      }],
    });
    const dir = makeFeature(rootWithUncoveredTask, state, makeEvidence(), {
      extraTasks: {
        "TASK-002": {
          title: "Two",
          status: "pending",
          allowedFiles: ["two.txt"],
          forbiddenFiles: [],
          requiresHumanConfirmation: false,
          acceptance: ".spec/feature/acceptance/TASK-002.md",
          report: ".spec/feature/reports/TASK-002-report.md",
        },
      },
    });
    const result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("unfinished task without approved exception: TASK-002")), result.issues.join("; "));
  } finally {
    fs.rmSync(rootWithUncoveredTask, { recursive: true, force: true });
  }
}

{
  const rootV2 = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-v2-test-"));
  try {
    const dir = makeV2Feature(rootV2);
    const result = validatePipelineState(dir);
    assert.equal(result.valid, true, result.issues.join("; "));
    assert.equal(result.schema_version, 2);
  } finally {
    fs.rmSync(rootV2, { recursive: true, force: true });
  }
}

{
  const rootV2 = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-v2-test-"));
  try {
    const dir = makeV2Feature(rootV2, {
      evidence: {
        checks: [{ id: "CHECK-001", kind: "unit", covers: ["FR-001"], status: "skipped", reason: "Docker unavailable" }],
        skipped_checks: [{ id: "CHECK-001", reason: "Docker unavailable" }],
      },
    });
    const result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("blocking required check CHECK-001 must pass")), result.issues.join("; "));
  } finally {
    fs.rmSync(rootV2, { recursive: true, force: true });
  }
}

{
  const rootV2 = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-v2-test-"));
  try {
    const dir = makeV2Feature(rootV2, { state: { open_change_requests: ["CR-0001"] } });
    const result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("completed state cannot have open change requests")), result.issues.join("; "));
  } finally {
    fs.rmSync(rootV2, { recursive: true, force: true });
  }
}

{
  const rootV2 = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-v2-test-"));
  try {
    const dir = makeV2Feature(rootV2, { state: { completion_level: "code_complete" } });
    const result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("does not satisfy required behavior_verified")), result.issues.join("; "));
  } finally {
    fs.rmSync(rootV2, { recursive: true, force: true });
  }
}

{
  const rootV2 = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-v2-carry-forward-test-"));
  try {
    const dir = makeV2Feature(rootV2);
    for (const name of ["contract.json", "traceability.json", "task-scope.json"]) {
      const file = path.join(dir, name);
      const value = JSON.parse(fs.readFileSync(file, "utf8"));
      if (name === "contract.json") value.contractRevision = 2;
      else value.contractRevision = 2;
      write(file, `${JSON.stringify(value, null, 2)}\n`);
    }
    const statePath = path.join(dir, "pipeline-state.json");
    const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
    state.contract_revision = 2;
    write(statePath, `${JSON.stringify(state, null, 2)}\n`);
    const revisionPath = path.join(dir, "revisions/REV-0002.json");
    write(revisionPath, `${JSON.stringify({
      schemaVersion: 2,
      feature_name: "feature-v2",
      revision: 2,
      change_request: "CR-0002",
      applied_at: "2026-07-10T00:00:00Z",
      updates: [],
      revalidation_tasks: [],
      carried_forward_tasks: ["TASK-001"],
    }, null, 2)}\n`);
    let result = validatePipelineState(dir);
    assert.equal(result.valid, true, result.issues.join("; "));
    const revision = JSON.parse(fs.readFileSync(revisionPath, "utf8"));
    revision.carried_forward_tasks = [];
    write(revisionPath, `${JSON.stringify(revision, null, 2)}\n`);
    result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("not explicitly carried forward")), result.issues.join("; "));
  } finally {
    fs.rmSync(rootV2, { recursive: true, force: true });
  }
}

{
  const rootV2 = fs.mkdtempSync(path.join(os.tmpdir(), "pipeline-state-v2-migration-test-"));
  try {
    const dir = makeV2Feature(rootV2);
    const contractPath = path.join(dir, "contract.json");
    const contract = JSON.parse(fs.readFileSync(contractPath, "utf8"));
    contract.profile = "behavior_preserving_migration";
    contract.requiredCompletionLevel = "parity_verified";
    contract.baseline = { ref: "dev", sha: "004db546", tree: "tree004db546", immutable: true };
    write(contractPath, `${JSON.stringify(contract, null, 2)}\n`);
    const statePath = path.join(dir, "pipeline-state.json");
    const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
    state.completion_level = "parity_verified";
    write(statePath, `${JSON.stringify(state, null, 2)}\n`);
    const dimensions = Object.fromEntries(PARITY_DIMENSIONS.map((name) => [name, { status: "unknown", evidence_refs: [] }]));
    const manifest = {
      schemaVersion: 2,
      feature_name: "feature-v2",
      contractRevision: 1,
      baseline: { ref: "dev", sha: "004db546", tree: "tree004db546", immutable: true },
      capabilities: [{
        id: "CAP-001",
        mandatory: true,
        source: { entrypoints: ["HTTP-1"], implementation_refs: ["dev:file.go#Fn"], tests: ["dev:file_test.go"] },
        target: { implementation_refs: ["target:file.go#Fn"] },
        dimensions,
      }],
    };
    write(path.join(dir, "baseline/behavior-manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
    let result = validatePipelineState(dir);
    assert.equal(result.valid, false);
    assert(result.issues.some((issue) => issue.includes("is not parity verified")), result.issues.join("; "));
    for (const dimension of PARITY_DIMENSIONS) manifest.capabilities[0].dimensions[dimension] = { status: "verified_equal", evidence_refs: [`CHECK-${dimension}`] };
    write(path.join(dir, "baseline/behavior-manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
    result = validatePipelineState(dir);
    assert.equal(result.valid, true, result.issues.join("; "));
  } finally {
    fs.rmSync(rootV2, { recursive: true, force: true });
  }
}

console.log("pipeline-state.test: PASS");
