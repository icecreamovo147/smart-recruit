#!/usr/bin/env node

import fs from "node:fs";
import process from "node:process";
import { isObject, validateStringArray } from "./contract-utils.mjs";

const STATUSES = new Set(["unmapped", "planned", "implemented", "verified", "blocked", "approved_delta"]);

export function validateTraceability(traceability, options = {}) {
  const issues = [];
  const ids = new Set();
  const checkIds = new Set();
  if (!isObject(traceability)) return { valid: false, issues: ["traceability.json must be an object"], check_ids: [] };
  if (traceability.schemaVersion !== 2) issues.push("traceability.json schemaVersion must be 2");
  if (typeof traceability.feature_name !== "string" || traceability.feature_name.length === 0) issues.push("traceability feature_name is required");
  if (!Number.isInteger(traceability.contractRevision) || traceability.contractRevision < 1) issues.push("traceability contractRevision must be a positive integer");
  if (!Array.isArray(traceability.requirements) || traceability.requirements.length === 0) issues.push("traceability requirements must be a non-empty array");

  for (const [index, requirement] of (traceability.requirements || []).entries()) {
    const label = `requirements[${index}]`;
    if (!isObject(requirement)) {
      issues.push(`${label} must be an object`);
      continue;
    }
    if (typeof requirement.id !== "string" || requirement.id.length === 0) issues.push(`${label}.id is required`);
    else if (ids.has(requirement.id)) issues.push(`duplicate requirement id: ${requirement.id}`);
    else ids.add(requirement.id);
    if (typeof requirement.mandatory !== "boolean") issues.push(`${label}.mandatory must be boolean`);
    if (!STATUSES.has(requirement.status)) issues.push(`${label}.status is invalid`);
    for (const field of ["design_refs", "task_ids", "acceptance_ids", "check_ids", "evidence_refs"]) {
      validateStringArray(requirement[field], `${label}.${field}`, issues, {
        nonEmpty: requirement.mandatory === true && field !== "evidence_refs",
      });
    }
    for (const checkId of requirement.check_ids || []) checkIds.add(checkId);
    if (requirement.status === "approved_delta" && (typeof requirement.approval_ref !== "string" || requirement.approval_ref.length === 0)) {
      issues.push(`${label} approved_delta requires approval_ref`);
    }
    if (options.requireVerified && requirement.mandatory) {
      if (!new Set(["verified", "approved_delta"]).has(requirement.status)) issues.push(`${label} mandatory requirement is not verified`);
      if (!Array.isArray(requirement.evidence_refs) || requirement.evidence_refs.length === 0) issues.push(`${label}.evidence_refs must not be empty at completion`);
    }
  }
  return { valid: issues.length === 0, issues, check_ids: [...checkIds].sort(), requirement_ids: [...ids].sort() };
}

function parseArgs(argv) {
  const args = { file: null, requireVerified: false, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--file") args.file = argv[++index];
    else if (argv[index] === "--require-verified") args.requireVerified = true;
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!args.file) throw new Error("--file is required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = validateTraceability(JSON.parse(fs.readFileSync(args.file, "utf8")), args);
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      for (const issue of result.issues) console.error(`error: ${issue}`);
      console.log(`traceability_result: ${result.valid ? "PASS" : "FAIL"}`);
    }
    process.exit(result.valid ? 0 : 1);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
