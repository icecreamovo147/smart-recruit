#!/usr/bin/env node

import fs from "node:fs";
import process from "node:process";
import { isObject, validateStringArray } from "./contract-utils.mjs";

const REVIEW_OUTCOMES = new Set(["pass", "implementation_defect", "contract_gap", "requirement_conflict", "environment_blocker"]);
const CHECK_KINDS = new Set(["static", "build", "unit", "integration", "contract", "differential", "e2e", "manual"]);
const COVERAGE_STATUSES = new Set(["verified", "approved_delta", "missing", "unknown", "blocked"]);

function validateKnowledgeImpact(evidence, issues, options) {
  const allowedKnowledgeResults = new Set(["none", "update_required", "candidate_required", "stale_detected", "conflict_detected", "coverage_gap"]);
  const allowedDocumentVerdicts = new Set(["UNCHANGED", "UPDATED", "STALE", "CANDIDATE", "CONFLICT"]);
  if (!options.requireKnowledgeImpact && !("knowledgeImpact" in evidence)) return;
  const impact = evidence.knowledgeImpact;
  if (!isObject(impact)) {
    issues.push("knowledgeImpact must be an object");
    return;
  }
  if (!allowedKnowledgeResults.has(impact.result)) issues.push(`knowledgeImpact.result is invalid: ${impact.result}`);
  if (!Array.isArray(impact.triggeredBy)) issues.push("knowledgeImpact.triggeredBy must be an array");
  if (!Array.isArray(impact.reviewResults)) issues.push("knowledgeImpact.reviewResults must be an array");
  if (typeof impact.coverageGap !== "boolean") issues.push("knowledgeImpact.coverageGap must be boolean");
  if (!Number.isInteger(impact.validationExitCode)) issues.push("knowledgeImpact.validationExitCode must be an integer");
  for (const [index, review] of (impact.reviewResults || []).entries()) {
    if (!isObject(review)) {
      issues.push(`knowledgeImpact.reviewResults[${index}] must be an object`);
      continue;
    }
    if (typeof review.document !== "string" || review.document.length === 0) issues.push(`knowledgeImpact.reviewResults[${index}].document is required`);
    if (!allowedDocumentVerdicts.has(review.verdict)) issues.push(`knowledgeImpact.reviewResults[${index}].verdict is invalid`);
    if (!Array.isArray(review.evidence) || review.evidence.length === 0) issues.push(`knowledgeImpact.reviewResults[${index}].evidence must be a non-empty array`);
  }
  if (impact.result === "none" && impact.coverageGap) issues.push("knowledgeImpact cannot be none when coverageGap is true");
  if (evidence.review?.verdict === "通过") {
    if (new Set(["stale_detected", "conflict_detected"]).has(impact.result)) issues.push("passing review cannot contain stale_detected or conflict_detected knowledgeImpact");
    if ((impact.reviewResults || []).some((review) => ["STALE", "CONFLICT"].includes(review.verdict))) issues.push("passing review cannot contain STALE or CONFLICT knowledge document verdicts");
  }
}

function validateV1Checks(evidence, issues) {
  if (!Array.isArray(evidence.checks) || evidence.checks.length === 0) issues.push("checks must be a non-empty array");
  for (const [index, check] of (evidence.checks || []).entries()) {
    if (!check.command) issues.push(`checks[${index}] has no command`);
    if (!Number.isInteger(check.exit_code)) issues.push(`checks[${index}] exit_code must be an integer`);
    if (!check.started_at) issues.push(`checks[${index}] has no started_at`);
    if (!Number.isFinite(check.duration_ms) || check.duration_ms < 0) issues.push(`checks[${index}] duration_ms must be non-negative`);
    const expectedStatus = check.exit_code === 0 ? "passed" : "failed";
    if (check.status !== expectedStatus) issues.push(`checks[${index}] status contradicts exit_code`);
  }
}

function validateV2Checks(evidence, issues, requiredChecks) {
  if (!Array.isArray(evidence.checks) || evidence.checks.length === 0) issues.push("checks must be a non-empty array");
  if (!Array.isArray(evidence.skipped_checks)) issues.push("skipped_checks must be an array");
  for (const [index, skipped] of (evidence.skipped_checks || []).entries()) {
    if (!isObject(skipped) || typeof skipped.id !== "string" || skipped.id.length === 0 || typeof skipped.reason !== "string" || skipped.reason.length === 0) {
      issues.push(`skipped_checks[${index}] must include id and reason`);
    }
  }
  const byId = new Map();
  const skippedReasons = new Map((evidence.skipped_checks || []).map((item) => [item?.id, item?.reason]));
  for (const [index, check] of (evidence.checks || []).entries()) {
    const label = `checks[${index}]`;
    if (!isObject(check)) {
      issues.push(`${label} must be an object`);
      continue;
    }
    if (typeof check.id !== "string" || check.id.length === 0) issues.push(`${label}.id is required`);
    else if (byId.has(check.id)) issues.push(`duplicate evidence check id: ${check.id}`);
    else byId.set(check.id, check);
    if (!CHECK_KINDS.has(check.kind)) issues.push(`${label}.kind is invalid`);
    validateStringArray(check.covers, `${label}.covers`, issues, { nonEmpty: true });
    if (!new Set(["passed", "failed", "skipped", "not_applicable"]).has(check.status)) issues.push(`${label}.status is invalid`);
    if (new Set(["passed", "failed"]).has(check.status)) {
      if (!check.command) issues.push(`${label} has no command`);
      if (!Number.isInteger(check.exit_code)) issues.push(`${label}.exit_code must be an integer`);
      if (!check.started_at) issues.push(`${label} has no started_at`);
      if (!Number.isFinite(check.duration_ms) || check.duration_ms < 0) issues.push(`${label}.duration_ms must be non-negative`);
      const expectedStatus = check.exit_code === 0 ? "passed" : "failed";
      if (check.status !== expectedStatus) issues.push(`${label}.status contradicts exit_code`);
    } else if (!check.reason && !skippedReasons.get(check.id)) issues.push(`${label} skipped/not_applicable check requires a reason`);
  }

  for (const required of requiredChecks || []) {
    const actual = byId.get(required.id);
    if (!actual) {
      issues.push(`missing required check evidence: ${required.id}`);
      continue;
    }
    if (actual.kind !== required.kind) issues.push(`required check ${required.id} kind mismatch: expected ${required.kind}, got ${actual.kind}`);
    for (const coverage of required.covers || []) if (!(actual.covers || []).includes(coverage)) issues.push(`required check ${required.id} does not claim coverage for ${coverage}`);
    if (required.blocking && actual.status !== "passed") issues.push(`blocking required check ${required.id} must pass; got ${actual.status}`);
  }
}

export function validateEvidence(evidence, options = {}) {
  const issues = [];
  const requiredFields = ["schemaVersion", "feature_name", "task_id", "base_sha", "head_sha", "changed_files", "scope", "checks", "review", "human_confirmation", "exceptions"];
  for (const field of requiredFields) if (!(field in evidence)) issues.push(`missing field: ${field}`);
  if (![1, 2].includes(evidence.schemaVersion)) issues.push("schemaVersion must be 1 or 2");
  if (!Array.isArray(evidence.changed_files)) issues.push("changed_files must be an array");
  if (!isObject(evidence.scope) || !new Set(["passed", "failed", "pending"]).has(evidence.scope.status)) issues.push("scope.status is invalid");
  if (evidence.scope?.status === "passed" && ((evidence.scope.out_of_scope || []).length > 0 || (evidence.scope.forbidden || []).length > 0)) issues.push("passed scope contains forbidden or out-of-scope files");

  if (evidence.schemaVersion === 1) validateV1Checks(evidence, issues);
  if (evidence.schemaVersion === 2) {
    for (const field of ["contract_revision", "task_definition_hash", "traceability_hash", "coverage_claims", "skipped_checks"]) {
      if (!(field in evidence)) issues.push(`missing v2 field: ${field}`);
    }
    if (!Number.isInteger(evidence.contract_revision) || evidence.contract_revision < 1) issues.push("contract_revision must be a positive integer");
    if (typeof evidence.task_definition_hash !== "string" || evidence.task_definition_hash.length === 0) issues.push("task_definition_hash is required");
    if (typeof evidence.traceability_hash !== "string" || evidence.traceability_hash.length === 0) issues.push("traceability_hash is required");
    validateV2Checks(evidence, issues, options.requiredChecks || []);
    if (!Array.isArray(evidence.coverage_claims)) issues.push("coverage_claims must be an array");
    for (const [index, claim] of (evidence.coverage_claims || []).entries()) {
      if (!isObject(claim) || typeof claim.id !== "string" || !COVERAGE_STATUSES.has(claim.status)) issues.push(`coverage_claims[${index}] is invalid`);
      if (claim?.status === "approved_delta" && (typeof claim.approval_ref !== "string" || claim.approval_ref.length === 0)) issues.push(`coverage_claims[${index}] approved_delta requires approval_ref`);
    }
    if (!REVIEW_OUTCOMES.has(evidence.review?.outcome)) issues.push("review.outcome is invalid");
    if (!options.allowPendingReview && evidence.review?.outcome !== "pass") issues.push("review outcome must be pass");
    if (evidence.review?.outcome === "pass" && (evidence.coverage_claims || []).some((claim) => ["missing", "unknown", "blocked"].includes(claim.status))) {
      issues.push("passing review contains unresolved coverage claims");
    }
  }

  if (!options.allowPendingReview && evidence.review?.verdict !== "通过") issues.push("review verdict must be 通过");
  if (evidence.review?.verdict === "通过" && (evidence.checks || []).some((check) => check.status === "failed" || check.exit_code > 0)) issues.push("passing review contains failed checks");
  if (!evidence.human_confirmation?.confirmed && !options.allowMissingConfirmation) issues.push("required human confirmation is missing");
  if (!Array.isArray(evidence.exceptions)) issues.push("exceptions must be an array");
  validateKnowledgeImpact(evidence, issues, options);
  return { valid: issues.length === 0, issues };
}

function parseArgs(argv) {
  const args = { file: null, allowPendingReview: false, allowMissingConfirmation: false, requireKnowledgeImpact: false, requiredChecksFile: null, taskScopeFile: null, taskId: null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--file") args.file = argv[++index];
    else if (argv[index] === "--allow-pending-review") args.allowPendingReview = true;
    else if (argv[index] === "--allow-missing-confirmation") args.allowMissingConfirmation = true;
    else if (argv[index] === "--require-knowledge-impact") args.requireKnowledgeImpact = true;
    else if (argv[index] === "--required-checks") args.requiredChecksFile = argv[++index];
    else if (argv[index] === "--task-scope") args.taskScopeFile = argv[++index];
    else if (argv[index] === "--task") args.taskId = argv[++index];
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!args.file) throw new Error("--file is required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const evidence = JSON.parse(fs.readFileSync(args.file, "utf8"));
    let requiredChecks = args.requiredChecksFile ? JSON.parse(fs.readFileSync(args.requiredChecksFile, "utf8")) : undefined;
    if (args.taskScopeFile) {
      const taskScope = JSON.parse(fs.readFileSync(args.taskScopeFile, "utf8"));
      const taskId = args.taskId || evidence.task_id;
      if (!taskScope.tasks?.[taskId]) throw new Error(`task not found in task scope: ${taskId}`);
      requiredChecks = taskScope.tasks[taskId].requiredChecks;
    }
    const result = validateEvidence(evidence, { ...args, requiredChecks });
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      for (const issue of result.issues) console.error(`error: ${issue}`);
      console.log(`evidence_result: ${result.valid ? "PASS" : "FAIL"}`);
    }
    process.exit(result.valid ? 0 : 1);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
