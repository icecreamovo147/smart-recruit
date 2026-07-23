#!/usr/bin/env node

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync, spawnSync } from "node:child_process";
import { compileGlob, extractBodyVerificationDate, findCaseConflicts, normalizeRepoPath, parseFrontmatter, parseManifest, scanKnowledge } from "./validate-knowledge.mjs";
import { collectChangedFiles, detectImpact } from "./detect-impact.mjs";
import { checkReferences } from "./check-references.mjs";
import { generateCatalog } from "./generate-catalog.mjs";

let passed = 0;
function test(name, fn) {
  try { fn(); passed += 1; console.log(`ok - ${name}`); }
  catch (error) { console.error(`not ok - ${name}`); throw error; }
}
function write(file, content) { fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, content, "utf8"); }
function git(root, args) { return execFileSync("git", ["-C", root, ...args], { encoding: "utf8" }).trim(); }

const validDoc = (id = "system-overview", source = "README.md") => `---
schema_version: 1
id: ${id}
title: Test document
kind: architecture
status: active
owners:
  - engineering-platform
tags:
  - architecture
applies_to:
  - smart-recruit-recruitment-service/**
source_refs:
  - ${source}
last_verified: 2026-07-10
review_after: 2026-10-10
---

# Test

## Verification

Verified against fixture sources on 2026-07-10.
`;

const manifest = `schema_version: 1
policies:
  default_review_days: 90
  allowed_statuses:
    - active
  direct_update_kinds:
    - runbook
  approval_required_kinds:
    - decision
routes:
  - match:
      - smart-recruit-recruitment-service/internal/application/service/*.go
    documents:
      - system-overview
    triggers:
      - covered-path-changed
global_triggers:
  - new-top-level-module
  - database-schema-changed
  - public-api-changed
  - service-boundary-changed
  - authorization-changed
  - sensitive-data-flow-changed
  - configuration-changed
`;

function fixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "knowledge-test-"));
  write(path.join(root, "README.md"), "# Fixture\n");
  write(path.join(root, ".knowledge", "manifest.yaml"), manifest);
  write(path.join(root, ".knowledge", "INDEX.md"), "# Index\n\n`architecture/system-overview.md`\n");
  write(path.join(root, ".knowledge", "architecture", "system-overview.md"), validDoc());
  return root;
}

test("frontmatter parses required scalar and list fields", () => {
  const parsed = parseFrontmatter(validDoc());
  assert.equal(parsed.data.schema_version, 1);
  assert.deepEqual(parsed.data.owners, ["engineering-platform"]);
});

test("manifest parser handles routes", () => {
  const parsed = parseManifest(manifest);
  assert.equal(parsed.schema_version, 1);
  assert.deepEqual(parsed.routes[0].documents, ["system-overview"]);
});

test("manifest parser rejects duplicate and unknown keys", () => {
  assert.throws(() => parseManifest(`${manifest}\nschema_version: 1\n`), /duplicate key schema_version/);
  assert.throws(() => parseManifest(manifest.replace("  default_review_days: 90", "  default_review_days: 90\n  default_review_days: 30")), /duplicate policy key/);
  const withoutRoutes = manifest.replace(/routes:[\s\S]*?global_triggers:/, "global_triggers:");
  assert.throws(() => parseManifest(withoutRoutes), /missing required top-level key routes/);
  assert.throws(() => parseManifest(`${manifest}\nunknown:\n`), /unknown top-level section/);
  assert.throws(() => parseManifest(manifest.replace("    documents:\n", "    documents:\n    documents:\n")), /duplicate route key documents/);
});

test("manifest validation rejects invalid and duplicate route or trigger items", () => {
  const root = fixture();
  const invalid = manifest
    .replace("      - system-overview", "      - 123\n      - 123")
    .replace("      - covered-path-changed", "      - Bad_Trigger\n      - Bad_Trigger")
    .replace("  - configuration-changed", "  - 456\n  - 456");
  write(path.join(root, ".knowledge", "manifest.yaml"), invalid);
  const errors = scanKnowledge(root).errors.join("\n");
  assert.match(errors, /documents items must be non-empty strings/);
  assert.match(errors, /documents must contain unique items/);
  assert.match(errors, /triggers has invalid item Bad_Trigger/);
  assert.match(errors, /global_triggers items must be non-empty strings/);
});

test("repository paths reject absolute and Windows forms", () => {
  assert.equal(normalizeRepoPath("smart-recruit-recruitment-service/internal/application/service/a.go"), "smart-recruit-recruitment-service/internal/application/service/a.go");
  assert.throws(() => normalizeRepoPath("/Users/test/repo/file"));
  assert.throws(() => normalizeRepoPath("C:\\repo\\file"));
  assert.throws(() => normalizeRepoPath("../outside"));
});

test("glob matching is deterministic", () => {
  const regex = compileGlob("smart-recruit-recruitment-service/internal/application/service/*.go");
  assert.equal(regex.test("smart-recruit-recruitment-service/internal/application/service/a.go"), true);
  assert.equal(regex.test("smart-recruit-recruitment-service/internal/application/service/nested/a.go"), false);
  assert.throws(() => compileGlob("smart-recruit-recruitment-service/[ab].go"), /unsupported character class/);
});

test("valid repository passes scan and reference checks", () => {
  const root = fixture();
  assert.deepEqual(scanKnowledge(root).errors, []);
  assert.deepEqual(checkReferences(root, { strictRoutes: true }).errors, []);
});

test("duplicate IDs and missing sources fail", () => {
  const root = fixture();
  write(path.join(root, ".knowledge", "domains", "duplicate.md"), validDoc("system-overview", "missing.md"));
  const errors = scanKnowledge(root).errors.join("\n");
  assert.match(errors, /duplicate id/);
  assert.match(errors, /missing source_ref/);
});

test("metadata types, uniqueness, and calendar dates fail closed", () => {
  const root = fixture();
  const invalid = validDoc("invalid-meta")
    .replace("title: Test document", "title: ")
    .replace("  - engineering-platform", "  - Bad Owner\n  - Bad Owner")
    .replace("review_after: 2026-10-10", "review_after: 2026-02-30");
  write(path.join(root, ".knowledge", "domains", "invalid.md"), invalid);
  const errors = scanKnowledge(root).errors.join("\n");
  assert.match(errors, /title must be a non-empty string/);
  assert.match(errors, /owners must contain unique items/);
  assert.match(errors, /invalid owners item/);
  assert.match(errors, /invalid review_after/);
});

test("Verification section date must match last_verified when present", () => {
  assert.equal(extractBodyVerificationDate("## Verification\n\nVerified against sources on 2026-07-23.\n"), "2026-07-23");
  const root = fixture();
  const mismatched = validDoc("date-mismatch").replace("on 2026-07-10.", "on 2026-07-09.");
  write(path.join(root, ".knowledge", "domains", "date-mismatch.md"), mismatched);
  const errors = scanKnowledge(root).errors.join("\n");
  assert.match(errors, /Verification section date 2026-07-09 does not match last_verified 2026-07-10/);
});

test("Inbox and archive status constraints fail closed", () => {
  const root = fixture();
  write(path.join(root, ".knowledge", "inbox", "bad.md"), validDoc("bad-inbox"));
  write(path.join(root, ".knowledge", "archive", "bad.md"), validDoc("bad-archive"));
  const errors = scanKnowledge(root).errors.join("\n");
  assert.match(errors, /Inbox documents must be candidate\/draft/);
  assert.match(errors, /Archive document status/);
});

test("case-conflicting knowledge paths are rejected", () => {
  assert.deepEqual(findCaseConflicts(["domains/Case.md", "domains/case.md"]), [["domains/Case.md", "domains/case.md"]]);
  assert.deepEqual(findCaseConflicts(["domains/case.md", "domains/other.md"]), []);
});

test("catalog output is sorted", () => {
  const root = fixture();
  write(path.join(root, ".knowledge", "domains", "alpha.md"), validDoc("alpha"));
  const catalog = generateCatalog(root);
  assert.deepEqual(catalog.entries.map((entry) => entry.id), ["alpha", "system-overview"]);
});

test("ADR relationships require ADR targets and reciprocal links", () => {
  const root = fixture();
  const adr = (id, decisionStatus, supersedes, supersededBy) => validDoc(id)
    .replace("kind: architecture", "kind: decision")
    .replace("status: active", decisionStatus === "superseded" ? "status: archived" : "status: active")
    .replace("review_after: 2026-10-10", `review_after: 2026-10-10\ndecision_status: ${decisionStatus}\nsupersedes:${supersedes.length ? `\n${supersedes.map((value) => `  - ${value}`).join("\n")}` : " []"}\nsuperseded_by: ${supersededBy || "null"}`);
  write(path.join(root, ".knowledge", "decisions", "old.md"), adr("adr-20260101-old", "superseded", [], "adr-20260710-new"));
  write(path.join(root, ".knowledge", "decisions", "new.md"), adr("adr-20260710-new", "accepted", ["adr-20260101-old"], null));
  assert.deepEqual(scanKnowledge(root).errors, []);
  write(path.join(root, ".knowledge", "decisions", "bad.md"), adr("adr-20260710-bad", "accepted", ["system-overview"], null));
  assert.match(scanKnowledge(root).errors.join("\n"), /is not an ADR/);
});

test("strict references reject missing and escaping INDEX targets", () => {
  const root = fixture();
  write(path.join(root, ".knowledge", "INDEX.md"), "`domains/missing.md`\n`domains/../../outside.md`\n");
  const result = checkReferences(root, { strictRoutes: true });
  assert.match(result.errors.join("\n"), /target is not created yet/);
  assert.match(result.errors.join("\n"), /invalid target/);
});

test("impact detection requires a valid baseline and includes tracked and untracked files", () => {
  const root = fixture();
  git(root, ["init", "-q"]);
  git(root, ["config", "user.email", "test@example.com"]);
  git(root, ["config", "user.name", "Knowledge Test"]);
  write(path.join(root, "smart-recruit-recruitment-service", "internal", "application", "service", "existing.go"), "package service\n");
  git(root, ["add", "-A"]);
  git(root, ["commit", "-qm", "fixture"]);
  const base = git(root, ["rev-parse", "HEAD"]);
  write(path.join(root, "smart-recruit-recruitment-service", "internal", "application", "service", "existing.go"), "package service\n// changed\n");
  write(path.join(root, "smart-recruit-recruitment-service", "internal", "application", "service", "new.go"), "package service\n");
  const changed = collectChangedFiles(root, base);
  assert.deepEqual(changed, ["smart-recruit-recruitment-service/internal/application/service/existing.go", "smart-recruit-recruitment-service/internal/application/service/new.go"]);
  const impact = detectImpact({ root, baseTree: base });
  assert.deepEqual(impact.reviewed_documents, ["system-overview"]);
  assert.equal(impact.knowledge_impact.result, "update_required");
  assert.equal(impact.triggered_by.includes("new-top-level-module"), false);
  write(path.join(root, "smart-recruit-gateway", "config", "config.example.yaml"), "mode: test\n");
  const triggerOnly = detectImpact({ root, baseTree: base });
  assert.equal(triggerOnly.triggered_by.includes("configuration-changed"), true);
  assert.notEqual(triggerOnly.knowledge_impact.result, "none");
  write(path.join(root, "db.sql"), "CREATE TABLE example (id INT);\n");
  const schemaImpact = detectImpact({ root, baseTree: base });
  assert.equal(schemaImpact.triggered_by.includes("database-schema-changed"), true);
  assert.equal(schemaImpact.knowledge_impact.result, "coverage_gap");
  write(path.join(root, "brand-new-module", "service", "new.go"), "package service\n");
  const moduleImpact = detectImpact({ root, baseTree: base });
  assert.equal(moduleImpact.triggered_by.includes("new-top-level-module"), true);
  assert.throws(() => collectChangedFiles(root, "missing-tree"), /invalid base tree/);
});

test("impact detection rejects a semantically invalid manifest", () => {
  const root = fixture();
  git(root, ["init", "-q"]);
  git(root, ["config", "user.email", "test@example.com"]);
  git(root, ["config", "user.name", "Knowledge Test"]);
  git(root, ["add", "-A"]);
  git(root, ["commit", "-qm", "fixture"]);
  const base = git(root, ["rev-parse", "HEAD"]);
  write(path.join(root, ".knowledge", "manifest.yaml"), manifest.replace(/routes:[\s\S]*?global_triggers:/, "global_triggers:"));
  assert.throws(() => detectImpact({ root, baseTree: base }), /knowledge validation failed/);
  const script = path.resolve(".knowledge/scripts/detect-impact.mjs");
  const invalidContract = spawnSync(process.execPath, [script, "--root", root, "--base-tree", base], { encoding: "utf8" });
  assert.equal(invalidContract.status, 1);
  const cleanEnv = { ...process.env };
  delete cleanEnv.TASK_BASE_TREE;
  const invalidUsage = spawnSync(process.execPath, [script, "--root", root], { encoding: "utf8", env: cleanEnv });
  assert.equal(invalidUsage.status, 2);
});

test("catalog CLI supports JSON and usage errors return exit code 2", () => {
  const root = fixture();
  const script = path.resolve(".knowledge/scripts/generate-catalog.mjs");
  const json = spawnSync(process.execPath, [script, "--root", root, "--json"], { encoding: "utf8" });
  assert.equal(json.status, 0);
  const output = JSON.parse(json.stdout);
  assert.equal(output.schema_version, 1);
  assert.equal(path.isAbsolute(output.markdown), false);
  const bad = spawnSync(process.execPath, [script, "--bogus"], { encoding: "utf8" });
  assert.equal(bad.status, 2);
});

test("repository Manifest routes cover platform frontend, fingerprint, audit skill, and isolate legacy interviewer", () => {
  const repoRoot = path.resolve(".");
  const scanned = scanKnowledge(repoRoot);
  assert.deepEqual(scanned.errors, []);
  const routes = scanned.manifest.routes.map((route, index) => ({
    index,
    documents: new Set(route.documents),
    patterns: route.match.map((pattern) => ({ pattern, regex: compileGlob(pattern) })),
  }));
  function matchedDocuments(file) {
    const ids = new Set();
    for (const route of routes) {
      if (route.patterns.some(({ regex }) => regex.test(file))) {
        for (const id of route.documents) ids.add(id);
      }
    }
    return ids;
  }
  const platform = matchedDocuments("platform-frontend/src/views/PlanCatalogView.vue");
  assert.equal(platform.has("frontend-apps"), true);
  assert.equal(platform.has("frontend-validation"), true);
  assert.equal(platform.has("frontend-menu-consistency"), true);
  const fingerprint = matchedDocuments("scripts/dev-build-fingerprint.go");
  assert.equal(fingerprint.has("local-development"), true);
  const auditSkill = matchedDocuments(".agents/skills/knowledge-current-state-audit/SKILL.md");
  assert.equal(auditSkill.has("knowledge-coverage-audit"), true);
  const legacy = matchedDocuments("interviewer-frontend/src/stores/auth.ts");
  assert.equal(legacy.has("frontend-apps"), true);
  assert.equal(legacy.has("local-development"), true);
  assert.equal(legacy.has("frontend-validation"), false);
  assert.equal(legacy.has("auth-rbac-security"), false);
});

console.log(`knowledge-validator tests: PASS (${passed})`);
