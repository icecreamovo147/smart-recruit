#!/usr/bin/env node

import fs from "node:fs";
import process from "node:process";
import { isObject, validateStringArray } from "./contract-utils.mjs";

const STATUSES = new Set(["proposed", "reviewed", "approved", "applied", "rejected"]);
const CLASSIFICATIONS = new Set(["contract_gap", "requirement_conflict", "scope_correction", "baseline_change", "approved_behavior_change"]);
const LEVELS = new Set(["L0", "L1", "L2"]);

export function validateAmendment(amendment) {
  const issues = [];
  if (!isObject(amendment)) return { valid: false, issues: ["amendment must be an object"] };
  if (amendment.schemaVersion !== 2) issues.push("amendment schemaVersion must be 2");
  if (!/^CR-[0-9]{4,}$/.test(amendment.changeRequestId || "")) issues.push("changeRequestId must match CR-0001");
  if (typeof amendment.feature_name !== "string" || amendment.feature_name.length === 0) issues.push("feature_name is required");
  if (!Number.isInteger(amendment.baseRevision) || amendment.baseRevision < 1) issues.push("baseRevision must be a positive integer");
  if (!STATUSES.has(amendment.status)) issues.push(`invalid amendment status: ${amendment.status}`);
  if (!CLASSIFICATIONS.has(amendment.classification)) issues.push(`invalid amendment classification: ${amendment.classification}`);
  if (!LEVELS.has(amendment.approvalLevel)) issues.push(`invalid approvalLevel: ${amendment.approvalLevel}`);
  if (typeof amendment.reason !== "string" || amendment.reason.length === 0) issues.push("reason is required");
  validateStringArray(amendment.evidenceRefs, "evidenceRefs", issues, { nonEmpty: true });
  if (!isObject(amendment.trigger) || typeof amendment.trigger.task_id !== "string" || typeof amendment.trigger.phase !== "string") {
    issues.push("trigger.task_id and trigger.phase are required");
  }
  if (!isObject(amendment.impact)) issues.push("impact is required");
  else {
    for (const field of ["requirements", "tasks", "completedTasksRequiringRevalidation"]) {
      validateStringArray(amendment.impact[field], `impact.${field}`, issues);
    }
  }
  validateStringArray(amendment.revalidationPlan, "revalidationPlan", issues);
  if ("updates" in amendment) {
    if (!Array.isArray(amendment.updates) || amendment.updates.length === 0) issues.push("updates must be a non-empty array when present");
    for (const [index, update] of (amendment.updates || []).entries()) {
      if (!isObject(update)) {
        issues.push(`updates[${index}] must be an object`);
        continue;
      }
      if (typeof update.target !== "string" || update.target.length === 0) issues.push(`updates[${index}].target is required`);
      if (typeof update.staged_file !== "string" || update.staged_file.length === 0) issues.push(`updates[${index}].staged_file is required`);
      if (!(update.before_sha256 === null || /^[a-f0-9]{64}$/.test(update.before_sha256 || ""))) issues.push(`updates[${index}].before_sha256 must be null or SHA-256`);
      if (!/^[a-f0-9]{64}$/.test(update.after_sha256 || "")) issues.push(`updates[${index}].after_sha256 must be SHA-256`);
    }
  }
  if (new Set(["approved", "applied"]).has(amendment.status)) {
    if (!Number.isInteger(amendment.resultingRevision) || amendment.resultingRevision !== amendment.baseRevision + 1) {
      issues.push("approved/applied amendment resultingRevision must equal baseRevision + 1");
    }
    if (!isObject(amendment.approval) || !amendment.approval.approved_by || !amendment.approval.approved_at) {
      issues.push("approved/applied amendment requires approval metadata");
    }
    if (amendment.approvalLevel === "L2" && amendment.approval?.approved_by !== "user") {
      issues.push("L2 amendment must be approved by user");
    }
  }
  return { valid: issues.length === 0, issues };
}

function parseArgs(argv) {
  const args = { file: null, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === "--file") args.file = argv[++index];
    else if (argv[index] === "--json") args.json = true;
    else throw new Error(`unknown argument: ${argv[index]}`);
  }
  if (!args.file) throw new Error("--file is required");
  return args;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const args = parseArgs(process.argv.slice(2));
    const result = validateAmendment(JSON.parse(fs.readFileSync(args.file, "utf8")));
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      for (const issue of result.issues) console.error(`error: ${issue}`);
      console.log(`amendment_result: ${result.valid ? "PASS" : "FAIL"}`);
    }
    process.exit(result.valid ? 0 : 1);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
