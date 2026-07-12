#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const repoRoot = process.cwd();
const schemaPath = path.join(repoRoot, "db.sql");
const manifestPath = path.join(repoRoot, ".spec/backend-ddd-microservices-evolution/docs/backend-ddd-microservices-evolution-table-ownership-manifest.json");

function loadJson(file) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    throw new Error(`failed to read JSON ${path.relative(repoRoot, file)}: ${error.message}`);
  }
}

function extractTables(sql) {
  const tables = [];
  const seen = new Set();
  const pattern = /CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?`?([A-Za-z0-9_]+)`?/gi;
  for (const match of sql.matchAll(pattern)) {
    const table = match[1];
    if (!seen.has(table)) {
      seen.add(table);
      tables.push(table);
    }
  }
  return tables.sort();
}

function validateStringArray(errors, table, field, value) {
  if (!Array.isArray(value) || value.length === 0) {
    errors.push(`${table}: ${field} must be a non-empty array`);
    return;
  }
  const seen = new Set();
  for (const item of value) {
    if (typeof item !== "string" || item.trim() === "") errors.push(`${table}: ${field} contains an empty item`);
    if (seen.has(item)) errors.push(`${table}: ${field} contains duplicate item ${item}`);
    seen.add(item);
  }
}

function validateTransitionalAccess(errors, table, owner, entries) {
  if (entries === undefined) return;
  if (!Array.isArray(entries)) {
    errors.push(`${table}: transitional_shared_access must be an array`);
    return;
  }
  for (const [index, entry] of entries.entries()) {
    const label = `${table}.transitional_shared_access[${index}]`;
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
      errors.push(`${label}: must be an object`);
      continue;
    }
    for (const field of ["table", "owner", "accessor", "reason", "risk", "removal_plan"]) {
      if (typeof entry[field] !== "string" || entry[field].trim() === "") errors.push(`${label}: missing ${field}`);
    }
    if (entry.table && entry.table !== table) errors.push(`${label}: table must match ${table}`);
    if (entry.owner && entry.owner !== owner) errors.push(`${label}: owner must match table owner ${owner}`);
  }
}

function main() {
  const schema = fs.readFileSync(schemaPath, "utf8");
  const schemaTables = extractTables(schema);
  const manifest = loadJson(manifestPath);
  const errors = [];

  if (manifest.schemaVersion !== 1) errors.push("manifest schemaVersion must be 1");
  if (manifest.feature_name !== "backend-ddd-microservices-evolution") errors.push("manifest feature_name mismatch");
  if (!manifest.tables || typeof manifest.tables !== "object" || Array.isArray(manifest.tables)) {
    errors.push("manifest tables must be an object");
  }

  const manifestTables = Object.keys(manifest.tables || {}).sort();
  const schemaSet = new Set(schemaTables);
  const manifestSet = new Set(manifestTables);

  for (const table of schemaTables) {
    if (!manifestSet.has(table)) errors.push(`missing manifest entry for table ${table}`);
  }
  for (const table of manifestTables) {
    if (!schemaSet.has(table)) errors.push(`manifest entry has no matching db.sql table ${table}`);
  }

  for (const table of manifestTables) {
    const entry = manifest.tables[table];
    if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
      errors.push(`${table}: manifest entry must be an object`);
      continue;
    }
    if (typeof entry.owner !== "string" || entry.owner.trim() === "") errors.push(`${table}: owner must be a non-empty string`);
    validateStringArray(errors, table, "allowed_readers", entry.allowed_readers);
    validateStringArray(errors, table, "allowed_writers", entry.allowed_writers);
    if (entry.owner && Array.isArray(entry.allowed_readers) && !entry.allowed_readers.includes(entry.owner)) {
      errors.push(`${table}: allowed_readers must include owner ${entry.owner}`);
    }
    if (entry.owner && Array.isArray(entry.allowed_writers) && !entry.allowed_writers.includes(entry.owner)) {
      errors.push(`${table}: allowed_writers must include owner ${entry.owner}`);
    }
    validateTransitionalAccess(errors, table, entry.owner, entry.transitional_shared_access);
  }

  if (errors.length) {
    for (const error of errors) console.error(`error: ${error}`);
    console.error(`table_ownership_result: FAIL (${errors.length} errors)`);
    process.exit(1);
  }

  const transitionalCount = Object.values(manifest.tables || {}).reduce(
    (total, entry) => total + (Array.isArray(entry.transitional_shared_access) ? entry.transitional_shared_access.length : 0),
    0,
  );
  console.log(`table_ownership_result: PASS (${schemaTables.length} tables, ${transitionalCount} transitional shared access entries)`);
}

try {
  main();
} catch (error) {
  console.error(`error: ${error.message}`);
  process.exit(2);
}
