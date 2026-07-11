#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { validateEvidence } from "../../spec-harness/scripts/validate-evidence.mjs";

const ALLOWED_STATUS = new Set(["pending", "in_progress", "blocked", "completed", "completed_with_exceptions"]);
const ALLOWED_PHASE = new Set(["pending", "implement", "review", "fix", "finalize", "completed", "blocked"]);

function readJson(file, issues, label) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    issues.push(`${label} is not valid JSON: ${error.message}`);
    return null;
  }
}

function hasApproval(exception) {
  return Boolean(
    exception &&
      exception.approved_object &&
      exception.approved_by &&
      exception.approved_at &&
      exception.reason,
  );
}

function asList(value) {
  if (Array.isArray(value)) return value;
  if (value === undefined || value === null) return [];
  return [value];
}

function uniqueList(values, label, issues) {
  const seen = new Set();
  for (const value of values || []) {
    if (typeof value !== "string" || value.length === 0) {
      issues.push(`${label} entries must be non-empty strings`);
      continue;
    }
    if (seen.has(value)) issues.push(`${label} contains duplicate task: ${value}`);
    seen.add(value);
  }
}

function exceptionTaskIds(exception) {
  return asList(exception?.task_ids).concat(asList(exception?.task_id)).filter(Boolean);
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

  if (state.schemaVersion !== 1) issues.push("pipeline-state.json schemaVersion must be 1");
  if (state.feature_name !== featureName) issues.push("pipeline-state.json feature_name does not match directory");
  if (!ALLOWED_STATUS.has(state.status)) issues.push(`invalid pipeline status: ${state.status}`);
  if (!ALLOWED_PHASE.has(state.current_phase)) issues.push(`invalid current_phase: ${state.current_phase}`);
  if (!Array.isArray(state.completed_tasks)) issues.push("completed_tasks must be an array");
  if (!Array.isArray(state.failed_tasks)) issues.push("failed_tasks must be an array");
  if (!Array.isArray(state.blocked_tasks)) issues.push("blocked_tasks must be an array");
  if (!state.task_runs || typeof state.task_runs !== "object" || Array.isArray(state.task_runs)) {
    issues.push("task_runs must be an object");
  }

  const tasks = scope.tasks && typeof scope.tasks === "object" && !Array.isArray(scope.tasks) ? scope.tasks : {};
  const completed = new Set(state.completed_tasks || []);
  const failed = state.failed_tasks || [];
  const blocked = state.blocked_tasks || [];
  const skippedHumanConfirmation = Array.isArray(state.skipped_human_confirmation_tasks) ? state.skipped_human_confirmation_tasks : [];
  const exceptions = Array.isArray(state.approved_exceptions) ? state.approved_exceptions : [];
  const evidenceResults = {};
  const taskIds = Object.keys(tasks);
  const knownTasks = new Set(taskIds);
  const exceptionCoveredTasks = new Set();

  uniqueList(state.completed_tasks, "completed_tasks", issues);
  uniqueList(failed, "failed_tasks", issues);
  uniqueList(blocked, "blocked_tasks", issues);
  uniqueList(skippedHumanConfirmation, "skipped_human_confirmation_tasks", issues);

  for (const [label, values] of [
    ["completed_tasks", state.completed_tasks || []],
    ["failed_tasks", failed],
    ["blocked_tasks", blocked],
    ["skipped_human_confirmation_tasks", skippedHumanConfirmation],
  ]) {
    for (const taskId of values) {
      if (!knownTasks.has(taskId)) issues.push(`${label} contains unknown task: ${taskId}`);
    }
  }

  for (const taskId of completed) {
    if (failed.includes(taskId)) issues.push(`task cannot be both completed and failed: ${taskId}`);
    if (blocked.includes(taskId)) issues.push(`task cannot be both completed and blocked: ${taskId}`);
    if (skippedHumanConfirmation.includes(taskId)) issues.push(`task cannot be both completed and skipped for human confirmation: ${taskId}`);
  }

  for (const [index, exception] of exceptions.entries()) {
    if (!hasApproval(exception)) {
      issues.push(`approved_exceptions[${index}] must include approved_object, approved_by, approved_at, and reason`);
    }
    const covered = exceptionTaskIds(exception);
    if (covered.length === 0) {
      issues.push(`approved_exceptions[${index}] must include task_id or task_ids`);
    }
    for (const taskId of covered) {
      if (!knownTasks.has(taskId)) issues.push(`approved_exceptions[${index}] references unknown task: ${taskId}`);
      exceptionCoveredTasks.add(taskId);
    }
  }

  for (const taskId of completed) {
    const task = tasks[taskId];
    const run = state.task_runs?.[taskId];
    if (!task) {
      issues.push(`completed task is not defined in task-scope.json: ${taskId}`);
      continue;
    }
    if (!run) {
      issues.push(`completed task has no task_runs entry: ${taskId}`);
      continue;
    }
    if (!run.evidence) {
      issues.push(`completed task has no evidence path: ${taskId}`);
      continue;
    }
    const evidencePath = path.resolve(absoluteDir, "..", "..", run.evidence);
    if (!fs.existsSync(evidencePath)) {
      issues.push(`completed task evidence file is missing: ${taskId} -> ${run.evidence}`);
      continue;
    }
    const evidence = readJson(evidencePath, issues, `${taskId} evidence`);
    if (!evidence) continue;
    const evidenceResult = validateEvidence(evidence, {
      allowMissingConfirmation: task.requiresHumanConfirmation !== true,
      requireKnowledgeImpact: task.requiredKnowledgeImpact === true,
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
    if (task.requiresHumanConfirmation && !run.human_confirmation?.confirmed) {
      issues.push(`${taskId} requires human confirmation but run has no confirmation`);
    }
    if (task.requiresHumanConfirmation && !run.human_confirmation?.confirmed_at) {
      issues.push(`${taskId} requires human confirmation but run has no confirmed_at`);
    }
  }

  if (state.status === "completed") {
    if (failed.length > 0) issues.push("completed status cannot have failed_tasks");
    if (blocked.length > 0) issues.push("completed status cannot have blocked_tasks");
    if (exceptions.length > 0) issues.push("completed status cannot have approved_exceptions; use completed_with_exceptions");
    for (const taskId of taskIds) {
      if (!completed.has(taskId)) issues.push(`completed status is missing completed task: ${taskId}`);
    }
  }

  if (state.status === "completed_with_exceptions") {
    if (exceptions.length === 0) issues.push("completed_with_exceptions requires approved_exceptions");
    for (const taskId of taskIds) {
      if (!completed.has(taskId) && !exceptionCoveredTasks.has(taskId)) {
        issues.push(`completed_with_exceptions has unfinished task without approved exception: ${taskId}`);
      }
    }
    for (const taskId of [...failed, ...blocked, ...skippedHumanConfirmation]) {
      if (!exceptionCoveredTasks.has(taskId)) {
        issues.push(`completed_with_exceptions has ${taskId} in failed/blocked/skipped lists without approved exception`);
      }
    }
  }

  if (state.skip_human_confirm === true) {
    if (state.status === "completed" && skippedHumanConfirmation.length > 0) {
      issues.push("skip_human_confirm=true with skipped tasks cannot produce completed status");
    }
    if (state.status === "completed_with_exceptions") {
      for (const taskId of skippedHumanConfirmation) {
        if (!exceptionCoveredTasks.has(taskId)) {
          issues.push(`skip_human_confirm skipped task requires approved exception: ${taskId}`);
        }
      }
    }
  }

  return {
    valid: issues.length === 0,
    feature_name: featureName,
    status: state.status,
    current_phase: state.current_phase,
    completed_tasks: state.completed_tasks || [],
    failed_tasks: failed,
    blocked_tasks: blocked,
    evidence_results: evidenceResults,
    issues,
    warnings,
  };
}

function parseArgs(argv) {
  const args = { featureDir: null, statePath: null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const item = argv[index];
    if (item === "--feature-dir") args.featureDir = argv[++index];
    else if (item === "--state") args.statePath = argv[++index];
    else if (item === "--json") args.json = true;
    else throw new Error(`unknown argument: ${item}`);
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
      console.log(`status: ${result.status || "unknown"}`);
      console.log(`current_phase: ${result.current_phase || "unknown"}`);
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
