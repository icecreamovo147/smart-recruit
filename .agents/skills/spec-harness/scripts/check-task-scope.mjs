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

function firstMatch(file, patterns) {
  for (const pattern of patterns || []) {
    if (compileGlob(pattern).test(file)) return pattern;
  }
  return null;
}

export function checkTaskScope({ root, featureDir, taskId, baseTree }) {
  const feature = validateFeature(featureDir, { requirePipeline: false });
  if (feature.classification === CLASSIFICATIONS.UNSUPPORTED) {
    throw new Error(`feature is unsupported: ${feature.issues.join("; ")}`);
  }
  const scope = JSON.parse(fs.readFileSync(path.join(featureDir, "task-scope.json"), "utf8"));
  const task = scope.tasks?.[taskId];
  if (!task) throw new Error(`unknown TASK-ID: ${taskId}`);

  const changedFiles = collectChangedFiles(root, baseTree);
  const rows = changedFiles.map((file) => {
    const forbidden = firstMatch(file, task.forbiddenFiles);
    const allowed = firstMatch(file, task.allowedFiles);
    if (forbidden) return { file, result: "FORBIDDEN", pattern: forbidden };
    if (allowed) return { file, result: "ALLOWED", pattern: allowed };
    return { file, result: "OUT_OF_SCOPE", pattern: null };
  });
  const failed = rows.some((row) => row.result !== "ALLOWED");
  return { feature_name: scope.feature_name, task_id: taskId, base_tree: baseTree, classification: feature.classification, rows, failed };
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
      console.log(`scope_result: ${result.failed ? "FAIL" : "PASS"} (${result.task_id})`);
    }
    process.exit(result.failed ? 1 : 0);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
