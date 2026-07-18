#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { hashJson, isObject, validateContract, validateStringArray } from "./contract-utils.mjs";
import { validateTraceability } from "./validate-traceability.mjs";
import { validateParityManifest } from "./validate-parity.mjs";
import { validateAmendment } from "./validate-amendment.mjs";

export const CLASSIFICATIONS = Object.freeze({
  CURRENT: "current",
  LEGACY_COMPATIBLE: "legacy-compatible",
  UNSUPPORTED: "unsupported",
});

const TASK_LIFECYCLES = new Set(["draft", "ready", "in_progress", "review", "completed", "amendment_pending", "needs_revalidation", "superseded", "blocked"]);
const CHECK_KINDS = new Set(["static", "build", "unit", "integration", "contract", "differential", "e2e", "manual"]);
const DESTRUCTIVE_ACTIONS = new Set(["delete_files", "bulk_delete", "delete_source", "cutover", "schema_change", "public_api_change", "security_change"]);
const REVIEW_POLICIES = new Set(["self_allowed", "independent_required"]);

function readJson(file, issues, label) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    issues.push(`${label} is not valid JSON: ${error.message}`);
    return null;
  }
}

function isRepositoryRootDocsPattern(pattern) {
  if (typeof pattern !== "string") return false;
  const normalized = pattern.replace(/^\.\//, "").replace(/\/+$/, "");
  return normalized === "docs" || normalized.startsWith("docs/");
}

function isBroadContractScope(pattern, featureName) {
  if (typeof pattern !== "string") return false;
  const normalized = pattern.replace(/^\.\//, "").replace(/\/+$/, "");
  return normalized === `.spec/${featureName}` || normalized === `.spec/${featureName}/**`;
}

function isContractOwnedPattern(pattern, featureName) {
  if (typeof pattern !== "string") return false;
  const normalized = pattern.replace(/^\.\//, "").replace(/\/+$/, "");
  const prefix = `.spec/${featureName}/`;
  if (!normalized.startsWith(prefix)) return false;
  const relative = normalized.slice(prefix.length);
  const exact = new Set(["contract.json", "traceability.json", "task-scope.json", "TASKS.md", "AGENT_RULES.md", `${featureName}-SPEC.md`, `${featureName}-SDD.md`, "pipeline-state.json"]);
  return exact.has(relative) || ["acceptance/", "baseline/", "changes/", "revisions/", "reviews/", "prompts/"].some((ownedPrefix) => relative.startsWith(ownedPrefix));
}

export function compileGlob(pattern) {
  if (typeof pattern !== "string" || pattern.length === 0) throw new Error("glob pattern must be a non-empty string");
  if (pattern.includes("[") || pattern.includes("]")) throw new Error(`unsupported character class syntax in glob: ${pattern}`);
  let source = "^";
  for (let index = 0; index < pattern.length; index += 1) {
    const character = pattern[index];
    if (character === "*" && pattern[index + 1] === "*") {
      source += ".*";
      index += 1;
    } else if (character === "*") source += "[^/]*";
    else if (character === "?") source += "[^/]";
    else source += character.replace(/[|\\{}()[\]^$+?.]/g, "\\$&");
  }
  return new RegExp(`${source}$`);
}

function validateRequiredCheck(check, label, issues) {
  if (!isObject(check)) {
    issues.push(`${label} must be an object`);
    return;
  }
  if (typeof check.id !== "string" || check.id.length === 0) issues.push(`${label}.id is required`);
  if (!CHECK_KINDS.has(check.kind)) issues.push(`${label}.kind is invalid`);
  if (typeof check.blocking !== "boolean") issues.push(`${label}.blocking must be boolean`);
  validateStringArray(check.covers, `${label}.covers`, issues, { nonEmpty: true });
  if (typeof check.successCriteria !== "string" || check.successCriteria.length === 0) issues.push(`${label}.successCriteria is required`);
  if (check.kind !== "manual" && (typeof check.command !== "string" || check.command.length === 0)) issues.push(`${label}.command is required for executable checks`);
}

export function validateFeature(featureDir, options = {}) {
  const absoluteDir = path.resolve(featureDir);
  const featureName = path.basename(absoluteDir);
  const issues = [];
  const warnings = [];
  const taskIds = [];
  let schemaVersion = null;
  let profile = null;
  let contractRevision = null;
  const appliedAmendmentRevisions = new Set();

  if (!fs.existsSync(absoluteDir) || !fs.statSync(absoluteDir).isDirectory()) {
    return { feature_name: featureName, feature_dir: absoluteDir, classification: CLASSIFICATIONS.UNSUPPORTED, issues: [`feature directory does not exist: ${absoluteDir}`], warnings, task_ids: [] };
  }

  const commonFiles = [
    `${featureName}-SPEC.md`, `${featureName}-SDD.md`, "TASKS.md", "AGENT_RULES.md", "task-scope.json",
    "prompts/implement-task.md", "prompts/self-review.md", "prompts/fix-check-failures.md",
    "scripts/check-task-scope.sh", "scripts/agent-check.sh",
  ];
  const commonDirectories = ["acceptance", "prompts", "scripts", "reports"];
  for (const relative of commonFiles) if (!fs.existsSync(path.join(absoluteDir, relative))) issues.push(`missing required file: ${relative}`);
  for (const relative of commonDirectories) {
    const candidate = path.join(absoluteDir, relative);
    if (!fs.existsSync(candidate) || !fs.statSync(candidate).isDirectory()) issues.push(`missing required directory: ${relative}/`);
  }

  const scopePath = path.join(absoluteDir, "task-scope.json");
  const scope = fs.existsSync(scopePath) ? readJson(scopePath, issues, "task-scope.json") : null;
  let classification = CLASSIFICATIONS.CURRENT;
  let contract = null;
  let traceability = null;

  if (scope) {
    schemaVersion = scope.schemaVersion;
    if (new Set([1, 2]).has(schemaVersion) && scope.feature_name === featureName && isObject(scope.tasks)) {
      classification = CLASSIFICATIONS.CURRENT;
      if (schemaVersion === 1) warnings.push("schemaVersion 1 is supported for compatibility; new features and behavior-preserving migrations must use schemaVersion 2");
    } else if (schemaVersion === undefined && scope.feature_name === featureName && isObject(scope.tasks)) {
      classification = CLASSIFICATIONS.LEGACY_COMPATIBLE;
      warnings.push("task-scope.json has no schemaVersion; treating feature as legacy-compatible");
    } else {
      classification = CLASSIFICATIONS.UNSUPPORTED;
      issues.push("unsupported task-scope.json shape; expected schemaVersion 1 or 2 with feature_name + tasks");
    }
  }

  if (schemaVersion === 2) {
    for (const relative of ["contract.json", "traceability.json", "prompts/propose-amendment.md", "prompts/review-amendment.md", "prompts/reconcile-plan.md"]) {
      if (!fs.existsSync(path.join(absoluteDir, relative))) issues.push(`missing schemaVersion 2 file: ${relative}`);
    }
    for (const relative of ["changes", "revisions", "reviews"]) {
      const candidate = path.join(absoluteDir, relative);
      if (!fs.existsSync(candidate) || !fs.statSync(candidate).isDirectory()) issues.push(`missing schemaVersion 2 directory: ${relative}/`);
    }
    const contractPath = path.join(absoluteDir, "contract.json");
    const traceabilityPath = path.join(absoluteDir, "traceability.json");
    contract = fs.existsSync(contractPath) ? readJson(contractPath, issues, "contract.json") : null;
    traceability = fs.existsSync(traceabilityPath) ? readJson(traceabilityPath, issues, "traceability.json") : null;
    if (contract) {
      const result = validateContract(contract, featureName);
      for (const issue of result.issues) issues.push(`contract.json: ${issue}`);
      profile = contract.profile;
      contractRevision = contract.contractRevision;
      if (scope.contractRevision !== contractRevision) issues.push("task-scope.json contractRevision does not match contract.json");
    }
    if (traceability) {
      const result = validateTraceability(traceability);
      for (const issue of result.issues) issues.push(`traceability.json: ${issue}`);
      if (traceability.feature_name !== featureName) issues.push("traceability.json feature_name does not match directory");
      if (contract && traceability.contractRevision !== contractRevision) issues.push("traceability.json contractRevision does not match contract.json");
    }
    if (profile === "behavior_preserving_migration") {
      const manifestPath = path.join(absoluteDir, "baseline", "behavior-manifest.json");
      if (!fs.existsSync(manifestPath)) issues.push("behavior_preserving_migration requires baseline/behavior-manifest.json");
      else {
        const manifest = readJson(manifestPath, issues, "behavior-manifest.json");
        if (manifest) {
          const result = validateParityManifest(manifest);
          for (const issue of result.issues) issues.push(`behavior-manifest.json: ${issue}`);
          if (manifest.feature_name !== featureName) issues.push("behavior-manifest.json feature_name does not match directory");
          if (manifest.contractRevision !== contractRevision) issues.push("behavior-manifest.json contractRevision does not match contract.json");
          if (contract?.baseline?.sha !== manifest.baseline?.sha) issues.push("behavior manifest baseline.sha does not match contract.json");
          if (contract?.baseline?.tree !== manifest.baseline?.tree) issues.push("behavior manifest baseline.tree does not match contract.json");
        }
      }
    }
    const changesDir = path.join(absoluteDir, "changes");
    if (fs.existsSync(changesDir)) {
      for (const name of fs.readdirSync(changesDir).filter((item) => /^CR-[0-9]+\.json$/.test(item)).sort()) {
        const amendment = readJson(path.join(changesDir, name), issues, `changes/${name}`);
        if (!amendment) continue;
        const result = validateAmendment(amendment);
        for (const issue of result.issues) issues.push(`changes/${name}: ${issue}`);
        if (amendment.feature_name !== featureName) issues.push(`changes/${name} feature_name does not match directory`);
        if (amendment.baseRevision > contractRevision) issues.push(`changes/${name} baseRevision exceeds active contract revision`);
        if (amendment.status === "applied") appliedAmendmentRevisions.add(amendment.resultingRevision);
      }
    }
    if (contract?.planningStatus === "approved") {
      const planReviewPath = path.join(absoluteDir, "reviews", `PLAN-REV-${String(contractRevision).padStart(4, "0")}.json`);
      if (fs.existsSync(planReviewPath)) {
        const review = readJson(planReviewPath, issues, "plan review");
        if (review) {
          if (review.schemaVersion !== 2 || review.feature_name !== featureName || review.contractRevision !== contractRevision) issues.push("plan review identity does not match active contract");
          if (review.contract_hash !== hashJson(contract)) issues.push("plan review contract_hash does not match active contract");
          if (review.outcome !== "pass" || review.verdict !== "通过") issues.push("approved contract requires a passing plan review");
          if (contract.reviewPolicy?.plan === "independent_required") {
            if (review.reviewer_type !== "independent_agent") issues.push("plan review must be independent");
            if (!review.planner_run_id || !review.reviewer_run_id || review.planner_run_id === review.reviewer_run_id) issues.push("independent plan review requires distinct planner and reviewer run IDs");
          }
        }
      } else if (!appliedAmendmentRevisions.has(contractRevision)) {
        issues.push(`missing plan review for approved contract revision ${contractRevision}`);
      }
    }
    const revisionsDir = path.join(absoluteDir, "revisions");
    if (fs.existsSync(revisionsDir)) {
      for (const name of fs.readdirSync(revisionsDir).filter((item) => /^REV-[0-9]+\.json$/.test(item)).sort()) {
        const revision = readJson(path.join(revisionsDir, name), issues, `revisions/${name}`);
        if (!revision) continue;
        const expectedRevision = Number(name.match(/[0-9]+/)[0]);
        if (revision.schemaVersion !== 2) issues.push(`revisions/${name} schemaVersion must be 2`);
        if (revision.feature_name !== featureName) issues.push(`revisions/${name} feature_name does not match directory`);
        if (revision.revision !== expectedRevision) issues.push(`revisions/${name} revision does not match filename`);
        validateStringArray(revision.revalidation_tasks, `revisions/${name} revalidation_tasks`, issues);
        validateStringArray(revision.carried_forward_tasks, `revisions/${name} carried_forward_tasks`, issues);
      }
      if (contractRevision > 1 && !fs.existsSync(path.join(revisionsDir, `REV-${String(contractRevision).padStart(4, "0")}.json`))) {
        issues.push(`missing active revision manifest for contractRevision ${contractRevision}`);
      }
    }
  }

  if (scope?.tasks && isObject(scope.tasks)) {
    const tasksMarkdownPath = path.join(absoluteDir, "TASKS.md");
    const tasksMarkdown = fs.existsSync(tasksMarkdownPath) ? fs.readFileSync(tasksMarkdownPath, "utf8") : "";
    const allCheckIds = new Set();
    const knownRequirementIds = new Set((traceability?.requirements || []).map((item) => item.id));
    for (const [taskId, task] of Object.entries(scope.tasks)) {
      taskIds.push(taskId);
      if (!tasksMarkdown.includes(`## ${taskId} -`)) issues.push(`TASKS.md is missing section for ${taskId}`);
      if (!isObject(task)) {
        issues.push(`invalid task entry: ${taskId}`);
        continue;
      }
      if (!Array.isArray(task.allowedFiles) || task.allowedFiles.length === 0) issues.push(`${taskId} has no allowedFiles`);
      if (!Array.isArray(task.forbiddenFiles)) issues.push(`${taskId} forbiddenFiles must be an array`);
      for (const pattern of task.allowedFiles || []) {
        if (isRepositoryRootDocsPattern(pattern)) issues.push(`${taskId} allowedFiles must not target repository-root docs/: use .spec/${featureName}/docs/** for feature-owned documentation`);
        if (schemaVersion === 2 && isBroadContractScope(pattern, featureName)) issues.push(`${taskId} v2 allowedFiles must not grant broad access to the feature contract directory`);
        if (schemaVersion === 2 && isContractOwnedPattern(pattern, featureName)) issues.push(`${taskId} v2 allowedFiles must not include contract-owned path: ${pattern}`);
      }
      for (const pattern of [...(task.allowedFiles || []), ...(task.forbiddenFiles || [])]) {
        try { compileGlob(pattern); } catch (error) { issues.push(`${taskId} has invalid glob: ${error.message}`); }
      }
      if ("requiredKnowledgeImpact" in task && typeof task.requiredKnowledgeImpact !== "boolean") issues.push(`${taskId} requiredKnowledgeImpact must be boolean when present`);
      if ("knowledge" in task) {
        if (!isObject(task.knowledge)) issues.push(`${taskId} knowledge must be an object when present`);
        else {
          for (const key of Object.keys(task.knowledge)) if (!["review", "modify", "candidate"].includes(key)) issues.push(`${taskId} knowledge has unknown key ${key}`);
          for (const key of ["review", "modify", "candidate"]) {
            if (!(key in task.knowledge)) continue;
            validateStringArray(task.knowledge[key], `${taskId} knowledge.${key}`, issues);
            for (const pattern of task.knowledge[key] || []) {
              try { compileGlob(pattern); } catch (error) { issues.push(`${taskId} knowledge.${key} has invalid glob: ${error.message}`); }
            }
          }
        }
      }
      const expectedAcceptance = `.spec/${featureName}/acceptance/${taskId}.md`;
      const acceptancePath = path.join(absoluteDir, "acceptance", `${taskId}.md`);
      if (task.acceptance !== expectedAcceptance) issues.push(`${taskId} acceptance path must be ${expectedAcceptance}`);
      if (!fs.existsSync(acceptancePath)) issues.push(`missing acceptance file for ${taskId}`);
      const expectedReport = `.spec/${featureName}/reports/${taskId}-report.md`;
      if (task.report !== expectedReport) issues.push(`${taskId} report path must be ${expectedReport}`);

      if (schemaVersion === 2) {
        if (!TASK_LIFECYCLES.has(task.lifecycle)) issues.push(`${taskId} lifecycle is invalid`);
        if (typeof task.requiresHumanConfirmation !== "boolean") issues.push(`${taskId} requiresHumanConfirmation must be boolean`);
        if (!REVIEW_POLICIES.has(task.reviewPolicy)) issues.push(`${taskId} reviewPolicy is invalid`);
        for (const field of ["requirements", "behaviorSurfaces", "dependencies", "destructiveActions"]) validateStringArray(task[field], `${taskId} ${field}`, issues);
        validateStringArray(task.allowedActions, `${taskId} allowedActions`, issues, { nonEmpty: true });
        for (const requirement of task.requirements || []) if (!knownRequirementIds.has(requirement)) issues.push(`${taskId} references unknown requirement: ${requirement}`);
        if (!Array.isArray(task.requiredChecks) || task.requiredChecks.length === 0) issues.push(`${taskId} requiredChecks must be a non-empty array`);
        const acceptanceText = fs.existsSync(acceptancePath) ? fs.readFileSync(acceptancePath, "utf8") : "";
        for (const [index, check] of (task.requiredChecks || []).entries()) {
          validateRequiredCheck(check, `${taskId} requiredChecks[${index}]`, issues);
          if (check?.id) {
            if (allCheckIds.has(check.id)) issues.push(`duplicate required check id: ${check.id}`);
            allCheckIds.add(check.id);
            if (!acceptanceText.includes(check.id)) issues.push(`${taskId} acceptance file does not reference required check ${check.id}`);
          }
        }
        for (const action of task.destructiveActions || []) if (!DESTRUCTIVE_ACTIONS.has(action)) issues.push(`${taskId} has invalid destructive action: ${action}`);
        if ((task.destructiveActions || []).length > 0 && task.requiresHumanConfirmation !== true) issues.push(`${taskId} destructive actions require human confirmation`);
        if ((task.destructiveActions || []).length > 0 && task.reviewPolicy !== "independent_required") issues.push(`${taskId} destructive actions require independent review`);
        if ((task.destructiveActions || []).includes("cutover") && (task.destructiveActions || []).includes("delete_source")) issues.push(`${taskId} cutover and delete_source must be separate TASKs`);
      }
    }
    if (schemaVersion === 2) {
      const knownTasks = new Set(taskIds);
      for (const [taskId, task] of Object.entries(scope.tasks)) for (const dependency of task.dependencies || []) if (!knownTasks.has(dependency)) issues.push(`${taskId} references unknown dependency: ${dependency}`);
      const rollingWindow = contract?.rollingWindow || 2;
      const readyTasks = Object.entries(scope.tasks).filter(([, task]) => task?.lifecycle === "ready").map(([taskId]) => taskId);
      if (readyTasks.length > rollingWindow) issues.push(`ready TASK count ${readyTasks.length} exceeds rollingWindow ${rollingWindow}`);
      for (const requirement of traceability?.requirements || []) {
        for (const taskId of requirement.task_ids || []) if (!knownTasks.has(taskId)) issues.push(`traceability requirement ${requirement.id} references unknown task: ${taskId}`);
        for (const checkId of requirement.check_ids || []) if (!allCheckIds.has(checkId)) issues.push(`traceability requirement ${requirement.id} references unknown check: ${checkId}`);
      }
    }
  }

  const pipelinePath = path.join(absoluteDir, "pipeline-state.json");
  if (options.requirePipeline && !fs.existsSync(pipelinePath)) issues.push("missing pipeline-state.json for pipeline execution");
  else if (fs.existsSync(pipelinePath)) {
    const state = readJson(pipelinePath, issues, "pipeline-state.json");
    if (state && state.feature_name !== featureName) issues.push("pipeline-state.json feature_name does not match directory");
    if (state && schemaVersion === 2 && state.schemaVersion !== 2) issues.push("pipeline-state.json schemaVersion must match task-scope.json");
  }

  if (issues.length > 0) classification = CLASSIFICATIONS.UNSUPPORTED;
  return { feature_name: featureName, feature_dir: absoluteDir, classification, schema_version: schemaVersion, profile, contract_revision: contractRevision, issues, warnings, task_ids: taskIds.sort() };
}

function parseArgs(argv) {
  const result = { feature: null, requirePipeline: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--feature") result.feature = argv[++index];
    else if (argv[index] === "--require-pipeline") result.requirePipeline = true;
    else if (argv[index] === "--json") result.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!result.feature) throw new Error("--feature is required");
  return result;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = validateFeature(args.feature, { requirePipeline: args.requirePipeline });
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      console.log(`feature: ${result.feature_name}`);
      console.log(`classification: ${result.classification}`);
      console.log(`schema_version: ${result.schema_version ?? "legacy"}`);
      if (result.profile) console.log(`profile: ${result.profile}`);
      for (const warning of result.warnings) console.log(`warning: ${warning}`);
      for (const issue of result.issues) console.error(`error: ${issue}`);
    }
    process.exit(result.classification === CLASSIFICATIONS.UNSUPPORTED ? 1 : 0);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
