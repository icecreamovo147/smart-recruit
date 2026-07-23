#!/usr/bin/env node
import fs from "node:fs";
import process from "node:process";

const schemaPath = process.argv[2] || "db.sql";
const classificationPath = process.argv[3] || "smart-recruit-deploy/mysql-tenant-classification.json";

const schema = fs.readFileSync(schemaPath, "utf8");
const manifest = JSON.parse(fs.readFileSync(classificationPath, "utf8"));
const schemaTables = new Set(
  [...schema.matchAll(/CREATE TABLE IF NOT EXISTS\s+`?([a-zA-Z0-9_]+)`?/gi)].map((match) => match[1]),
);
const classifiedTables = new Set(Object.keys(manifest.tables || {}));
const allowedScopes = new Set(manifest.allowedScopes || []);
const tenantColumnExceptions = manifest.tenantColumnExceptions || {};
const issues = [];

function hasColumn(table, column) {
  const escapedTable = table.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const escapedColumn = column.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const create = new RegExp(
    `CREATE TABLE IF NOT EXISTS\\s+\`?${escapedTable}\`?\\s*\\(([\\s\\S]*?)\\)\\s*ENGINE`,
    "i",
  ).exec(schema)?.[1] || "";
  if (new RegExp(`\`?${escapedColumn}\`?\\s+`, "i").test(create)) return true;
  return new RegExp(
    `ALTER TABLE\\s+\`?${escapedTable}\`?[\\s\\S]*?ADD COLUMN\\s+\`?${escapedColumn}\`?\\s+`,
    "i",
  ).test(schema);
}

if (manifest.schemaVersion !== 1) issues.push("schemaVersion must be 1");
if (manifest.isolationModel !== "shared-database-shared-schema") {
  issues.push("isolationModel must be shared-database-shared-schema");
}
for (const expected of ["global", "tenant_required", "mixed_scope"]) {
  if (!allowedScopes.has(expected)) issues.push(`allowedScopes is missing ${expected}`);
}
for (const table of [...schemaTables].sort()) {
  if (!classifiedTables.has(table)) issues.push(`table ${table} is missing from tenant classification`);
}
for (const table of [...classifiedTables].sort()) {
  const entry = manifest.tables[table];
  if (!schemaTables.has(table)) issues.push(`classified table ${table} is missing from db.sql`);
  if (!entry || !allowedScopes.has(entry.scope)) issues.push(`table ${table} has invalid scope ${entry?.scope}`);
  if (!entry?.reason || entry.reason.trim().length < 12) issues.push(`table ${table} must include a useful reason`);
  if (entry?.scope !== "global" && !hasColumn(table, "tenant_id") && !tenantColumnExceptions[table]) {
    issues.push(`table ${table} requires tenant_id or an explicit tenantColumnExceptions entry`);
  }
}
for (const [table, reason] of Object.entries(tenantColumnExceptions)) {
  if (!classifiedTables.has(table)) issues.push(`tenant column exception ${table} is not classified`);
  if (!reason || reason.trim().length < 20) issues.push(`tenant column exception ${table} must explain its enforcement`);
}

if (issues.length > 0) {
  for (const issue of issues) console.error(`tenant_classification: ${issue}`);
  process.exit(1);
}

const counts = {};
for (const entry of Object.values(manifest.tables)) counts[entry.scope] = (counts[entry.scope] || 0) + 1;
console.log(
  `tenant_classification: PASS (${classifiedTables.size} tables; ` +
    [...allowedScopes].map((scope) => `${scope}=${counts[scope] || 0}`).join(", ") +
    ")",
);
