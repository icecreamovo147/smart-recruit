#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";

export const KINDS = new Set([
  "architecture", "domain", "decision", "runbook", "pitfall",
  "navigation", "security", "public-contract", "candidate",
]);
export const STATUSES = new Set(["draft", "active", "stale", "deprecated", "archived"]);
export const DECISION_STATUSES = new Set(["proposed", "accepted", "rejected", "superseded"]);

export function normalizeRepoPath(value) {
  if (typeof value !== "string" || value.trim() === "") throw new Error("path must be a non-empty string");
  const raw = value.trim();
  if (/^[A-Za-z]:[\\/]/.test(raw) || raw.startsWith("/") || raw.startsWith("~") || raw.includes("\\")) {
    throw new Error(`path must be repository-relative POSIX form: ${raw}`);
  }
  const normalized = path.posix.normalize(raw);
  if (normalized === ".." || normalized.startsWith("../")) throw new Error(`path escapes repository root: ${raw}`);
  return normalized.replace(/^\.\//, "");
}

function scalar(value) {
  const text = value.trim();
  if (text === "null") return null;
  if (text === "true") return true;
  if (text === "false") return false;
  if (text === "[]") return [];
  if (text === "{}") return {};
  if (/^-?\d+$/.test(text)) return Number(text);
  if ((text.startsWith('"') && text.endsWith('"')) || (text.startsWith("'") && text.endsWith("'"))) return text.slice(1, -1);
  return text;
}

export function parseFrontmatter(content, file = "<memory>") {
  const normalized = content.replace(/\r\n/g, "\n");
  if (!normalized.startsWith("---\n")) throw new Error(`${file}: missing opening frontmatter delimiter`);
  const end = normalized.indexOf("\n---\n", 4);
  if (end < 0) throw new Error(`${file}: missing closing frontmatter delimiter`);
  const lines = normalized.slice(4, end).split("\n");
  const data = {};
  let listKey = null;
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (!line.trim() || line.trimStart().startsWith("#")) continue;
    const list = line.match(/^\s{2}-\s+(.+)$/);
    if (list) {
      if (!listKey || !Array.isArray(data[listKey])) throw new Error(`${file}:${index + 2}: list item without a top-level list key`);
      data[listKey].push(scalar(list[1]));
      continue;
    }
    if (/^\s/.test(line)) throw new Error(`${file}:${index + 2}: only top-level scalars and two-space lists are supported`);
    const field = line.match(/^([a-z_]+):(?:\s*(.*))?$/);
    if (!field) throw new Error(`${file}:${index + 2}: unsupported frontmatter syntax`);
    const [, key, raw = ""] = field;
    if (Object.hasOwn(data, key)) throw new Error(`${file}:${index + 2}: duplicate key ${key}`);
    if (raw === "") {
      data[key] = [];
      listKey = key;
    } else {
      data[key] = scalar(raw);
      listKey = null;
    }
  }
  return { data, body: normalized.slice(end + 5) };
}

export function parseManifest(content, file = "manifest.yaml") {
  const result = { policies: {}, routes: [], global_triggers: [] };
  const seenTop = new Set();
  const seenPolicies = new Set();
  const lines = content.replace(/\r\n/g, "\n").split("\n");
  let section = null;
  let policyList = null;
  let route = null;
  let routeList = null;
  let routeKeys = null;
  const allowedTopKeys = new Set(["schema_version", "policies", "routes", "global_triggers"]);
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const topScalar = line.match(/^([a-z_]+):\s*(.+)$/);
    if (topScalar) {
      if (!allowedTopKeys.has(topScalar[1])) throw new Error(`${file}:${index + 1}: unknown top-level key ${topScalar[1]}`);
      if (seenTop.has(topScalar[1])) throw new Error(`${file}:${index + 1}: duplicate key ${topScalar[1]}`);
      seenTop.add(topScalar[1]);
      result[topScalar[1]] = scalar(topScalar[2]);
      section = null;
      continue;
    }
    const topSection = line.match(/^([a-z_]+):\s*$/);
    if (topSection) {
      if (!allowedTopKeys.has(topSection[1])) throw new Error(`${file}:${index + 1}: unknown top-level section ${topSection[1]}`);
      if (seenTop.has(topSection[1])) throw new Error(`${file}:${index + 1}: duplicate key ${topSection[1]}`);
      seenTop.add(topSection[1]);
      section = topSection[1];
      policyList = null;
      routeList = null;
      continue;
    }
    if (section === "policies") {
      const field = line.match(/^\s{2}([a-z_]+):(?:\s*(.*))?$/);
      if (field) {
        const [, key, raw = ""] = field;
        if (seenPolicies.has(key)) throw new Error(`${file}:${index + 1}: duplicate policy key ${key}`);
        seenPolicies.add(key);
        result.policies[key] = raw === "" ? [] : scalar(raw);
        policyList = raw === "" ? key : null;
        continue;
      }
      const item = line.match(/^\s{4}-\s+(.+)$/);
      if (item && policyList) { result.policies[policyList].push(scalar(item[1])); continue; }
    }
    if (section === "routes") {
      const start = line.match(/^\s{2}-\s+([a-z_]+):\s*$/);
      if (start) {
        if (start[1] !== "match") throw new Error(`${file}:${index + 1}: route must start with match`);
        route = { match: [], documents: [], triggers: [] };
        result.routes.push(route);
        routeList = start[1];
        routeKeys = new Set([start[1]]);
        continue;
      }
      const field = line.match(/^\s{4}([a-z_]+):\s*$/);
      if (field && route) {
        if (!["documents", "triggers"].includes(field[1])) throw new Error(`${file}:${index + 1}: unknown route key ${field[1]}`);
        if (routeKeys.has(field[1])) throw new Error(`${file}:${index + 1}: duplicate route key ${field[1]}`);
        routeKeys.add(field[1]);
        routeList = field[1];
        continue;
      }
      const item = line.match(/^\s{6}-\s+(.+)$/);
      if (item && route && Array.isArray(route[routeList])) { route[routeList].push(scalar(item[1])); continue; }
    }
    if (section === "global_triggers") {
      const item = line.match(/^\s{2}-\s+(.+)$/);
      if (item) { result.global_triggers.push(scalar(item[1])); continue; }
    }
    throw new Error(`${file}:${index + 1}: unsupported manifest syntax: ${trimmed}`);
  }
  for (const key of allowedTopKeys) if (!seenTop.has(key)) throw new Error(`${file}: missing required top-level key ${key}`);
  return result;
}

export function compileGlob(pattern) {
  const normalized = normalizeRepoPath(pattern);
  if (normalized.includes("[") || normalized.includes("]")) throw new Error(`unsupported character class syntax in glob: ${pattern}`);
  let source = "^";
  for (let index = 0; index < normalized.length; index += 1) {
    const character = normalized[index];
    if (character === "*" && normalized[index + 1] === "*") { source += ".*"; index += 1; }
    else if (character === "*") source += "[^/]*";
    else if (character === "?") source += "[^/]";
    else source += character.replace(/[|\\{}()[\]^$+?.]/g, "\\$&");
  }
  return new RegExp(`${source}$`);
}

function walk(root) {
  if (!fs.existsSync(root)) return [];
  const files = [];
  for (const entry of fs.readdirSync(root, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    if ([".local", "generated"].includes(entry.name)) continue;
    const candidate = path.join(root, entry.name);
    if (entry.isDirectory()) files.push(...walk(candidate));
    else files.push(candidate);
  }
  return files;
}

export function findCaseConflicts(paths) {
  const seen = new Map();
  const conflicts = [];
  for (const value of paths) {
    const folded = value.toLowerCase();
    if (seen.has(folded) && seen.get(folded) !== value) conflicts.push([seen.get(folded), value]);
    else seen.set(folded, value);
  }
  return conflicts;
}

function isFormalDocument(relative) {
  if (!relative.endsWith(".md") || path.posix.basename(relative) === "README.md") return false;
  return ["architecture/", "domains/", "decisions/", "runbooks/", "pitfalls/", "inbox/", "archive/"].some((prefix) => relative.startsWith(prefix));
}

function validDate(value) {
  if (typeof value !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const [year, month, day] = value.split("-").map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day;
}

function validateStringList(item, field, relative, errors, pattern = null) {
  const values = item[field];
  if (!Array.isArray(values) || values.length === 0) { errors.push(`${relative}: ${field} must be a non-empty list`); return; }
  if (new Set(values).size !== values.length) errors.push(`${relative}: ${field} must contain unique items`);
  for (const value of values) {
    if (typeof value !== "string" || value.length === 0) errors.push(`${relative}: ${field} items must be non-empty strings`);
    else if (pattern && !pattern.test(value)) errors.push(`${relative}: invalid ${field} item ${value}`);
  }
}

function validateManifestList(values, label, errors, pattern = null) {
  if (!Array.isArray(values) || values.length === 0) { errors.push(`${label} must be a non-empty list`); return; }
  if (new Set(values).size !== values.length) errors.push(`${label} must contain unique items`);
  for (const value of values) {
    if (typeof value !== "string" || value.length === 0) errors.push(`${label} items must be non-empty strings`);
    else if (pattern && !pattern.test(value)) errors.push(`${label} has invalid item ${value}`);
  }
}

export function scanKnowledge(root) {
  const repoRoot = path.resolve(root);
  const knowledgeRoot = path.join(repoRoot, ".knowledge");
  const errors = [];
  const warnings = [];
  const documents = [];
  if (!fs.existsSync(knowledgeRoot)) return { root: repoRoot, errors: ["missing .knowledge directory"], warnings, documents, manifest: null };

  let manifest = null;
  try { manifest = parseManifest(fs.readFileSync(path.join(knowledgeRoot, "manifest.yaml"), "utf8")); }
  catch (error) { errors.push(error.message); }

  const knowledgeFiles = walk(knowledgeRoot);
  const relativeFiles = knowledgeFiles.map((absolute) => path.relative(knowledgeRoot, absolute).split(path.sep).join("/"));
  for (const [first, second] of findCaseConflicts(relativeFiles)) errors.push(`case-conflicting paths: ${first} and ${second}`);
  for (const absolute of knowledgeFiles) {
    const relative = path.relative(knowledgeRoot, absolute).split(path.sep).join("/");
    if (!isFormalDocument(relative)) continue;
    try {
      const raw = fs.readFileSync(absolute, "utf8");
      if (/-----BEGIN [A-Z ]*PRIVATE KEY-----/.test(raw) || /\bsk-[A-Za-z0-9_-]{20,}\b/.test(raw)) errors.push(`${relative}: possible secret material`);
      const { data } = parseFrontmatter(raw, relative);
      documents.push({ path: relative, absolute, metadata: data });
    } catch (error) { errors.push(error.message); }
  }

  const byId = new Map();
  const required = ["schema_version", "id", "title", "kind", "status", "owners", "tags", "applies_to", "source_refs", "last_verified", "review_after"];
  for (const document of documents) {
    const { metadata: item, path: relative } = document;
    for (const field of required) if (!(field in item)) errors.push(`${relative}: missing field ${field}`);
    if (item.schema_version !== 1) errors.push(`${relative}: schema_version must be 1`);
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(item.id || "")) errors.push(`${relative}: invalid id`);
    if (typeof item.title !== "string" || item.title.trim() === "") errors.push(`${relative}: title must be a non-empty string`);
    if (byId.has(item.id)) errors.push(`duplicate id ${item.id}: ${byId.get(item.id)} and ${relative}`); else byId.set(item.id, relative);
    if (!KINDS.has(item.kind)) errors.push(`${relative}: invalid kind ${item.kind}`);
    if (!STATUSES.has(item.status)) errors.push(`${relative}: invalid status ${item.status}`);
    validateStringList(item, "owners", relative, errors, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    validateStringList(item, "tags", relative, errors, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    validateStringList(item, "applies_to", relative, errors);
    validateStringList(item, "source_refs", relative, errors);
    for (const field of ["applies_to", "source_refs"]) for (const value of item[field] || []) {
      try { normalizeRepoPath(value); } catch (error) { errors.push(`${relative}: ${error.message}`); }
    }
    for (const ref of item.source_refs || []) {
      if (/[*?]/.test(ref)) { errors.push(`${relative}: source_refs must be exact paths: ${ref}`); continue; }
      if (item.status === "active" && !fs.existsSync(path.join(repoRoot, ref))) errors.push(`${relative}: missing source_ref ${ref}`);
    }
    if (!validDate(item.last_verified)) errors.push(`${relative}: invalid last_verified`);
    if (!validDate(item.review_after)) errors.push(`${relative}: invalid review_after`);
    if (relative.startsWith("inbox/") && (item.kind !== "candidate" || item.status !== "draft")) errors.push(`${relative}: Inbox documents must be candidate/draft`);
    if (relative.startsWith("archive/") && !["deprecated", "archived"].includes(item.status)) errors.push(`${relative}: Archive document status must be deprecated or archived`);
    if (item.kind === "decision") {
      if (!DECISION_STATUSES.has(item.decision_status)) errors.push(`${relative}: invalid decision_status`);
      if (!Array.isArray(item.supersedes)) errors.push(`${relative}: supersedes must be a list`);
      if (!("superseded_by" in item)) errors.push(`${relative}: missing superseded_by`);
    }
  }

  for (const document of documents.filter((entry) => entry.metadata.kind === "decision")) {
    for (const id of document.metadata.supersedes || []) {
      if (id === document.metadata.id) errors.push(`${document.path}: ADR cannot supersede itself`);
      const targetPath = byId.get(id);
      const target = documents.find((entry) => entry.path === targetPath);
      if (!target) errors.push(`${document.path}: unknown supersedes ADR ${id}`);
      else if (target.metadata.kind !== "decision") errors.push(`${document.path}: supersedes target ${id} is not an ADR`);
      else if (target.metadata.superseded_by !== document.metadata.id || target.metadata.decision_status !== "superseded") errors.push(`${document.path}: supersedes relationship with ${id} is not reciprocal`);
    }
    if (document.metadata.superseded_by) {
      if (document.metadata.superseded_by === document.metadata.id) errors.push(`${document.path}: ADR cannot be superseded by itself`);
      const targetPath = byId.get(document.metadata.superseded_by);
      const target = documents.find((entry) => entry.path === targetPath);
      if (!target) errors.push(`${document.path}: unknown superseded_by ADR ${document.metadata.superseded_by}`);
      else if (target.metadata.kind !== "decision") errors.push(`${document.path}: superseded_by target ${document.metadata.superseded_by} is not an ADR`);
      else if (!(target.metadata.supersedes || []).includes(document.metadata.id)) errors.push(`${document.path}: superseded_by relationship with ${document.metadata.superseded_by} is not reciprocal`);
    }
  }

  if (manifest) {
    if (manifest.schema_version !== 1) errors.push("manifest.yaml: schema_version must be 1");
    if (!Number.isInteger(manifest.policies.default_review_days) || manifest.policies.default_review_days < 1) errors.push("manifest.yaml: invalid default_review_days");
    const requiredPolicies = ["default_review_days", "allowed_statuses", "direct_update_kinds", "approval_required_kinds"];
    for (const key of requiredPolicies) if (!(key in manifest.policies)) errors.push(`manifest.yaml: missing policy ${key}`);
    for (const key of Object.keys(manifest.policies)) if (!requiredPolicies.includes(key)) errors.push(`manifest.yaml: unknown policy ${key}`);
    for (const [key, allowed] of [["allowed_statuses", STATUSES], ["direct_update_kinds", KINDS], ["approval_required_kinds", KINDS]]) {
      const values = manifest.policies[key];
      if (!Array.isArray(values) || values.length === 0) errors.push(`manifest.yaml: ${key} must be a non-empty list`);
      else {
        if (new Set(values).size !== values.length) errors.push(`manifest.yaml: ${key} must contain unique items`);
        for (const value of values) if (!allowed.has(value)) errors.push(`manifest.yaml: invalid ${key} item ${value}`);
      }
    }
    if (!Array.isArray(manifest.global_triggers) || manifest.global_triggers.length === 0) errors.push("manifest.yaml: global_triggers must be non-empty");
    else validateManifestList(manifest.global_triggers, "manifest.yaml: global_triggers", errors, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    for (const [index, route] of manifest.routes.entries()) {
      validateManifestList(route.match, `manifest route ${index}: match`, errors);
      validateManifestList(route.documents, `manifest route ${index}: documents`, errors, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
      validateManifestList(route.triggers, `manifest route ${index}: triggers`, errors, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
      for (const pattern of route.match || []) try { compileGlob(pattern); } catch (error) { errors.push(`manifest route ${index}: ${error.message}`); }
      for (const id of route.documents || []) if (!byId.has(id)) warnings.push(`manifest route ${index}: document ${id} is not created yet`);
    }
  }
  return { root: repoRoot, errors: [...new Set(errors)].sort(), warnings: [...new Set(warnings)].sort(), documents: documents.sort((a, b) => a.path.localeCompare(b.path)), manifest };
}

function parseArgs(argv) {
  const args = { root: process.cwd(), json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--root") args.root = path.resolve(argv[++index]);
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = scanKnowledge(args.root);
    const output = { schema_version: 1, knowledge_root: ".knowledge", documents: result.documents.map((entry) => ({ path: entry.path, id: entry.metadata.id, status: entry.metadata.status })), warnings: result.warnings, errors: result.errors };
    if (args.json) console.log(JSON.stringify(output, null, 2));
    else {
      for (const warning of result.warnings) console.log(`warning: ${warning}`);
      for (const error of result.errors) console.error(`error: ${error}`);
      console.log(`knowledge_result: ${result.errors.length ? "FAIL" : "PASS"} (${result.documents.length} formal documents)`);
    }
    process.exit(result.errors.length ? 1 : 0);
  } catch (error) { console.error(`error: ${error.message}`); process.exit(2); }
}
