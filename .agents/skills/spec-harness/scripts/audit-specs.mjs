#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { validateFeature } from "./validate-feature.mjs";

function parseArgs(argv) {
  const args = { root: ".spec", strict: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--root") args.root = argv[++index];
    else if (argv[index] === "--strict") args.strict = true;
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  return args;
}

try {
  const args = parseArgs(process.argv.slice(2));
  const root = path.resolve(args.root);
  const before = fs.readdirSync(root).sort();
  const results = before
    .filter((name) => fs.statSync(path.join(root, name)).isDirectory())
    .map((name) => validateFeature(path.join(root, name), { requirePipeline: false }));
  const summary = results.reduce((value, item) => {
    value[item.classification] = (value[item.classification] || 0) + 1;
    return value;
  }, {});
  if (args.json) console.log(JSON.stringify({ root, summary, results }, null, 2));
  else {
    for (const result of results) {
      console.log(`${result.classification}\t${result.feature_name}`);
      for (const issue of result.issues) console.log(`  error: ${issue}`);
      for (const warning of result.warnings) console.log(`  warning: ${warning}`);
    }
    console.log(`summary: ${JSON.stringify(summary)}`);
  }
  const after = fs.readdirSync(root).sort();
  if (JSON.stringify(before) !== JSON.stringify(after)) throw new Error("audit modified the .spec directory entries");
  const unsupported = results.filter((result) => result.classification === "unsupported").length;
  process.exit(args.strict && unsupported > 0 ? 1 : 0);
} catch (error) {
  console.error(`error: ${error.message}`);
  process.exit(2);
}
