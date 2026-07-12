#!/usr/bin/env node
import fs from "node:fs";
import process from "node:process";

const required = [
  "notification-consumer",
  "email-consumer",
  "resume-parse-consumer",
  "embedding-consumer",
  "agent-run-consumer",
  "analytics-projection-consumer",
];

const source = fs.readFileSync("smart-recruit-domain-go/internal/platform/events/boundary.go", "utf8");
const missing = [];
for (const name of required) {
  if (!source.includes(`Name: "${name}"`)) missing.push(`${name}: missing boundary`);
}
for (const token of ["InboxRequired: true", "RetryRequired: true", "DLQRequired: true", "ReplayRequired: true"]) {
  if (!source.includes(token)) missing.push(`${token}: missing required safety flag`);
}
if (!source.includes("envelope.idempotency_key") || !source.includes("envelope.event_id")) {
  missing.push("idempotency sources must reference envelope.idempotency_key or envelope.event_id");
}

if (missing.length > 0) {
  for (const item of missing) console.error(`idempotency_check: ${item}`);
  process.exit(1);
}

console.log("idempotency_check: PASS");
