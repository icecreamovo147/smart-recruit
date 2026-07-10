#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { normalizeRepoPath, scanKnowledge } from "./validate-knowledge.mjs";

function parseArgs(argv) {
  const args = { root: process.cwd(), json: false, strictRoutes: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--root") args.root = path.resolve(argv[++index]);
    else if (argv[index] === "--json") args.json = true;
    else if (argv[index] === "--strict-routes") args.strictRoutes = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  return args;
}

export function checkReferences(root, options = {}) {
  const result = scanKnowledge(root);
  const knowledgeRoot = path.join(path.resolve(root), ".knowledge");
  const errors = [...result.errors];
  const warnings = [...result.warnings];
  const byId = new Set(result.documents.map((entry) => entry.metadata.id));
  const indexPath = path.join(knowledgeRoot, "INDEX.md");
  if (!fs.existsSync(indexPath)) errors.push("missing .knowledge/INDEX.md");
  else {
    const content = fs.readFileSync(indexPath, "utf8");
    for (const match of content.matchAll(/`((?:architecture|domains|decisions|runbooks|pitfalls|inbox|archive)\/[^`]+\.md)`/g)) {
      let message = null;
      try {
        const normalized = normalizeRepoPath(match[1]);
        if (!fs.existsSync(path.join(knowledgeRoot, normalized))) message = `INDEX.md: target is not created yet: ${match[1]}`;
      } catch (error) { message = `INDEX.md: invalid target ${match[1]}: ${error.message}`; }
      if (message) { if (options.strictRoutes) errors.push(message); else warnings.push(message); }
    }
  }
  for (const [index, route] of (result.manifest?.routes || []).entries()) {
    for (const id of route.documents) {
      if (!byId.has(id)) {
        const message = `manifest route ${index}: unknown document id ${id}`;
        if (options.strictRoutes) errors.push(message); else warnings.push(message);
      }
    }
  }
  return { errors: [...new Set(errors)].sort(), warnings: [...new Set(warnings)].sort() };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = checkReferences(args.root, { strictRoutes: args.strictRoutes });
    if (args.json) console.log(JSON.stringify({ schema_version: 1, ...result }, null, 2));
    else {
      for (const warning of result.warnings) console.log(`warning: ${warning}`);
      for (const error of result.errors) console.error(`error: ${error}`);
      console.log(`reference_result: ${result.errors.length ? "FAIL" : "PASS"}`);
    }
    process.exit(result.errors.length ? 1 : 0);
  } catch (error) { console.error(`error: ${error.message}`); process.exit(2); }
}
