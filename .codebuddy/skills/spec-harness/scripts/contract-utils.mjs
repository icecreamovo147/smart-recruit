import crypto from "node:crypto";
import fs from "node:fs";

export const PROFILES = new Set(["feature_delivery", "behavior_preserving_migration"]);
export const PLANNING_STATUSES = new Set(["draft", "review_pending", "approved", "amendment_pending", "blocked"]);
const REVIEW_POLICIES = new Set(["self_allowed", "independent_required", "independent_and_human"]);
const COMPLETION_LEVELS = Object.freeze({
  feature_delivery: ["planned", "code_complete", "integration_verified", "behavior_verified", "completed"],
  behavior_preserving_migration: ["discovered", "baseline_locked", "implemented", "parity_verified", "cutover_ready", "cutover_observed", "retirement_ready", "retired"],
});

export function isObject(value) {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

export function canonicalize(value) {
  if (Array.isArray(value)) return value.map(canonicalize);
  if (!isObject(value)) return value;
  return Object.fromEntries(Object.keys(value).sort().map((key) => [key, canonicalize(value[key])]));
}

export function hashJson(value) {
  return crypto.createHash("sha256").update(JSON.stringify(canonicalize(value))).digest("hex");
}

export function hashFile(file) {
  return crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex");
}

export function validateStringArray(value, label, issues, { nonEmpty = false } = {}) {
  if (!Array.isArray(value)) {
    issues.push(`${label} must be an array`);
    return;
  }
  if (nonEmpty && value.length === 0) issues.push(`${label} must not be empty`);
  for (const item of value) {
    if (typeof item !== "string" || item.length === 0) issues.push(`${label} entries must be non-empty strings`);
  }
}

export function validateContract(contract, featureName, options = {}) {
  const issues = [];
  if (!isObject(contract)) return { valid: false, issues: ["contract.json must be an object"] };
  if (contract.schemaVersion !== 2) issues.push("contract.json schemaVersion must be 2");
  if (contract.feature_name !== featureName) issues.push("contract.json feature_name does not match directory");
  if (!PROFILES.has(contract.profile)) issues.push(`invalid contract profile: ${contract.profile}`);
  if (!Number.isInteger(contract.contractRevision) || contract.contractRevision < 1) issues.push("contractRevision must be a positive integer");
  if (!PLANNING_STATUSES.has(contract.planningStatus)) issues.push(`invalid planningStatus: ${contract.planningStatus}`);
  if (typeof contract.requiredCompletionLevel !== "string" || contract.requiredCompletionLevel.length === 0) {
    issues.push("requiredCompletionLevel must be a non-empty string");
  } else if (PROFILES.has(contract.profile) && !COMPLETION_LEVELS[contract.profile].includes(contract.requiredCompletionLevel)) {
    issues.push(`requiredCompletionLevel is invalid for profile ${contract.profile}`);
  } else if (contract.profile === "behavior_preserving_migration" && COMPLETION_LEVELS.behavior_preserving_migration.indexOf(contract.requiredCompletionLevel) < COMPLETION_LEVELS.behavior_preserving_migration.indexOf("parity_verified")) {
    issues.push("behavior_preserving_migration must require at least parity_verified");
  }
  if ("rollingWindow" in contract && (!Number.isInteger(contract.rollingWindow) || contract.rollingWindow < 1 || contract.rollingWindow > 5)) {
    issues.push("rollingWindow must be an integer between 1 and 5");
  }
  if (!Array.isArray(contract.assumptions)) issues.push("assumptions must be an array");
  if (!isObject(contract.reviewPolicy)) issues.push("reviewPolicy must be an object");
  else {
    for (const key of ["plan", "highRiskTask", "amendment", "deleteSource"]) {
      if (!REVIEW_POLICIES.has(contract.reviewPolicy[key])) issues.push(`reviewPolicy.${key} is invalid`);
    }
  }
  for (const [index, assumption] of (contract.assumptions || []).entries()) {
    if (!isObject(assumption)) {
      issues.push(`assumptions[${index}] must be an object`);
      continue;
    }
    if (typeof assumption.id !== "string" || assumption.id.length === 0) issues.push(`assumptions[${index}].id is required`);
    if (!new Set(["verified", "open", "rejected"]).has(assumption.status)) issues.push(`assumptions[${index}].status is invalid`);
    if (typeof assumption.blocking !== "boolean") issues.push(`assumptions[${index}].blocking must be boolean`);
    validateStringArray(assumption.evidenceRefs, `assumptions[${index}].evidenceRefs`, issues);
    validateStringArray(assumption.affects, `assumptions[${index}].affects`, issues);
  }
  if (contract.planningStatus === "approved" && (contract.assumptions || []).some((item) => item.blocking && item.status !== "verified")) {
    issues.push("approved contract contains an unresolved blocking assumption");
  }
  if (contract.profile === "behavior_preserving_migration") {
    if (!isObject(contract.baseline)) {
      issues.push("behavior_preserving_migration requires baseline");
    } else {
      if (typeof contract.baseline.ref !== "string" || contract.baseline.ref.length === 0) issues.push("baseline.ref is required");
      if (typeof contract.baseline.sha !== "string" || contract.baseline.sha.length < 7) issues.push("baseline.sha must be a pinned Git identifier");
      if (typeof contract.baseline.tree !== "string" || contract.baseline.tree.length < 7) issues.push("baseline.tree must be a pinned Git tree identifier");
      if (contract.baseline.immutable !== true) issues.push("migration baseline must set immutable=true");
    }
  }
  if (options.requireApproved && contract.planningStatus !== "approved") issues.push("contract planningStatus must be approved");
  return { valid: issues.length === 0, issues };
}
