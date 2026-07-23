#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const args = parseArgs(process.argv.slice(2));
const root = process.cwd();
const featureDir = path.resolve(root, args["feature-dir"] || ".spec/backend-ddd-microservices-evolution");
const featureDocsDir = path.join(featureDir, "docs");
const featureDocsRel = path.relative(root, featureDocsDir);
const allowCurrentTask = args["allow-current-task"] || "";
const output = args.output || "";

const taskScope = readJSON(path.join(featureDir, "task-scope.json"));
const state = readJSON(path.join(featureDir, "pipeline-state.json"));
const tableManifest = readJSON(path.join(featureDocsDir, "backend-ddd-microservices-evolution-table-ownership-manifest.json"));
const taskIds = Object.keys(taskScope.tasks || {});
const completed = new Set(state.completed_tasks || []);
const missingEvidence = [];
for (const taskId of taskIds) {
  if (taskId === allowCurrentTask) continue;
  const run = state.task_runs?.[taskId];
  if (!completed.has(taskId) || !run?.evidence || !fs.existsSync(path.join(root, run.evidence))) {
    missingEvidence.push(taskId);
  }
}

const requiredArtifacts = [
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-architecture-baseline.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-bounded-context-contract.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-table-ownership-manifest.json"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-schema-separation-plan.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-internal-service-security.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-observability-baseline.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-deployment-readiness-baseline.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-load-test-harness.md"),
  path.join(featureDocsRel, "backend-ddd-microservices-evolution-load-test-initial-evidence.json"),
  "scripts/check-backend-boundaries.mjs",
  "scripts/check-table-ownership.mjs",
  "scripts/backend-load-test.mjs",
];
const missingArtifacts = requiredArtifacts.filter((artifact) => !fs.existsSync(path.join(root, artifact)));

const transitionalAccess = [];
for (const [table, details] of Object.entries(tableManifest.tables || {})) {
  for (const entry of details.transitional_shared_access || []) {
    transitionalAccess.push({
      table,
      owner: details.owner,
      accessor: entry.accessor,
      has_removal_plan: Boolean(entry.removal_plan),
      removal_plan: entry.removal_plan || "",
    });
  }
}

const transitionalWithoutPlan = transitionalAccess.filter((entry) => !entry.has_removal_plan);
const result = {
  status: missingEvidence.length === 0 && missingArtifacts.length === 0 && transitionalWithoutPlan.length === 0 ? "passed" : "failed",
  feature_name: state.feature_name,
  task_count: taskIds.length,
  completed_count_excluding_allowed_current: taskIds.length - (allowCurrentTask ? 1 : 0) - missingEvidence.length,
  allow_current_task: allowCurrentTask,
  missing_evidence: missingEvidence,
  missing_artifacts: missingArtifacts,
  transitional_shared_access_count: transitionalAccess.length,
  transitional_without_removal_plan: transitionalWithoutPlan,
  generated_at: new Date().toISOString(),
};

const json = JSON.stringify(result, null, 2);
if (output) {
  fs.writeFileSync(output, `${json}\n`);
}
console.log(json);
if (result.status !== "passed") {
  process.exit(1);
}

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function parseArgs(argv) {
  const parsed = {};
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (!arg.startsWith("--")) continue;
    parsed[arg.slice(2)] = argv[++i] || "";
  }
  return parsed;
}
