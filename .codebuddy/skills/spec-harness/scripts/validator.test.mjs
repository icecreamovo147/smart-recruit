#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { validateFeature } from "./validate-feature.mjs";
import { checkTaskScope } from "./check-task-scope.mjs";
import { validateEvidence } from "./validate-evidence.mjs";
import { validateTraceability } from "./validate-traceability.mjs";
import { validateParityManifest, PARITY_DIMENSIONS } from "./validate-parity.mjs";
import { validateAmendment } from "./validate-amendment.mjs";
import { applyAmendment } from "./apply-amendment.mjs";
import { hashFile, hashJson } from "./contract-utils.mjs";

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
  const knowledgeTask = {
    ...task,
    acceptance: ".spec/knowledge-feature/acceptance/TASK-001.md",
    report: ".spec/knowledge-feature/reports/TASK-001-report.md",
    allowedFiles: ["allowed.txt", ".spec/knowledge-feature/**"],
    requiredKnowledgeImpact: true,
    knowledge: {
      review: [".knowledge/architecture/**"],
      modify: [".knowledge/runbooks/*.md"],
      candidate: [".knowledge/inbox/**"],
    },
  };
  const currentDir = createFeature(temp, "current", { schemaVersion: 1, feature_name: "current", tasks: { "TASK-001": task } });
  const knowledgeDir = createFeature(temp, "knowledge-feature", { schemaVersion: 1, feature_name: "knowledge-feature", tasks: { "TASK-001": knowledgeTask } });
  const invalidKnowledgeTask = {
    ...knowledgeTask,
    acceptance: ".spec/invalid-knowledge/acceptance/TASK-001.md",
    report: ".spec/invalid-knowledge/reports/TASK-001-report.md",
    knowledge: { review: ["bad[glob"], extra: [] },
  };
  const invalidKnowledgeDir = createFeature(temp, "invalid-knowledge", { schemaVersion: 1, feature_name: "invalid-knowledge", tasks: { "TASK-001": invalidKnowledgeTask } });
  const legacyTask = { ...task, acceptance: ".spec/legacy/acceptance/TASK-001.md", report: ".spec/legacy/reports/TASK-001-report.md", allowedFiles: ["allowed.txt", ".spec/legacy/**"] };
  const legacyDir = createFeature(temp, "legacy", { feature_name: "legacy", tasks: { "TASK-001": legacyTask } });
  const unsupportedDir = createFeature(temp, "unsupported", { feature: "unsupported", allowedFiles: ["allowed.txt"] });
  const invalidPatternTask = { ...task, acceptance: ".spec/invalid-pattern/acceptance/TASK-001.md", report: ".spec/invalid-pattern/reports/TASK-001-report.md", allowedFiles: ["bad[pattern"] };
  const invalidPatternDir = createFeature(temp, "invalid-pattern", { schemaVersion: 1, feature_name: "invalid-pattern", tasks: { "TASK-001": invalidPatternTask } });
  const invalidRootDocsTask = { ...task, acceptance: ".spec/invalid-root-docs/acceptance/TASK-001.md", report: ".spec/invalid-root-docs/reports/TASK-001-report.md", allowedFiles: ["docs/architecture/**"] };
  const invalidRootDocsDir = createFeature(temp, "invalid-root-docs", { schemaVersion: 1, feature_name: "invalid-root-docs", tasks: { "TASK-001": invalidRootDocsTask } });
  const missingAcceptanceTask = { ...task, acceptance: ".spec/missing-acceptance/acceptance/TASK-001.md", report: ".spec/missing-acceptance/reports/TASK-001-report.md", allowedFiles: [".spec/missing-acceptance/**"] };
  const missingAcceptanceDir = createFeature(temp, "missing-acceptance", { schemaVersion: 1, feature_name: "missing-acceptance", tasks: { "TASK-001": missingAcceptanceTask } });
  fs.rmSync(path.join(missingAcceptanceDir, "acceptance/TASK-001.md"));
  const requiredCheck = { id: "CHECK-001", kind: "unit", blocking: true, covers: ["FR-001"], successCriteria: "unit behavior passes", command: "node test.mjs" };
  const v2Task = {
    ...task,
    lifecycle: "ready",
    requirements: ["FR-001"],
    behaviorSurfaces: ["FLOW-001"],
    dependencies: [],
    destructiveActions: [],
    requiredChecks: [requiredCheck],
    reviewPolicy: "self_allowed",
    acceptance: ".spec/v2-feature/acceptance/TASK-001.md",
    report: ".spec/v2-feature/reports/TASK-001-report.md",
    allowedFiles: ["allowed.txt"],
  };
  const v2Dir = createFeature(temp, "v2-feature", { schemaVersion: 2, feature_name: "v2-feature", contractRevision: 1, tasks: { "TASK-001": v2Task } });
  const v2Contract = {
    schemaVersion: 2,
    feature_name: "v2-feature",
    profile: "feature_delivery",
    contractRevision: 1,
    planningStatus: "approved",
    requiredCompletionLevel: "behavior_verified",
    assumptions: [],
    reviewPolicy: {
      plan: "independent_required",
      highRiskTask: "independent_required",
      amendment: "independent_required",
      deleteSource: "independent_and_human",
    },
  };
  write(path.join(v2Dir, "contract.json"), `${JSON.stringify(v2Contract, null, 2)}\n`);
  const traceability = {
    schemaVersion: 2,
    feature_name: "v2-feature",
    contractRevision: 1,
    requirements: [{
      id: "FR-001",
      mandatory: true,
      status: "planned",
      design_refs: ["SDD#design"],
      task_ids: ["TASK-001"],
      acceptance_ids: ["ACASE-001"],
      check_ids: ["CHECK-001"],
      evidence_refs: [],
    }],
  };
  write(path.join(v2Dir, "traceability.json"), `${JSON.stringify(traceability, null, 2)}\n`);
  write(path.join(v2Dir, "acceptance/TASK-001.md"), "# Acceptance\n\n## CHECK-001\n");
  write(path.join(v2Dir, "prompts/propose-amendment.md"));
  write(path.join(v2Dir, "prompts/review-amendment.md"));
  write(path.join(v2Dir, "prompts/reconcile-plan.md"));
  for (const name of ["changes", "revisions", "reviews"]) fs.mkdirSync(path.join(v2Dir, name), { recursive: true });
  write(path.join(v2Dir, "reviews/PLAN-REV-0001.json"), `${JSON.stringify({
    schemaVersion: 2,
    feature_name: "v2-feature",
    contractRevision: 1,
    contract_hash: hashJson(v2Contract),
    outcome: "pass",
    verdict: "通过",
    reviewer_type: "independent_agent",
    planner_run_id: "planner-1",
    reviewer_run_id: "reviewer-1",
  }, null, 2)}\n`);
  write(path.join(temp, "allowed.txt"), "base\n");
  write(path.join(temp, "committed.txt"), "base\n");
  write(path.join(temp, "forbidden.txt"), "base\n");
  git(temp, ["add", "-A"]);
  git(temp, ["commit", "-qm", "baseline"]);
  const baseTree = git(temp, ["rev-parse", "HEAD^{tree}"]);

  assert.equal(validateFeature(currentDir).classification, "current");
  assert.equal(validateFeature(knowledgeDir).classification, "current");
  assert.equal(validateFeature(invalidKnowledgeDir).classification, "unsupported");
  assert(validateFeature(invalidKnowledgeDir).issues.some((issue) => issue.includes("knowledge.review has invalid glob")));
  assert.equal(validateFeature(legacyDir).classification, "legacy-compatible");
  assert.equal(validateFeature(unsupportedDir).classification, "unsupported");
  assert.equal(validateFeature(invalidPatternDir).classification, "unsupported");
  assert(validateFeature(invalidPatternDir).issues.some((issue) => issue.includes("invalid glob")));
  assert.equal(validateFeature(invalidRootDocsDir).classification, "unsupported");
  assert(validateFeature(invalidRootDocsDir).issues.some((issue) => issue.includes("repository-root docs")));
  assert.equal(validateFeature(missingAcceptanceDir).classification, "unsupported");
  assert(validateFeature(missingAcceptanceDir).issues.some((issue) => issue.includes("missing acceptance file")));
  assert.equal(validateFeature(v2Dir).classification, "current", validateFeature(v2Dir).issues.join("; "));
  assert.equal(validateFeature(v2Dir).schema_version, 2);

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
  assert.equal(validateEvidence({ ...evidence, knowledgeImpact: {
    result: "update_required",
    triggeredBy: ["covered-path-changed"],
    reviewResults: [{ document: ".knowledge/runbooks/local-development.md", verdict: "UNCHANGED", evidence: ["README.md"] }],
    coverageGap: false,
    validationExitCode: 0,
  } }).valid, true);
  assert.equal(validateEvidence(evidence, { requireKnowledgeImpact: true }).valid, false);
  assert.equal(validateEvidence({ ...evidence, knowledgeImpact: {
    result: "conflict_detected",
    triggeredBy: ["covered-path-changed"],
    reviewResults: [{ document: ".knowledge/architecture/system-overview.md", verdict: "CONFLICT", evidence: ["AGENTS.md"] }],
    coverageGap: false,
    validationExitCode: 0,
  } }).valid, false);
  evidence.checks[0].exit_code = 1;
  assert.equal(validateEvidence(evidence).valid, false);

  const v2Evidence = {
    schemaVersion: 2,
    feature_name: "v2-feature",
    task_id: "TASK-001",
    contract_revision: 1,
    task_definition_hash: "task-hash",
    traceability_hash: "trace-hash",
    base_sha: "a",
    head_sha: "b",
    changed_files: ["allowed.txt"],
    scope: { status: "passed", out_of_scope: [], forbidden: [] },
    checks: [{ id: "CHECK-001", kind: "unit", covers: ["FR-001"], command: "node test.mjs", exit_code: 0, started_at: new Date().toISOString(), duration_ms: 1, status: "passed" }],
    skipped_checks: [],
    coverage_claims: [{ id: "FR-001", status: "verified" }],
    review: { reviewer_type: "self-review", outcome: "pass", verdict: "通过", round: 1 },
    human_confirmation: { confirmed: true },
    exceptions: [],
  };
  assert.equal(validateEvidence(v2Evidence, { requiredChecks: [requiredCheck] }).valid, true);
  const missingRequired = validateEvidence({ ...v2Evidence, checks: [{ ...v2Evidence.checks[0], id: "CHECK-OTHER" }] }, { requiredChecks: [requiredCheck] });
  assert.equal(missingRequired.valid, false);
  assert(missingRequired.issues.some((issue) => issue.includes("missing required check evidence")));
  const skippedBlocking = validateEvidence({
    ...v2Evidence,
    checks: [{ id: "CHECK-001", kind: "unit", covers: ["FR-001"], status: "skipped", reason: "environment unavailable" }],
    skipped_checks: [{ id: "CHECK-001", reason: "environment unavailable" }],
  }, { requiredChecks: [requiredCheck] });
  assert.equal(skippedBlocking.valid, false);
  assert(skippedBlocking.issues.some((issue) => issue.includes("blocking required check CHECK-001 must pass")));
  const staticSubstitution = validateEvidence({ ...v2Evidence, checks: [{ ...v2Evidence.checks[0], kind: "static" }] }, { requiredChecks: [requiredCheck] });
  assert.equal(staticSubstitution.valid, false);
  assert(staticSubstitution.issues.some((issue) => issue.includes("kind mismatch")));

  assert.equal(validateTraceability(traceability).valid, true);
  assert.equal(validateTraceability(traceability, { requireVerified: true }).valid, false);
  const verifiedTraceability = structuredClone(traceability);
  verifiedTraceability.requirements[0].status = "verified";
  verifiedTraceability.requirements[0].evidence_refs = ["reports/TASK-001-evidence.json#CHECK-001"];
  assert.equal(validateTraceability(verifiedTraceability, { requireVerified: true }).valid, true);

  const dimensions = Object.fromEntries(PARITY_DIMENSIONS.map((name) => [name, { status: "unknown", evidence_refs: [] }]));
  const parity = {
    schemaVersion: 2,
    feature_name: "migration",
    contractRevision: 1,
    baseline: { ref: "dev", sha: "004db546", tree: "tree004db546", immutable: true },
    capabilities: [{
      id: "CAP-001",
      mandatory: true,
      source: { entrypoints: ["HTTP-1"], implementation_refs: ["dev:file.go#Fn"], tests: ["dev:file_test.go"] },
      target: { implementation_refs: [] },
      dimensions,
      events: [{ routing_key: "resume.parse", producer_refs: ["producer.go"], consumer_refs: ["consumer.go"], final_effect_checks: ["CHECK-EVENT-1"] }],
    }],
  };
  assert.equal(validateParityManifest(parity).valid, true);
  assert.equal(validateParityManifest(parity, { requireVerified: true }).valid, false);
  const orphanEvent = structuredClone(parity);
  orphanEvent.capabilities[0].events[0].consumer_refs = [];
  assert.equal(validateParityManifest(orphanEvent).valid, false);

  const amendment = {
    schemaVersion: 2,
    changeRequestId: "CR-0001",
    feature_name: "v2-feature",
    baseRevision: 1,
    status: "approved",
    classification: "contract_gap",
    approvalLevel: "L2",
    reason: "public contract correction",
    evidenceRefs: ["reports/TASK-001-report.md"],
    trigger: { task_id: "TASK-001", phase: "implement" },
    impact: { requirements: ["FR-001"], tasks: ["TASK-001"], completedTasksRequiringRevalidation: [] },
    revalidationPlan: ["TASK-001"],
    resultingRevision: 2,
    approval: { approved_by: "user", approved_at: new Date().toISOString() },
  };
  assert.equal(validateAmendment(amendment).valid, true);
  assert.equal(validateAmendment({ ...amendment, approval: { approved_by: "agent", approved_at: new Date().toISOString() } }).valid, false);

  const stagedDir = path.join(v2Dir, "changes", "CR-0002");
  const stagedContract = JSON.parse(fs.readFileSync(path.join(v2Dir, "contract.json"), "utf8"));
  stagedContract.contractRevision = 2;
  const stagedScope = JSON.parse(fs.readFileSync(path.join(v2Dir, "task-scope.json"), "utf8"));
  stagedScope.contractRevision = 2;
  const stagedTraceability = JSON.parse(fs.readFileSync(path.join(v2Dir, "traceability.json"), "utf8"));
  stagedTraceability.contractRevision = 2;
  write(path.join(stagedDir, "contract.json"), `${JSON.stringify(stagedContract, null, 2)}\n`);
  write(path.join(stagedDir, "task-scope.json"), `${JSON.stringify(stagedScope, null, 2)}\n`);
  write(path.join(stagedDir, "traceability.json"), `${JSON.stringify(stagedTraceability, null, 2)}\n`);
  const updates = ["contract.json", "task-scope.json", "traceability.json"].map((target) => ({
    target,
    staged_file: `changes/CR-0002/${target}`,
    before_sha256: hashFile(path.join(v2Dir, target)),
    after_sha256: hashFile(path.join(stagedDir, target)),
  }));
  const applicableAmendment = {
    ...amendment,
    changeRequestId: "CR-0002",
    classification: "scope_correction",
    approvalLevel: "L0",
    reason: "fixture revision",
    resultingRevision: 2,
    approval: { approved_by: "independent-reviewer", approved_at: new Date().toISOString() },
    updates,
  };
  const amendmentPath = path.join(v2Dir, "changes", "CR-0002.json");
  write(amendmentPath, `${JSON.stringify(applicableAmendment, null, 2)}\n`);
  assert.equal(applyAmendment({ featureDir: v2Dir, crFile: amendmentPath }).applied, false);
  assert.equal(applyAmendment({ featureDir: v2Dir, crFile: amendmentPath, apply: true }).applied, true);
  assert.equal(JSON.parse(fs.readFileSync(path.join(v2Dir, "contract.json"), "utf8")).contractRevision, 2);
  assert.equal(JSON.parse(fs.readFileSync(amendmentPath, "utf8")).status, "applied");
  assert.equal(validateFeature(v2Dir).classification, "current", validateFeature(v2Dir).issues.join("; "));

  console.log("validator.test: PASS");
} finally {
  fs.rmSync(temp, { recursive: true, force: true });
}
