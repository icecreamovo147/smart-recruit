#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const manifestPath = process.argv[2] || "smart-recruit-deploy/mysql-table-ownership.json";
const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const schema = fs.readFileSync("db.sql", "utf8");
const dbTables = [...schema.matchAll(/CREATE TABLE IF NOT EXISTS\s+`?([a-zA-Z0-9_]+)`?/gi)].map((m) => m[1]).sort();
const declared = Object.keys(manifest.tables || {}).sort();
const services = new Set(manifest.services || []);
const issues = [];
const validModes = new Set(["read", "write"]);

if (manifest.single_mysql_instance !== true) {
  issues.push("manifest must declare single_mysql_instance=true");
}
if (services.size === 0) {
  issues.push("manifest must declare services");
}
for (const table of dbTables) {
  if (!manifest.tables[table]) issues.push(`table ${table} is present in db.sql but missing from ownership manifest`);
}
for (const table of declared) {
  if (!dbTables.includes(table)) issues.push(`table ${table} is declared in manifest but missing from db.sql`);
  const entry = manifest.tables[table];
  if (!entry.owner) issues.push(`table ${table} has no owner`);
  if (entry.owner && !services.has(entry.owner)) issues.push(`table ${table} owner ${entry.owner} is not listed in services`);
  if (!entry.access || !entry.access[entry.owner] || !entry.access[entry.owner].includes("write")) {
    issues.push(`table ${table} owner ${entry.owner} must have write access`);
  }
  for (const [service, modes] of Object.entries(entry.access || {})) {
    if (!services.has(service)) issues.push(`table ${table} grants access to unknown service ${service}`);
    for (const mode of modes || []) {
      if (!validModes.has(mode)) issues.push(`table ${table} grants invalid access mode ${mode} to ${service}`);
    }
  }
}

const scanRoots = ["smart-recruit-domain-go/repository", "smart-recruit-domain-go/service", "smart-recruit-analytics-service", "smart-recruit-worker-service"];
for (const file of listFiles(scanRoots)) {
  const owner = inferOwner(file);
  if (!owner) continue;
  const text = fs.readFileSync(file, "utf8");
  for (const match of text.matchAll(/\.Table\(\s*["'`]([^"'`]+)["'`]\s*\)/g)) {
    const table = baseTableName(match[1]);
    const entry = manifest.tables[table];
    if (!entry) {
      issues.push(`${file}: table ${table} is used but missing from manifest`);
      continue;
    }
    const access = entry.access?.[owner] || [];
    if (access.length === 0) {
      issues.push(`${file}: ${owner} uses table ${table} without declared access`);
    }
    const operation = inferOperation(text, match.index);
    if (operation === "write" && !access.includes("write")) {
      issues.push(`${file}: ${owner} writes table ${table} without declared write access`);
    }
  }
}

if (issues.length > 0) {
  for (const issue of issues) console.error(`mysql_table_ownership: ${issue}`);
  process.exit(1);
}
console.log(`mysql_table_ownership: PASS (${declared.length} tables, single MySQL instance)`);

function listFiles(roots) {
  const files = [];
  for (const root of roots) {
    if (!fs.existsSync(root)) continue;
    walk(root, files);
  }
  return files.filter((file) => file.endsWith(".go") && !file.endsWith("_test.go"));
}

function walk(current, files) {
  const stat = fs.statSync(current);
  if (stat.isDirectory()) {
    for (const child of fs.readdirSync(current)) walk(path.join(current, child), files);
    return;
  }
  files.push(current);
}

function inferOwner(file) {
  if (file.includes("analytics_repo") || file.includes("smart-recruit-analytics-service")) return "analytics";
  if (file.includes("authz_repo") || file.includes("user_repo") || file.includes("refresh")) return "identity";
  if (file.includes("interview_repo")) return "interview";
  if (file.includes("offer_repo")) return "offer";
  if (file.includes("notification")) return "notification";
  if (file.includes("smart-recruit-worker-service")) return "worker";
  if (file.includes("application_repo") || file.includes("job") || file.includes("resume") || file.includes("candidate") || file.includes("collaboration")) return "recruitment";
  if (file.includes("ai") || file.includes("agent") || file.includes("embedding") || file.includes("prompt") || file.includes("mcp")) return "ai-agent";
  return null;
}

function baseTableName(raw) {
  return raw.trim().replace(/`/g, "").split(/\s+/)[0];
}

function inferOperation(text, tableCallIndex) {
  const chain = text.slice(tableCallIndex, tableCallIndex + 700);
  if (/\.(Create|Save|Updates?|UpdateColumn|UpdateColumns|Delete|Exec)\s*\(/.test(chain)) return "write";
  return "read";
}
