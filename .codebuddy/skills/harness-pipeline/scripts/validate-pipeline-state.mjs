#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { validateEvidence } from "../../spec-harness/scripts/validate-evidence.mjs";
import { hashJson, isObject, validateContract } from "../../spec-harness/scripts/contract-utils.mjs";
import { validateTraceability } from "../../spec-harness/scripts/validate-traceability.mjs";
import { validateParityManifest } from "../../spec-harness/scripts/validate-parity.mjs";

const ALLOWED_STATUS = new Set(["pending", "in_progress", "blocked", "completed", "completed_with_exceptions"]);
const ALLOWED_PHASE = new Set([
  "pending", "discovery", "plan_review", "ready", "implement", "review", "fix",
  "amendment_pending", "amendment_review", "replan", "revalidate", "finalize", "final_review", "completed", "blocked",
]);
const COMPLETION_LEVELS = Object.freeze({
  feature_delivery: ["planned", "code_complete", "integration_verified", "behavior_verified", "completed"],
  behavior_preserving_migration: ["discovered", "baseline_locked", "implemented", "parity_verified", "cutover_ready", "cutover_observed", "retirement_ready", "retired"],
});

function readJson(file, issues, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { issues.push(`${label} is not valid JSON: ${error.message}`); return null; }
}

function hasApproval(exception) {
  return Boolean(exception && exception.approved_object && exception.approved_by && exception.approved_at && exception.reason);
}

function asList(value) {
  if (Array.isArray(value)) return value;
  if (value === undefined || value === null) return [];
  return [value];
}

function uniqueList(values, label, issues) {
  const seen = new Set();
  for (const value of values || []) {
    if (typeof value !== "string" || value.length === 0) { issues.push(`${label} entries must be non-empty strings`); continue; }
    if (seen.has(value)) issues.push(`${label} contains duplicate task: ${value}`);
    seen.add(value);
  }
}

function exceptionTaskIds(exception) {
  return asList(exception?.task_ids).concat(asList(exception?.task_id)).filter(Boolean);
}

function completionLevelSatisfied(profile, actual, required) {
  const levels = COMPLETION_LEVELS[profile] || [];
  return levels.includes(actual) && levels.includes(required) && levels.indexOf(actual) >= levels.indexOf(required);
}

function requiresIndependentReview(task) {
  return task.reviewPolicy === "independent_required" || (task.destructiveActions || []).length > 0;
}

function validateIndependentReview(taskId, evidence, issues) {
  const review = evidence.review || {};
  if (review.reviewer_type !== "independent_agent") issues.push(`${taskId} requires reviewer_type independent_agent`);
  if (!review.implementer_run_id || !review.reviewer_run_id) issues.push(`${taskId} independent review requires implementer_run_id and reviewer_run_id`);
  if (review.implementer_run_id && review.implementer_run_id === review.reviewer_run_id) issues.push(`${taskId} reviewer_run_id must differ from implementer_run_id`);
}

function carriedForwardAcrossRevisions(featureDir, taskId, fromRevision, toRevision) {
  if (!Number.isInteger(fromRevision) || !Number.isInteger(toRevision) || fromRevision >= toRevision) return false;
  for (let revision = fromRevision + 1; revision <= toRevision; revision += 1) {
    const file = path.join(featureDir, "revisions", `REV-${String(revision).padStart(4, "0")}.json`);
    if (!fs.existsSync(file)) return false;
    try {
      const manifest = JSON.parse(fs.readFileSync(file, "utf8"));
      if (!(manifest.carried_forward_tasks || []).includes(taskId)) return false;
    } catch {
      return false;
    }
  }
  return true;
}

export function validatePipelineState(featureDir, options = {}) {
  const absoluteDir = path.resolve(featureDir);
  const featureName = path.basename(absoluteDir);
  const issues = [];
  const warnings = [];
  const statePath = options.statePath || path.join(absoluteDir, "pipeline-state.json");
  const scopePath = path.join(absoluteDir, "task-scope.json");
  const state = readJson(statePath, issues, "pipeline-state.json");
  const scope = readJson(scopePath, issues, "task-scope.json");
  if (!state || !scope) return { valid: false, feature_name: featureName, issues, warnings };

  const schemaVersion = scope.schemaVersion === 2 ? 2 : 1;
  let contract = null;
  let traceability = null;
  let profile = "feature_delivery";
  let parityResult = null;
  if (state.schemaVersion !== schemaVersion) issues.push(`pipeline-state.json schemaVersion must be ${schemaVersion}`);
  if (state.feature_name !== featureName) issues.push("pipeline-state.json feature_name does not match directory");
  if (!ALLOWED_STATUS.has(state.status)) issues.push(`invalid pipeline status: ${state.status}`);
  if (!ALLOWED_PHASE.has(state.current_phase)) issues.push(`invalid current_phase: ${state.current_phase}`);
  if (!Array.isArray(state.completed_tasks)) issues.push("completed_tasks must be an array");
  if (!Array.isArray(state.failed_tasks)) issues.push("failed_tasks must be an array");
  if (!Array.isArray(state.blocked_tasks)) issues.push("blocked_tasks must be an array");
  if (!isObject(state.task_runs)) issues.push("task_runs must be an object");

  if (schemaVersion === 2) {
    contract = readJson(path.join(absoluteDir, "contract.json"), issues, "contract.json");
    traceability = readJson(path.join(absoluteDir, "traceability.json"), issues, "traceability.json");
    if (contract) {
      const result = validateContract(contract, featureName, { requireApproved: new Set(["completed", "completed_with_exceptions"]).has(state.status) });
      for (const issue of result.issues) issues.push(`contract.json: ${issue}`);
      profile = contract.profile;
      if (state.contract_revision !== contract.contractRevision) issues.push("pipeline state contract_revision does not match contract.json");
    }
    if (!Array.isArray(state.open_change_requests)) issues.push("open_change_requests must be an array");
    if (!Array.isArray(state.needs_revalidation_tasks)) issues.push("needs_revalidation_tasks must be an array");
    if (state.current_phase === "amendment_pending" && (state.open_change_requests || []).length === 0) issues.push("amendment_pending requires an open change request");
    if (profile === "behavior_preserving_migration") {
      const manifest = readJson(path.join(absoluteDir, "baseline", "behavior-manifest.json"), issues, "behavior-manifest.json");
      if (manifest) {
        parityResult = validateParityManifest(manifest, { requireVerified: new Set(["completed", "completed_with_exceptions"]).has(state.status) });
        for (const issue of parityResult.issues) issues.push(`behavior-manifest.json: ${issue}`);
        if (contract?.baseline?.sha !== manifest.baseline?.sha) issues.push("behavior manifest baseline.sha does not match contract.json");
        if (contract?.baseline?.tree !== manifest.baseline?.tree) issues.push("behavior manifest baseline.tree does not match contract.json");
        if (manifest.contractRevision !== contract?.contractRevision) issues.push("behavior manifest contractRevision does not match contract.json");
      }
    }
  }

  const tasks = isObject(scope.tasks) ? scope.tasks : {};
  const applicableTaskIds = Object.entries(tasks).filter(([, task]) => task?.lifecycle !== "superseded").map(([taskId]) => taskId);
  const completed = new Set(state.completed_tasks || []);
  const failed = state.failed_tasks || [];
  const blocked = state.blocked_tasks || [];
  const skippedHumanConfirmation = Array.isArray(state.skipped_human_confirmation_tasks) ? state.skipped_human_confirmation_tasks : [];
  const exceptions = Array.isArray(state.approved_exceptions) ? state.approved_exceptions : [];
  const needsRevalidation = new Set(state.needs_revalidation_tasks || []);
  const evidenceResults = {};
  const knownTasks = new Set(Object.keys(tasks));
  const exceptionCoveredTasks = new Set();

  uniqueList(state.completed_tasks, "completed_tasks", issues);
  uniqueList(failed, "failed_tasks", issues);
  uniqueList(blocked, "blocked_tasks", issues);
  uniqueList(skippedHumanConfirmation, "skipped_human_confirmation_tasks", issues);
  if (schemaVersion === 2) {
    uniqueList(state.open_change_requests, "open_change_requests", issues);
    uniqueList(state.needs_revalidation_tasks, "needs_revalidation_tasks", issues);
  }

  for (const [label, values] of [["completed_tasks", state.completed_tasks || []], ["failed_tasks", failed], ["blocked_tasks", blocked], ["skipped_human_confirmation_tasks", skippedHumanConfirmation]]) {
    for (const taskId of values) if (!knownTasks.has(taskId)) issues.push(`${label} contains unknown task: ${taskId}`);
  }
  for (const taskId of completed) {
    if (failed.includes(taskId)) issues.push(`task cannot be both completed and failed: ${taskId}`);
    if (blocked.includes(taskId)) issues.push(`task cannot be both completed and blocked: ${taskId}`);
    if (skippedHumanConfirmation.includes(taskId)) issues.push(`task cannot be both completed and skipped for human confirmation: ${taskId}`);
  }

  for (const [index, exception] of exceptions.entries()) {
    if (!hasApproval(exception)) issues.push(`approved_exceptions[${index}] must include approved_object, approved_by, approved_at, and reason`);
    const covered = exceptionTaskIds(exception);
    if (covered.length === 0) issues.push(`approved_exceptions[${index}] must include task_id or task_ids`);
    for (const taskId of covered) {
      if (!knownTasks.has(taskId)) issues.push(`approved_exceptions[${index}] references unknown task: ${taskId}`);
      exceptionCoveredTasks.add(taskId);
    }
    if (profile === "behavior_preserving_migration" && ["behavioral_parity", "baseline", "delete_source"].includes(exception.category)) {
      issues.push(`approved_exceptions[${index}] cannot waive mandatory migration parity`);
    }
  }

  for (const taskId of completed) {
    const task = tasks[taskId];
    const run = state.task_runs?.[taskId];
    if (!task) { issues.push(`completed task is not defined in task-scope.json: ${taskId}`); continue; }
    if (!run) { issues.push(`completed task has no task_runs entry: ${taskId}`); continue; }
    if (!run.evidence) { issues.push(`completed task has no evidence path: ${taskId}`); continue; }
    const evidencePath = path.resolve(absoluteDir, "..", "..", run.evidence);
    if (!fs.existsSync(evidencePath)) { issues.push(`completed task evidence file is missing: ${taskId} -> ${run.evidence}`); continue; }
    const evidence = readJson(evidencePath, issues, `${taskId} evidence`);
    if (!evidence) continue;
    const evidenceResult = validateEvidence(evidence, {
      allowMissingConfirmation: task.requiresHumanConfirmation !== true,
      requireKnowledgeImpact: task.requiredKnowledgeImpact === true,
      requiredChecks: schemaVersion === 2 ? task.requiredChecks : undefined,
    });
    evidenceResults[taskId] = evidenceResult;
    for (const issue of evidenceResult.issues) issues.push(`${taskId} evidence: ${issue}`);
    if (evidence.feature_name !== featureName) issues.push(`${taskId} evidence feature_name does not match directory`);
    if (evidence.task_id !== taskId) issues.push(`${taskId} evidence task_id does not match completed task`);
    if (run.base_sha && evidence.base_sha && run.base_sha !== evidence.base_sha) issues.push(`${taskId} run base_sha does not match evidence`);
    if (run.head_sha && evidence.head_sha && run.head_sha !== evidence.head_sha) issues.push(`${taskId} run head_sha does not match evidence`);
    if (evidence.scope?.status !== "passed") issues.push(`${taskId} evidence scope.status must be passed`);
    if (run.scope_status !== "passed") issues.push(`${taskId} run scope_status must be passed`);
    if (run.checks_status !== "passed") issues.push(`${taskId} run checks_status must be passed`);
    if (run.review_verdict !== "通过") issues.push(`${taskId} run review_verdict must be 通过`);
    if (task.requiresHumanConfirmation && !run.human_confirmation?.confirmed) issues.push(`${taskId} requires human confirmation but run has no confirmation`);
    if (task.requiresHumanConfirmation && !run.human_confirmation?.confirmed_at) issues.push(`${taskId} requires human confirmation but run has no confirmed_at`);

    if (schemaVersion === 2) {
      const expectedTaskHash = hashJson(task);
      const expectedTraceabilityHash = hashJson(traceability);
      const carriedForward = carriedForwardAcrossRevisions(absoluteDir, taskId, evidence.contract_revision, contract?.contractRevision);
      if (!needsRevalidation.has(taskId)) {
        if ((run.contract_revision !== contract?.contractRevision || evidence.contract_revision !== contract?.contractRevision) && !carriedForward) issues.push(`${taskId} contract revision does not match active contract and is not explicitly carried forward`);
        if (run.task_definition_hash !== expectedTaskHash || evidence.task_definition_hash !== expectedTaskHash) issues.push(`${taskId} task_definition_hash does not match active TASK definition`);
        if ((run.traceability_hash !== expectedTraceabilityHash || evidence.traceability_hash !== expectedTraceabilityHash) && !carriedForward) issues.push(`${taskId} traceability_hash does not match active traceability and is not explicitly carried forward`);
      }
      if (run.review_outcome !== "pass" || evidence.review?.outcome !== "pass") issues.push(`${taskId} review outcome must be pass`);
      if (requiresIndependentReview(task)) validateIndependentReview(taskId, evidence, issues);
      if ((task.destructiveActions || []).length > 0 && !run.human_confirmation?.confirmed) issues.push(`${taskId} destructive action has no human confirmation`);
    }
  }

  const completionCandidate = new Set(["completed", "completed_with_exceptions"]).has(state.status);
  if (completionCandidate && schemaVersion === 2) {
    if ((state.open_change_requests || []).length > 0) issues.push("completed state cannot have open change requests");
    if ((state.needs_revalidation_tasks || []).length > 0) issues.push("completed state cannot have tasks needing revalidation");
    const traceResult = validateTraceability(traceability, { requireVerified: true });
    for (const issue of traceResult.issues) issues.push(`traceability completion: ${issue}`);
    if (!completionLevelSatisfied(profile, state.completion_level, contract?.requiredCompletionLevel)) {
      issues.push(`completion_level ${state.completion_level} does not satisfy required ${contract?.requiredCompletionLevel}`);
    }
    if (profile === "behavior_preserving_migration" && !parityResult?.valid) issues.push("migration parity must be fully verified before completion");
  }

  if (state.status === "completed") {
    if (failed.length > 0) issues.push("completed status cannot have failed_tasks");
    if (blocked.length > 0) issues.push("completed status cannot have blocked_tasks");
    if (exceptions.length > 0) issues.push("completed status cannot have approved_exceptions; use completed_with_exceptions");
    for (const taskId of applicableTaskIds) if (!completed.has(taskId)) issues.push(`completed status is missing completed task: ${taskId}`);
  }
  if (state.status === "completed_with_exceptions") {
    if (exceptions.length === 0) issues.push("completed_with_exceptions requires approved_exceptions");
    for (const taskId of applicableTaskIds) if (!completed.has(taskId) && !exceptionCoveredTasks.has(taskId)) issues.push(`completed_with_exceptions has unfinished task without approved exception: ${taskId}`);
    for (const taskId of [...failed, ...blocked, ...skippedHumanConfirmation]) if (!exceptionCoveredTasks.has(taskId)) issues.push(`completed_with_exceptions has ${taskId} in failed/blocked/skipped lists without approved exception`);
  }
  if (state.skip_human_confirm === true) {
    if (state.status === "completed" && skippedHumanConfirmation.length > 0) issues.push("skip_human_confirm=true with skipped tasks cannot produce completed status");
    if (state.status === "completed_with_exceptions") for (const taskId of skippedHumanConfirmation) if (!exceptionCoveredTasks.has(taskId)) issues.push(`skip_human_confirm skipped task requires approved exception: ${taskId}`);
  }

  return { valid: issues.length === 0, feature_name: featureName, schema_version: schemaVersion, profile, status: state.status, current_phase: state.current_phase, completion_level: state.completion_level, completed_tasks: state.completed_tasks || [], failed_tasks: failed, blocked_tasks: blocked, evidence_results: evidenceResults, issues, warnings };
}

function parseArgs(argv) {
  const args = { featureDir: null, statePath: null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--feature-dir") args.featureDir = argv[++index];
    else if (argv[index] === "--state") args.statePath = argv[++index];
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!args.featureDir) throw new Error("--feature-dir is required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = validatePipelineState(args.featureDir, { statePath: args.statePath });
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      console.log(`feature: ${result.feature_name}`);
      console.log(`schema_version: ${result.schema_version}`);
      console.log(`profile: ${result.profile}`);
      console.log(`status: ${result.status || "unknown"}`);
      console.log(`current_phase: ${result.current_phase || "unknown"}`);
      if (result.completion_level) console.log(`completion_level: ${result.completion_level}`);
      for (const warning of result.warnings) console.log(`warning: ${warning}`);
      for (const issue of result.issues) console.error(`error: ${issue}`);
      console.log(`pipeline_state_result: ${result.valid ? "PASS" : "FAIL"}`);
    }
    process.exit(result.valid ? 0 : 1);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
