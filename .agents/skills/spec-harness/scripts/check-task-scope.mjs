#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync } from "node:child_process";
import { compileGlob, validateFeature, CLASSIFICATIONS } from "./validate-feature.mjs";

function git(root, args, allowFailure = false) {
  try {
    return execFileSync("git", ["-C", root, ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
  } catch (error) {
    if (allowFailure) return null;
    throw new Error(`git ${args.join(" ")} failed: ${String(error.stderr || error.message).trim()}`);
  }
}

function lines(value) {
  return String(value || "").split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
}

function baselineBlob(root, tree, file) {
  return git(root, ["rev-parse", `${tree}:${file}`], true);
}

function workingBlob(root, file) {
  const absolute = path.join(root, file);
  if (!fs.existsSync(absolute) || fs.statSync(absolute).isDirectory()) return null;
  return git(root, ["hash-object", "--", file], true);
}

export function collectChangedFiles(root, baseTree) {
  if (!baseTree) throw new Error("a reliable --base-tree is required");
  if (!git(root, ["rev-parse", "--verify", `${baseTree}^{tree}`], true)) throw new Error(`invalid base tree: ${baseTree}`);

  const changed = new Set(lines(git(root, ["diff", "--name-only", baseTree, "--"])));
  const untracked = lines(git(root, ["ls-files", "--others", "--exclude-standard"]));
  for (const file of untracked) {
    const before = baselineBlob(root, baseTree, file);
    const after = workingBlob(root, file);
    if (before && before === after) changed.delete(file);
    else changed.add(file);
  }
  return [...changed].sort();
}

function collectDeletedFiles(root, baseTree) {
  return new Set(lines(git(root, ["diff", "--diff-filter=D", "--name-only", baseTree, "--"])));
}

function firstMatch(file, patterns) {
  for (const pattern of patterns || []) {
    if (compileGlob(pattern).test(file)) return pattern;
  }
  return null;
}

function isOrchestratorOwned(file, featureName, taskId) {
  const prefix = `.spec/${featureName}/`;
  return file === `${prefix}pipeline-state.json`
    || file === `${prefix}reports/${taskId}-report.md`
    || file === `${prefix}reports/${taskId}-evidence.json`
    || file === `${prefix}reports/pipeline-summary.md`;
}

export function checkTaskScope({ root, featureDir, taskId, baseTree }) {
  const feature = validateFeature(featureDir, { requirePipeline: false });
  if (feature.classification === CLASSIFICATIONS.UNSUPPORTED) {
    throw new Error(`feature is unsupported: ${feature.issues.join("; ")}`);
  }
  const scope = JSON.parse(fs.readFileSync(path.join(featureDir, "task-scope.json"), "utf8"));
  const task = scope.tasks?.[taskId];
  if (!task) throw new Error(`unknown TASK-ID: ${taskId}`);
  if (scope.schemaVersion === 2 && !["ready", "in_progress", "review"].includes(task.lifecycle)) {
    throw new Error(`TASK ${taskId} lifecycle ${task.lifecycle} is not executable; reconcile or amend the plan first`);
  }

  const changedFiles = collectChangedFiles(root, baseTree);
  const deletedFiles = collectDeletedFiles(root, baseTree);
  const policyIssues = [];
  const rows = changedFiles.map((file) => {
    const forbidden = firstMatch(file, task.forbiddenFiles);
    const allowed = firstMatch(file, task.allowedFiles);
    if (forbidden) return { file, result: "FORBIDDEN", pattern: forbidden };
    if (scope.schemaVersion === 2 && isOrchestratorOwned(file, scope.feature_name, taskId)) return { file, result: "ORCHESTRATOR", pattern: "canonical runtime artifact" };
    if (scope.schemaVersion === 2 && deletedFiles.has(file) && !(task.allowedActions || []).includes("delete")) {
      return { file, result: "ACTION_NOT_ALLOWED", pattern: "allowedActions lacks delete" };
    }
    if (allowed) return { file, result: "ALLOWED", pattern: allowed };
    return { file, result: "OUT_OF_SCOPE", pattern: null };
  });
  if (scope.schemaVersion === 2 && deletedFiles.size > 20 && !(task.destructiveActions || []).some((action) => ["bulk_delete", "delete_source"].includes(action))) {
    policyIssues.push(`TASK deletes ${deletedFiles.size} files but does not declare bulk_delete or delete_source`);
  }
  const failed = rows.some((row) => !["ALLOWED", "ORCHESTRATOR"].includes(row.result)) || policyIssues.length > 0;
  return { feature_name: scope.feature_name, task_id: taskId, base_tree: baseTree, classification: feature.classification, rows, policy_issues: policyIssues, failed };
}

function parseArgs(argv) {
  const args = { root: process.cwd(), featureDir: null, taskId: null, baseTree: process.env.TASK_BASE_TREE || null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const item = argv[index];
    if (item === "--root") args.root = path.resolve(argv[++index]);
    else if (item === "--feature-dir") args.featureDir = path.resolve(argv[++index]);
    else if (item === "--task") args.taskId = argv[++index];
    else if (item === "--base-tree") args.baseTree = argv[++index];
    else if (item === "--json") args.json = true;
    else throw new Error(`unknown argument: ${item}`);
  }
  if (!args.featureDir) throw new Error("--feature-dir is required");
  if (!args.taskId) throw new Error("--task is required");
  if (!args.baseTree) throw new Error("--base-tree or TASK_BASE_TREE is required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = checkTaskScope(args);
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      console.log(`feature: ${result.feature_name}`);
      console.log(`task: ${result.task_id}`);
      console.log(`base_tree: ${result.base_tree}`);
      console.log(`classification: ${result.classification}`);
      if (result.rows.length === 0) console.log("No TASK-local changed files detected.");
      for (const row of result.rows) console.log(`${row.result}\t${row.file}\t${row.pattern || "-"}`);
      for (const issue of result.policy_issues || []) console.error(`POLICY\t${issue}`);
      console.log(`scope_result: ${result.failed ? "FAIL" : "PASS"} (${result.task_id})`);
    }
    process.exit(result.failed ? 1 : 0);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
