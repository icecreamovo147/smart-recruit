#!/usr/bin/env node

import fs from "node:fs";
import process from "node:process";
import { isObject, validateStringArray } from "./contract-utils.mjs";

export const PARITY_DIMENSIONS = Object.freeze([
  "transport",
  "fields",
  "state",
  "persistence",
  "side_effects",
  "security",
  "failure_behavior",
  "operational_behavior",
]);
const STATUSES = new Set(["verified_equal", "approved_delta", "missing", "unknown", "blocked", "not_applicable"]);

export function validateParityManifest(manifest, options = {}) {
  const issues = [];
  const capabilityIds = new Set();
  if (!isObject(manifest)) return { valid: false, issues: ["behavior-manifest.json must be an object"] };
  if (manifest.schemaVersion !== 2) issues.push("behavior-manifest.json schemaVersion must be 2");
  if (typeof manifest.feature_name !== "string" || manifest.feature_name.length === 0) issues.push("behavior manifest feature_name is required");
  if (!Number.isInteger(manifest.contractRevision) || manifest.contractRevision < 1) issues.push("behavior manifest contractRevision must be a positive integer");
  if (!isObject(manifest.baseline)) issues.push("behavior manifest baseline is required");
  else {
    if (typeof manifest.baseline.ref !== "string" || manifest.baseline.ref.length === 0) issues.push("behavior manifest baseline.ref is required");
    if (typeof manifest.baseline.sha !== "string" || manifest.baseline.sha.length < 7) issues.push("behavior manifest baseline.sha must be pinned");
    if (typeof manifest.baseline.tree !== "string" || manifest.baseline.tree.length < 7) issues.push("behavior manifest baseline.tree must be pinned");
    if (manifest.baseline.immutable !== true) issues.push("behavior manifest baseline must set immutable=true");
  }
  if (!Array.isArray(manifest.capabilities) || manifest.capabilities.length === 0) issues.push("behavior manifest capabilities must be a non-empty array");

  for (const [index, capability] of (manifest.capabilities || []).entries()) {
    const label = `capabilities[${index}]`;
    if (!isObject(capability)) {
      issues.push(`${label} must be an object`);
      continue;
    }
    if (typeof capability.id !== "string" || capability.id.length === 0) issues.push(`${label}.id is required`);
    else if (capabilityIds.has(capability.id)) issues.push(`duplicate capability id: ${capability.id}`);
    else capabilityIds.add(capability.id);
    if (typeof capability.mandatory !== "boolean") issues.push(`${label}.mandatory must be boolean`);
    if (!isObject(capability.source)) issues.push(`${label}.source is required`);
    else {
      validateStringArray(capability.source.entrypoints, `${label}.source.entrypoints`, issues, { nonEmpty: true });
      validateStringArray(capability.source.implementation_refs, `${label}.source.implementation_refs`, issues, { nonEmpty: true });
      validateStringArray(capability.source.tests, `${label}.source.tests`, issues, { nonEmpty: true });
    }
    if (!isObject(capability.target)) issues.push(`${label}.target is required`);
    else validateStringArray(capability.target.implementation_refs, `${label}.target.implementation_refs`, issues, { nonEmpty: options.requireVerified && capability.mandatory });
    if (!isObject(capability.dimensions)) {
      issues.push(`${label}.dimensions is required`);
      continue;
    }
    for (const dimension of PARITY_DIMENSIONS) {
      const result = capability.dimensions[dimension];
      if (!isObject(result)) {
        issues.push(`${label}.dimensions.${dimension} is required`);
        continue;
      }
      if (!STATUSES.has(result.status)) issues.push(`${label}.dimensions.${dimension}.status is invalid`);
      validateStringArray(result.evidence_refs, `${label}.dimensions.${dimension}.evidence_refs`, issues);
      if (result.status === "approved_delta" && (typeof result.approval_ref !== "string" || result.approval_ref.length === 0)) {
        issues.push(`${label}.dimensions.${dimension} approved_delta requires approval_ref`);
      }
      if (options.requireVerified && capability.mandatory) {
        if (!new Set(["verified_equal", "approved_delta", "not_applicable"]).has(result.status)) {
          issues.push(`${label}.dimensions.${dimension} is not parity verified`);
        }
        if (result.status !== "not_applicable" && (!Array.isArray(result.evidence_refs) || result.evidence_refs.length === 0)) {
          issues.push(`${label}.dimensions.${dimension}.evidence_refs must not be empty at completion`);
        }
      }
    }
    for (const [eventIndex, event] of (capability.events || []).entries()) {
      const eventLabel = `${label}.events[${eventIndex}]`;
      if (!isObject(event)) {
        issues.push(`${eventLabel} must be an object`);
        continue;
      }
      if (typeof event.routing_key !== "string" || event.routing_key.length === 0) issues.push(`${eventLabel}.routing_key is required`);
      validateStringArray(event.producer_refs, `${eventLabel}.producer_refs`, issues, { nonEmpty: true });
      validateStringArray(event.consumer_refs, `${eventLabel}.consumer_refs`, issues, { nonEmpty: true });
      validateStringArray(event.final_effect_checks, `${eventLabel}.final_effect_checks`, issues, { nonEmpty: true });
    }
  }
  return { valid: issues.length === 0, issues, capability_ids: [...capabilityIds].sort() };
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
    const result = validateParityManifest(JSON.parse(fs.readFileSync(args.file, "utf8")), args);
    if (args.json) console.log(JSON.stringify(result, null, 2));
    else {
      for (const issue of result.issues) console.error(`error: ${issue}`);
      console.log(`parity_result: ${result.valid ? "PASS" : "FAIL"}`);
    }
    process.exit(result.valid ? 0 : 1);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exit(2);
  }
}
