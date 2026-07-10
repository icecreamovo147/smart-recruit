#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { scanKnowledge } from "./validate-knowledge.mjs";

function parseArgs(argv) {
  const args = { root: process.cwd(), output: null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--root") args.root = path.resolve(argv[++index]);
    else if (argv[index] === "--output") args.output = path.resolve(argv[++index]);
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  args.output ||= path.join(args.root, ".knowledge", "generated", "catalog.md");
  return args;
}

export function generateCatalog(root) {
  const result = scanKnowledge(root);
  if (result.errors.length) throw new Error(`knowledge validation failed: ${result.errors.join("; ")}`);
  const entries = result.documents.map((entry) => ({ id: entry.metadata.id, title: entry.metadata.title, kind: entry.metadata.kind, status: entry.metadata.status, path: entry.path })).sort((a, b) => a.id.localeCompare(b.id));
  const rows = entries.map((entry) => `| ${entry.id} | ${entry.kind} | ${entry.status} | \`${entry.path}\` |`);
  const markdown = ["# Generated Knowledge Catalog", "", "Do not edit. Generated from formal document frontmatter.", "", "| ID | Kind | Status | Path |", "|---|---|---|---|", ...rows, ""].join("\n");
  return { entries, markdown };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  let args;
  try { args = parseArgs(process.argv.slice(2)); }
  catch (error) { console.error(`error: ${error.message}`); process.exit(2); }
  try {
    const result = generateCatalog(args.root);
    fs.mkdirSync(path.dirname(args.output), { recursive: true });
    fs.writeFileSync(args.output, result.markdown, "utf8");
    const jsonPath = path.join(path.dirname(args.output), "catalog.json");
    fs.writeFileSync(jsonPath, `${JSON.stringify({ schema_version: 1, entries: result.entries }, null, 2)}\n`, "utf8");
    const relativeMarkdown = path.relative(args.root, args.output).split(path.sep).join("/");
    const relativeJson = path.relative(args.root, jsonPath).split(path.sep).join("/");
    if (args.json) console.log(JSON.stringify({ schema_version: 1, entries: result.entries, markdown: relativeMarkdown, json: relativeJson }, null, 2));
    else {
      console.log(`catalog_result: PASS (${result.entries.length} entries)`);
      console.log(`markdown: ${relativeMarkdown}`);
      console.log(`json: ${relativeJson}`);
    }
  } catch (error) { console.error(`error: ${error.message}`); process.exit(1); }
}
