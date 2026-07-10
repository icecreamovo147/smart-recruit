#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { execFileSync } from "node:child_process";
import { compileGlob, scanKnowledge } from "./validate-knowledge.mjs";

function git(root, args, allowFailure = false) {
  try { return execFileSync("git", ["-C", root, ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim(); }
  catch (error) {
    if (allowFailure) return null;
    throw new Error(`git ${args.join(" ")} failed: ${String(error.stderr || error.message).trim()}`);
  }
}

function lines(value) { return String(value || "").split(/\r?\n/).map((line) => line.trim()).filter(Boolean); }

function baselineBlob(root, tree, file) { return git(root, ["rev-parse", `${tree}:${file}`], true); }
function workingBlob(root, file) {
  const absolute = path.join(root, file);
  if (!fs.existsSync(absolute) || fs.statSync(absolute).isDirectory()) return null;
  return git(root, ["hash-object", "--", file], true);
}

export function collectChangedFiles(root, baseTree) {
  if (!baseTree) throw new Error("a reliable --base-tree is required");
  if (!git(root, ["rev-parse", "--verify", `${baseTree}^{tree}`], true)) throw new Error(`invalid base tree: ${baseTree}`);
  const changed = new Set(lines(git(root, ["diff", "--name-only", baseTree, "--"])));
  for (const file of lines(git(root, ["ls-files", "--others", "--exclude-standard"]))) {
    const before = baselineBlob(root, baseTree, file);
    const after = workingBlob(root, file);
    if (before && before === after) changed.delete(file); else changed.add(file);
  }
  return [...changed].map((file) => file.split(path.sep).join("/")).sort();
}

function collectStatuses(root, baseTree, changedFiles) {
  const statuses = new Map();
  for (const line of lines(git(root, ["diff", "--name-status", baseTree, "--"]))) {
    const parts = line.split("\t");
    const status = parts[0][0];
    const file = (parts.length > 2 ? parts[2] : parts[1]).split(path.sep).join("/");
    statuses.set(file, status);
  }
  for (const file of changedFiles) if (!statuses.has(file)) statuses.set(file, "A");
  return statuses;
}

function baseHasPath(root, baseTree, repoPath) { return git(root, ["cat-file", "-e", `${baseTree}:${repoPath}`], true) !== null; }

function addIfDeclared(target, declared, trigger, condition) { if (condition && declared.has(trigger)) target.add(trigger); }

function looksLikeCoreSurface(file) {
  return file === "db.sql" || /(^|\/)proto\//.test(file) || /(^|\/)config\//.test(file) || /(^|\/)service\/[^/]+\.go$/.test(file) || /(^|\/)router\/[^/]+\.(go|ts)$/.test(file);
}

export function detectImpact({ root, baseTree }) {
  const repoRoot = path.resolve(root);
  const validation = scanKnowledge(repoRoot);
  if (validation.errors.length) throw new Error(`knowledge validation failed: ${validation.errors.join("; ")}`);
  const manifest = validation.manifest;
  const changedFiles = collectChangedFiles(repoRoot, baseTree);
  const statuses = collectStatuses(repoRoot, baseTree, changedFiles);
  const matchedRoutes = [];
  const documentIds = new Set();
  const triggers = new Set();
  const matchedFiles = new Set();
  for (const [index, route] of manifest.routes.entries()) {
    const patterns = route.match.map((pattern) => ({ pattern, regex: compileGlob(pattern) }));
    const files = changedFiles.filter((file) => patterns.some(({ regex }) => regex.test(file)));
    if (!files.length) continue;
    files.forEach((file) => matchedFiles.add(file));
    route.documents.forEach((id) => documentIds.add(id));
    route.triggers.forEach((trigger) => triggers.add(trigger));
    matchedRoutes.push({ index, files, documents: [...route.documents].sort(), triggers: [...route.triggers].sort() });
  }
  const coverageGaps = changedFiles.filter((file) => looksLikeCoreSurface(file) && !matchedFiles.has(file));
  const declared = new Set(manifest.global_triggers || []);
  const addedTopLevels = changedFiles.filter((file) => file.includes("/") && statuses.get(file) === "A" && !baseHasPath(repoRoot, baseTree, file.split("/")[0]));
  addIfDeclared(triggers, declared, "new-top-level-module", addedTopLevels.length > 0);
  addIfDeclared(triggers, declared, "database-schema-changed", changedFiles.some((file) => file === "db.sql" || /(^|\/)migrations\//.test(file)));
  addIfDeclared(triggers, declared, "public-api-changed", changedFiles.some((file) => /(^|\/)proto\//.test(file) || /(^|\/)(handler|router)\//.test(file)));
  addIfDeclared(triggers, declared, "service-boundary-changed", changedFiles.some((file) => /(^|\/)(rpc|service)\//.test(file) && (statuses.get(file) === "A" || statuses.get(file) === "D" || /services\.go$/.test(file))));
  addIfDeclared(triggers, declared, "authorization-changed", changedFiles.some((file) => /(^|\/)(auth|authz|permission|rbac)(\/|_|\.)/i.test(file)));
  addIfDeclared(triggers, declared, "sensitive-data-flow-changed", changedFiles.some((file) => /(resume|candidate|privacy|security|oss)/i.test(file)));
  addIfDeclared(triggers, declared, "configuration-changed", changedFiles.some((file) => /(^|\/)(config|deploy|docker)(\/|\.)|(^|\/)\.env/i.test(file)));
  const changedBusinessTests = changedFiles.some((file) => !file.startsWith(".knowledge/") && !file.startsWith(".spec/") && /(_test\.go|\.test\.[jt]s|\/__tests__\/)/.test(file));
  const changedBusinessSource = changedFiles.some((file) => !file.startsWith(".knowledge/") && !file.startsWith(".spec/") && !/(_test\.go|\.test\.[jt]s|\/__tests__\/)/.test(file));
  addIfDeclared(triggers, declared, "recurring-defect-fixed", changedBusinessTests && changedBusinessSource);
  const hasGlobalTrigger = [...triggers].some((trigger) => declared.has(trigger));
  return {
    schema_version: 1,
    base_tree: baseTree,
    changed_files: changedFiles,
    matched_routes: matchedRoutes,
    triggered_by: [...triggers].sort(),
    reviewed_documents: [...documentIds].sort(),
    coverage_gaps: coverageGaps,
    knowledge_impact: {
      result: coverageGaps.length ? "coverage_gap" : (documentIds.size ? "update_required" : (hasGlobalTrigger ? "candidate_required" : "none")),
      triggered_by: [...triggers].sort(),
      reviewed_documents: [...documentIds].sort(),
      update_paths: [],
      coverage_gap: coverageGaps.length > 0,
      evidence: changedFiles,
      reason: documentIds.size || coverageGaps.length ? "Mechanical routing requires Agent review." : "No knowledge route matched."
    }
  };
}

function parseArgs(argv) {
  const args = { root: process.cwd(), baseTree: process.env.TASK_BASE_TREE || null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--root") args.root = path.resolve(argv[++index]);
    else if (argv[index] === "--base-tree") args.baseTree = argv[++index];
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!args.baseTree) throw new Error("--base-tree or TASK_BASE_TREE is required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  let args;
  try { args = parseArgs(process.argv.slice(2)); }
  catch (error) { console.error(`error: ${error.message}`); process.exit(2); }
  try {
    const result = detectImpact(args);
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      console.log(`base_tree: ${result.base_tree}`);
      console.log(`changed_files: ${result.changed_files.length}`);
      console.log(`matched_routes: ${result.matched_routes.length}`);
      for (const id of result.reviewed_documents) console.log(`review_document: ${id}`);
      for (const file of result.coverage_gaps) console.log(`coverage_gap: ${file}`);
      console.log(`impact_result: ${result.knowledge_impact.result}`);
    }
  } catch (error) { console.error(`error: ${error.message}`); process.exit(1); }
}
