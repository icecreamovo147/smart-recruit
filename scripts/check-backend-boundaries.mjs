#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const root = process.argv[2] ? path.resolve(process.argv[2]) : process.cwd();
const scanRoots = [
  "smart-recruit-domain-go",
  "smart-recruit-platform-go",
  "smart-recruit-proto",
  "smart-recruit-gateway",
  "smart-recruit-identity-service",
  "smart-recruit-recruitment-service",
  "smart-recruit-interview-service",
  "smart-recruit-offer-service",
  "smart-recruit-notification-service",
  "smart-recruit-ai-agent-service",
  "smart-recruit-analytics-service",
  "smart-recruit-worker-service",
];
const forbidden = ["logic-grpc-service", "web-gin-service"];
const issues = [];

for (const file of listFiles(scanRoots)) {
  const text = fs.readFileSync(file, "utf8");
  for (const token of forbidden) {
    if (text.includes(`"${token}`) || text.includes(`${token}/`)) {
      issues.push(`${toPosix(path.relative(root, file))}: references ${token}`);
    }
  }
}

if (issues.length > 0) {
  console.error("backend_boundary_result: FAIL");
  for (const issue of issues.sort()) console.error(`- ${issue}`);
  process.exit(1);
}

console.log("backend_boundary_result: PASS");

function listFiles(roots) {
  const files = [];
  for (const relRoot of roots) {
    const absRoot = path.join(root, relRoot);
    if (fs.existsSync(absRoot)) walk(absRoot, files);
  }
  return files.filter((file) => file.endsWith(".go") || file.endsWith(".mod"));
}

function walk(current, files) {
  const stat = fs.statSync(current);
  if (stat.isDirectory()) {
    for (const child of fs.readdirSync(current)) walk(path.join(current, child), files);
    return;
  }
  files.push(current);
}

function toPosix(filePath) {
  return filePath.split(path.sep).join("/");
}
