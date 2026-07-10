#!/usr/bin/env node

import fs from "node:fs";
import process from "node:process";

export function validateEvidence(evidence, options = {}) {
  const issues = [];
  const allowedKnowledgeResults = new Set(["none", "update_required", "candidate_required", "stale_detected", "conflict_detected", "coverage_gap"]);
  const allowedDocumentVerdicts = new Set(["UNCHANGED", "UPDATED", "STALE", "CANDIDATE", "CONFLICT"]);
  for (const field of ["schemaVersion", "feature_name", "task_id", "base_sha", "head_sha", "changed_files", "scope", "checks", "review", "human_confirmation", "exceptions"]) {
    if (!(field in evidence)) issues.push(`missing field: ${field}`);
  }
  if (evidence.schemaVersion !== 1) issues.push("schemaVersion must be 1");
  if (!Array.isArray(evidence.changed_files)) issues.push("changed_files must be an array");
  if (!Array.isArray(evidence.checks) || evidence.checks.length === 0) issues.push("checks must be a non-empty array");
  for (const [index, check] of (evidence.checks || []).entries()) {
    if (!check.command) issues.push(`checks[${index}] has no command`);
    if (!Number.isInteger(check.exit_code)) issues.push(`checks[${index}] exit_code must be an integer`);
    if (!check.started_at) issues.push(`checks[${index}] has no started_at`);
    if (!Number.isFinite(check.duration_ms) || check.duration_ms < 0) issues.push(`checks[${index}] duration_ms must be non-negative`);
    const expectedStatus = check.exit_code === 0 ? "passed" : "failed";
    if (check.status !== expectedStatus) issues.push(`checks[${index}] status contradicts exit_code`);
  }
  if (!evidence.scope || !["passed", "failed", "pending"].includes(evidence.scope.status)) issues.push("scope.status is invalid");
  if (evidence.scope?.status === "passed" && ((evidence.scope.out_of_scope || []).length > 0 || (evidence.scope.forbidden || []).length > 0)) {
    issues.push("passed scope contains forbidden or out-of-scope files");
  }
  if (!options.allowPendingReview && evidence.review?.verdict !== "通过") issues.push("review verdict must be 通过");
  if (evidence.review?.verdict === "通过" && (evidence.checks || []).some((check) => check.exit_code !== 0)) issues.push("passing review contains failed checks");
  if (!evidence.human_confirmation?.confirmed && !options.allowMissingConfirmation) issues.push("required human confirmation is missing");
  if (!Array.isArray(evidence.exceptions)) issues.push("exceptions must be an array");
  if (options.requireKnowledgeImpact || "knowledgeImpact" in evidence) {
    const impact = evidence.knowledgeImpact;
    if (!impact || typeof impact !== "object" || Array.isArray(impact)) {
      issues.push("knowledgeImpact must be an object");
    } else {
      if (!allowedKnowledgeResults.has(impact.result)) issues.push(`knowledgeImpact.result is invalid: ${impact.result}`);
      if (!Array.isArray(impact.triggeredBy)) issues.push("knowledgeImpact.triggeredBy must be an array");
      if (!Array.isArray(impact.reviewResults)) issues.push("knowledgeImpact.reviewResults must be an array");
      if (typeof impact.coverageGap !== "boolean") issues.push("knowledgeImpact.coverageGap must be boolean");
      if (!Number.isInteger(impact.validationExitCode)) issues.push("knowledgeImpact.validationExitCode must be an integer");
      for (const [index, review] of (impact.reviewResults || []).entries()) {
        if (!review || typeof review !== "object" || Array.isArray(review)) {
          issues.push(`knowledgeImpact.reviewResults[${index}] must be an object`);
          continue;
        }
        if (typeof review.document !== "string" || review.document.length === 0) issues.push(`knowledgeImpact.reviewResults[${index}].document is required`);
        if (!allowedDocumentVerdicts.has(review.verdict)) issues.push(`knowledgeImpact.reviewResults[${index}].verdict is invalid`);
        if (!Array.isArray(review.evidence) || review.evidence.length === 0) issues.push(`knowledgeImpact.reviewResults[${index}].evidence must be a non-empty array`);
      }
      if (impact.result === "none" && impact.coverageGap) issues.push("knowledgeImpact cannot be none when coverageGap is true");
      if (evidence.review?.verdict === "通过") {
        const blocking = new Set(["stale_detected", "conflict_detected"]);
        if (blocking.has(impact.result)) issues.push("passing review cannot contain stale_detected or conflict_detected knowledgeImpact");
        if ((impact.reviewResults || []).some((review) => ["STALE", "CONFLICT"].includes(review.verdict))) {
          issues.push("passing review cannot contain STALE or CONFLICT knowledge document verdicts");
        }
      }
    }
  }
  return { valid: issues.length === 0, issues };
}

function parseArgs(argv) {
  const args = { file: null, allowPendingReview: false, allowMissingConfirmation: false, requireKnowledgeImpact: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--file") args.file = argv[++index];
    else if (argv[index] === "--allow-pending-review") args.allowPendingReview = true;
    else if (argv[index] === "--allow-missing-confirmation") args.allowMissingConfirmation = true;
    else if (argv[index] === "--require-knowledge-impact") args.requireKnowledgeImpact = true;
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
    const result = validateEvidence(evidence, args);
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
