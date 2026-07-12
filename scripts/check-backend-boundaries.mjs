#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const root = process.argv[2] ? path.resolve(process.argv[2]) : process.cwd();
const internalRoot = path.join(root, "logic-grpc-service", "internal");

const contexts = [
  "identity",
  "recruitment",
  "interview",
  "offer",
  "notification",
  "aiagent",
  "analytics",
  path.join("platform", "workers"),
];
const layers = ["domain", "application", "infrastructure", "interfaces"];

const issues = [];

function toPosix(filePath) {
  return filePath.split(path.sep).join("/");
}

function readGoImports(filePath) {
  const source = fs.readFileSync(filePath, "utf8");
  const imports = [];
  const block = source.match(/import\s*\(([\s\S]*?)\)/m);
  if (block) {
    for (const line of block[1].split(/\r?\n/)) {
      const match = line.match(/"([^"]+)"/);
      if (match) imports.push(match[1]);
    }
  }
  const single = source.match(/import\s+"([^"]+)"/m);
  if (single) imports.push(single[1]);
  return imports;
}

function walk(dir, visitor) {
  if (!fs.existsSync(dir)) return;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(fullPath, visitor);
    } else {
      visitor(fullPath);
    }
  }
}

function contextFor(filePath) {
  const rel = path.relative(internalRoot, filePath);
  const parts = rel.split(path.sep);
  if (parts[0] === "platform" && parts[1] === "workers") {
    return { contextPath: path.join("platform", "workers"), layer: parts[2] };
  }
  return { contextPath: parts[0], layer: parts[1] };
}

for (const contextPath of contexts) {
  for (const layer of layers) {
    const docPath = path.join(internalRoot, contextPath, layer, "doc.go");
    if (!fs.existsSync(docPath)) {
      issues.push(`missing layer doc: ${toPosix(path.relative(root, docPath))}`);
    }
  }
}

walk(internalRoot, (filePath) => {
  if (!filePath.endsWith(".go") || filePath.endsWith("_test.go")) return;
  const rel = toPosix(path.relative(root, filePath));
  const base = path.basename(filePath);
  const { contextPath, layer } = contextFor(filePath);
  if (!contexts.includes(contextPath)) return;

  if (base.includes("service_read")) {
    issues.push(`forbidden service_read file: ${rel}`);
  }

  const imports = readGoImports(filePath);
  const forbidden = [];

  if (layer === "domain") {
    forbidden.push(
      "logic-grpc-service/repository",
      "logic-grpc-service/service",
      "logic-grpc-service/model",
      "logic-grpc-service/recruitment/pb",
      "gorm.io/gorm",
    );
  }
  if (layer === "application") {
    forbidden.push("logic-grpc-service/service");
  }
  if (layer === "interfaces") {
    forbidden.push("logic-grpc-service/repository", "gorm.io/gorm");
  }
  if (contextPath === "analytics") {
    forbidden.push("logic-grpc-service/service");
  }

  for (const imported of imports) {
    for (const prefix of forbidden) {
      if (imported === prefix || imported.startsWith(`${prefix}/`)) {
        issues.push(`${rel} imports forbidden ${imported}`);
      }
    }

    if (imported.startsWith("logic-grpc-service/internal/")) {
      const importedRel = imported.slice("logic-grpc-service/internal/".length);
      const importedContext = importedRel.startsWith("platform/workers/")
        ? "platform/workers"
        : importedRel.split("/")[0];
      if (contexts.includes(importedContext) && importedContext !== contextPath) {
        issues.push(`${rel} imports other bounded context ${imported}`);
      }
    }
  }
});

if (issues.length > 0) {
  console.error("backend_boundary_result: FAIL");
  for (const issue of issues.sort()) console.error(`- ${issue}`);
  process.exit(1);
}

console.log("backend_boundary_result: PASS");
