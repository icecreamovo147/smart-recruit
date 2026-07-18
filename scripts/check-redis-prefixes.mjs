#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const exceptionPath = process.argv[2] || "scripts/redis-prefix-exceptions.json";
const exceptions = loadExceptions(exceptionPath);
const exceptionEntries = [...exceptions.entries()];
const roots = ["smart-recruit-platform-go", "smart-recruit-commons", "smart-recruit-gateway", "smart-recruit-identity-service", "smart-recruit-recruitment-service", "smart-recruit-interview-service", "smart-recruit-offer-service", "smart-recruit-notification-service", "smart-recruit-ai-agent-service", "smart-recruit-analytics-service", "smart-recruit-worker-service"];
const extraRoots = (process.env.REDIS_PREFIX_EXTRA_ROOTS || "").split(path.delimiter).map((root) => root.trim()).filter(Boolean);
const issues = [];

for (const file of listGoFiles([...roots, ...extraRoots])) {
  const relFile = path.relative(process.cwd(), file);
  const text = fs.readFileSync(file, "utf8");
  if (file.includes("/vendor/") || file.includes("/recruitment/pb/")) continue;
  if (file.endsWith("_test.go") && !exceptions.has(relFile)) continue;

  const assignments = collectKeyAssignments(text);
  const operations = collectRedisOperations(text);
  for (const op of operations) {
    const keyArg = redisKeyArgument(op);
    if (!keyArg) continue;
    const normalized = keyArg.replace(/\s+/g, " ").trim();
    const assignment = assignments.get(normalized);
    const evidence = assignment || normalized;
    if (usesRedisKeyHelper(evidence)) continue;
    if (isAllowedException(relFile, evidence)) continue;
    issues.push(`${relFile}:${op.line}: ${op.name} uses unprefixed or unverifiable Redis key ${normalized}`);
  }
}

if (issues.length > 0) {
  for (const issue of issues) console.error(`redis_prefix: ${issue}`);
  process.exit(1);
}

console.log(`redis_prefix: PASS (${roots.length} roots, ${exceptions.size} exception files)`);

function loadExceptions(file) {
  const data = JSON.parse(fs.readFileSync(file, "utf8"));
  const byFile = new Map();
  for (const entry of data.exceptions || []) {
    byFile.set(entry.file, entry.patterns || []);
  }
  return byFile;
}

function listGoFiles(scanRoots) {
  const files = [];
  for (const root of scanRoots) {
    if (!fs.existsSync(root)) continue;
    walk(root, files);
  }
  return files.filter((file) => file.endsWith(".go"));
}

function walk(current, files) {
  const stat = fs.statSync(current);
  if (stat.isDirectory()) {
    for (const child of fs.readdirSync(current)) walk(path.join(current, child), files);
    return;
  }
  files.push(current);
}

function collectKeyAssignments(text) {
  const assignments = new Map();
  const assignmentRe = /^\s*([A-Za-z][A-Za-z0-9_]*(?:Key|Channel)?)\s*:=\s*(fmt\.Sprintf\([^)]*\)|rediskey\.[^(]+\(.*\)|[A-Za-z][A-Za-z0-9_]*\.Key\(.*\)|[A-Za-z][A-Za-z0-9_]*\.Channel\(.*\)|"[^"]*")/gm;
  for (const match of text.matchAll(assignmentRe)) {
    assignments.set(match[1], match[2]);
  }
  return assignments;
}

function collectRedisOperations(text) {
  const operations = [];
  const opRe = /\b([A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)?)\.((?:Get|Set|Del|Exists|Incr|Expire|Publish|Subscribe|PSubscribe|Eval|Run))\s*\(/g;
  for (const match of text.matchAll(opRe)) {
    const start = match.index;
    const open = start + match[0].length - 1;
    const args = readCallArgs(text, open);
    if (!args) continue;
    const receiver = match[1];
    const name = match[2];
    if (!isRedisReceiver(receiver, name)) continue;
    operations.push({receiver, name, args, line: lineNumber(text, start)});
  }
  return operations;
}

function readCallArgs(text, openIndex) {
  let depth = 0;
  let quote = "";
  for (let i = openIndex; i < text.length; i++) {
    const char = text[i];
    const prev = text[i - 1];
    if (quote) {
      if (char === quote && prev !== "\\") quote = "";
      continue;
    }
    if (char === '"' || char === "'" || char === "`") {
      quote = char;
      continue;
    }
    if (char === "(") depth++;
    if (char === ")") {
      depth--;
      if (depth === 0) return text.slice(openIndex + 1, i);
    }
  }
  return "";
}

function redisKeyArgument(op) {
  if (op.name === "Run") {
    const match = op.args.match(/\[\]string\s*\{\s*([^}]+)\s*\}/);
    return match ? firstArg(match[1]) : "";
  }
  const args = splitArgs(op.args);
  if (args.length < 2) return "";
  return args[1];
}

function splitArgs(raw) {
  const args = [];
  let start = 0;
  let depth = 0;
  let quote = "";
  for (let i = 0; i < raw.length; i++) {
    const char = raw[i];
    const prev = raw[i - 1];
    if (quote) {
      if (char === quote && prev !== "\\") quote = "";
      continue;
    }
    if (char === '"' || char === "'" || char === "`") {
      quote = char;
      continue;
    }
    if (char === "(" || char === "{" || char === "[") depth++;
    if (char === ")" || char === "}" || char === "]") depth--;
    if (char === "," && depth === 0) {
      args.push(raw.slice(start, i).trim());
      start = i + 1;
    }
  }
  args.push(raw.slice(start).trim());
  return args;
}

function firstArg(raw) {
  return splitArgs(raw)[0] || "";
}

function usesRedisKeyHelper(expr) {
  return /\brediskey\./.test(expr) || /\.(Key|Channel)\s*\(/.test(expr);
}

function isAllowedException(file, evidence) {
  const entry = exceptionEntries.find(([exceptionFile]) => file === exceptionFile || file.endsWith(exceptionFile) || exceptionFile.endsWith(file));
  const patterns = entry?.[1];
  if (!patterns) return false;
  return patterns.some((pattern) => evidence.includes(pattern));
}

function lineNumber(text, index) {
  return text.slice(0, index).split("\n").length;
}

function isRedisReceiver(receiver, name) {
  if (name === "Run") return /(^|[.])script$/.test(receiver);
  if (name === "Subscribe" || name === "PSubscribe") return /(^|[.])rdb$/.test(receiver);
  return /(^|[.])(rdb|redisClient|Rdb)$/.test(receiver);
}
