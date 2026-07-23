#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { hashFile } from "./contract-utils.mjs";
import { validateAmendment } from "./validate-amendment.mjs";
import { validateFeature, CLASSIFICATIONS } from "./validate-feature.mjs";

function readJson(file) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function writeJson(file, value) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
}

function safeRelative(value, label) {
  if (typeof value !== "string" || value.length === 0 || path.isAbsolute(value)) throw new Error(`${label} must be a relative path`);
  const normalized = path.posix.normalize(value.replaceAll("\\", "/"));
  if (normalized === ".." || normalized.startsWith("../")) throw new Error(`${label} escapes the feature directory`);
  return normalized;
}

function isContractTarget(relative, featureName) {
  const exact = new Set(["contract.json", "traceability.json", "task-scope.json", "TASKS.md", "AGENT_RULES.md", `${featureName}-SPEC.md`, `${featureName}-SDD.md`]);
  return exact.has(relative) || relative.startsWith("acceptance/") || relative.startsWith("baseline/") || relative.startsWith("prompts/");
}

function restore(backups) {
  for (const [file, content] of [...backups.entries()].reverse()) {
    if (content === null) fs.rmSync(file, { force: true });
    else {
      fs.mkdirSync(path.dirname(file), { recursive: true });
      fs.writeFileSync(file, content);
    }
  }
}

export function applyAmendment({ featureDir, crFile, apply = false }) {
  const absoluteDir = path.resolve(featureDir);
  const featureName = path.basename(absoluteDir);
  const absoluteCr = path.resolve(crFile);
  if (!absoluteCr.startsWith(`${absoluteDir}${path.sep}`)) throw new Error("CR must be inside the feature directory");
  const amendment = readJson(absoluteCr);
  const validation = validateAmendment(amendment);
  if (!validation.valid) throw new Error(`invalid amendment: ${validation.issues.join("; ")}`);
  if (amendment.status !== "approved") throw new Error("only an approved amendment can be applied");
  if (!Array.isArray(amendment.updates) || amendment.updates.length === 0) throw new Error("approved amendment has no staged updates");
  const contractPath = path.join(absoluteDir, "contract.json");
  const contract = readJson(contractPath);
  if (contract.contractRevision !== amendment.baseRevision) throw new Error("CR baseRevision does not match active contract");
  const stagedRoot = `changes/${amendment.changeRequestId}/`;
  const plan = [];
  for (const update of amendment.updates) {
    const target = safeRelative(update.target, "update.target");
    const staged = safeRelative(update.staged_file, "update.staged_file");
    if (!isContractTarget(target, featureName)) throw new Error(`update target is not contract-owned: ${target}`);
    if (!staged.startsWith(stagedRoot)) throw new Error(`staged_file must be under ${stagedRoot}`);
    const targetFile = path.join(absoluteDir, target);
    const stagedFile = path.join(absoluteDir, staged);
    if (!fs.existsSync(stagedFile) || !fs.statSync(stagedFile).isFile()) throw new Error(`staged file is missing: ${staged}`);
    const before = fs.existsSync(targetFile) ? hashFile(targetFile) : null;
    if (before !== update.before_sha256) throw new Error(`before hash mismatch for ${target}`);
    const after = hashFile(stagedFile);
    if (after !== update.after_sha256) throw new Error(`after hash mismatch for ${target}`);
    plan.push({ target, staged, targetFile, stagedFile, before, after });
  }
  if (!plan.some((item) => item.target === "contract.json")) throw new Error("amendment must stage contract.json");
  const stagedContract = readJson(plan.find((item) => item.target === "contract.json").stagedFile);
  if (stagedContract.contractRevision !== amendment.resultingRevision) throw new Error("staged contract revision does not match CR resultingRevision");
  if (!apply) return { applied: false, feature_name: featureName, change_request: amendment.changeRequestId, resulting_revision: amendment.resultingRevision, updates: plan.map(({ target, staged, before, after }) => ({ target, staged, before, after })) };

  const statePath = path.join(absoluteDir, "pipeline-state.json");
  const revisionPath = path.join(absoluteDir, "revisions", `REV-${String(amendment.resultingRevision).padStart(4, "0")}.json`);
  const backups = new Map();
  for (const file of [...plan.map((item) => item.targetFile), statePath, absoluteCr, revisionPath]) backups.set(file, fs.existsSync(file) ? fs.readFileSync(file) : null);
  try {
    for (const item of plan) {
      fs.mkdirSync(path.dirname(item.targetFile), { recursive: true });
      const temporary = `${item.targetFile}.amendment-${process.pid}.tmp`;
      fs.copyFileSync(item.stagedFile, temporary);
      fs.renameSync(temporary, item.targetFile);
    }
    const state = fs.existsSync(statePath) ? readJson(statePath) : null;
    let carriedForwardTasks = [];
    if (state) {
      const revalidationTasks = amendment.impact.completedTasksRequiringRevalidation || [];
      carriedForwardTasks = (state.completed_tasks || []).filter((taskId) => !revalidationTasks.includes(taskId));
      state.contract_revision = amendment.resultingRevision;
      state.open_change_requests = (state.open_change_requests || []).filter((item) => item !== amendment.changeRequestId);
      state.needs_revalidation_tasks = [...new Set([...(state.needs_revalidation_tasks || []), ...revalidationTasks])];
      state.current_phase = state.needs_revalidation_tasks.length > 0 ? "revalidate" : "replan";
      state.status = "in_progress";
      writeJson(statePath, state);
    }
    amendment.status = "applied";
    amendment.applied_at = new Date().toISOString();
    writeJson(absoluteCr, amendment);
    writeJson(revisionPath, {
      schemaVersion: 2,
      feature_name: featureName,
      revision: amendment.resultingRevision,
      change_request: amendment.changeRequestId,
      applied_at: amendment.applied_at,
      updates: plan.map(({ target, after }) => ({ target, sha256: after })),
      revalidation_tasks: amendment.impact.completedTasksRequiringRevalidation || [],
      carried_forward_tasks: carriedForwardTasks,
    });
    const featureResult = validateFeature(absoluteDir, { requirePipeline: fs.existsSync(statePath) });
    if (featureResult.classification === CLASSIFICATIONS.UNSUPPORTED) throw new Error(`applied contract is invalid: ${featureResult.issues.join("; ")}`);
    return { applied: true, feature_name: featureName, change_request: amendment.changeRequestId, resulting_revision: amendment.resultingRevision, updates: plan.map((item) => item.target) };
  } catch (error) {
    restore(backups);
    throw error;
  }
}

function parseArgs(argv) {
  const args = { featureDir: null, crFile: null, apply: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--feature-dir") args.featureDir = argv[++index];
    else if (argv[index] === "--cr") args.crFile = argv[++index];
    else if (argv[index] === "--apply") args.apply = true;
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!args.featureDir || !args.crFile) throw new Error("--feature-dir and --cr are required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = applyAmendment(args);
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      console.log(`change_request: ${result.change_request}`);
      console.log(`resulting_revision: ${result.resulting_revision}`);
      console.log(`amendment_result: ${result.applied ? "APPLIED" : "DRY_RUN_PASS"}`);
    }
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(1);
  }
}
