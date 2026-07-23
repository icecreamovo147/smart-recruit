#!/usr/bin/env node

import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const migrationsDir = path.join(root, "smart-recruit-commons", "migrations");
const lockPath = path.join(migrationsDir, "baseline-lock.json");
const lock = JSON.parse(await readFile(lockPath, "utf8"));

function fail(message) {
  console.error(`migration baseline check failed: ${message}`);
  process.exitCode = 1;
}

function sha256(data) {
  return createHash("sha256").update(data).digest("hex");
}

const baselinePath = path.join(migrationsDir, lock.baseline_file);
const baselineData = await readFile(baselinePath);
if (sha256(baselineData) !== lock.baseline_sha256) {
  fail(`${lock.baseline_file} is immutable; checksum differs from baseline-lock.json`);
}

const activeEntries = await readdir(migrationsDir, { withFileTypes: true });
const activeSQL = activeEntries
  .filter((entry) => entry.isFile() && /^\d{6}_.+\.sql$/.test(entry.name))
  .map((entry) => entry.name)
  .sort();

for (const name of activeSQL) {
  const version = Number(name.slice(0, 6));
  if (version < lock.baseline_version) {
    fail(`historical migration ${name} must remain in ${lock.archive_directory}`);
  }
}
if (!activeSQL.includes(lock.baseline_file)) {
  fail(`active baseline ${lock.baseline_file} is missing`);
}
if (activeSQL.includes(lock.baseline_file.replace(/\.sql$/, ".down.sql"))) {
  fail(`baseline ${lock.baseline_file} must not have a destructive down migration`);
}

const archivePath = path.join(migrationsDir, lock.archive_directory);
const archiveEntries = (await readdir(archivePath, { withFileTypes: true }))
  .filter((entry) => entry.isFile() && /^\d{6}_.+\.sql$/.test(entry.name))
  .map((entry) => entry.name)
  .sort();
const archivedUp = archiveEntries.filter((name) => !name.endsWith(".down.sql"));
const archivedDown = archiveEntries.filter((name) => name.endsWith(".down.sql"));

if (archivedUp.length !== lock.archive_up_files) {
  fail(`archive has ${archivedUp.length} up migrations; expected ${lock.archive_up_files}`);
}
if (archivedDown.length !== lock.archive_down_files) {
  fail(`archive has ${archivedDown.length} down migrations; expected ${lock.archive_down_files}`);
}
for (let version = 1; version <= lock.archived_through; version += 1) {
  const prefix = String(version).padStart(6, "0") + "_";
  if (!archivedUp.some((name) => name.startsWith(prefix))) {
    fail(`archive is missing up migration ${String(version).padStart(6, "0")}`);
  }
  if (!archivedDown.some((name) => name.startsWith(prefix))) {
    fail(`archive is missing down migration ${String(version).padStart(6, "0")}`);
  }
}

const archiveHash = createHash("sha256");
for (const name of archiveEntries) {
  archiveHash.update(name);
  archiveHash.update("\0");
  archiveHash.update(await readFile(path.join(archivePath, name)));
  archiveHash.update("\0");
}
if (archiveHash.digest("hex") !== lock.archive_sha256) {
  fail(`historical archive ${lock.archive_directory} was modified`);
}

if (!process.exitCode) {
  console.log(
    `migration baseline v${lock.baseline_version} verified ` +
      `(${archivedUp.length} archived up / ${archivedDown.length} archived down)`,
  );
}
