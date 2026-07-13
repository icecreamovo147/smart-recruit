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
  const relative = toPosix(path.relative(root, file));
  checkLegacyReferences(relative, text);

  if (!file.endsWith(".go")) continue;
  const imports = parseGoImports(text);
  checkLayerImports(relative, imports);
}

if (issues.length > 0) {
  console.error("backend_boundary_result: FAIL");
  for (const issue of issues.sort()) console.error(`- ${issue}`);
  process.exit(1);
}

console.log("backend_boundary_result: PASS");

function checkLegacyReferences(relative, text) {
  for (const token of forbidden) {
    if (text.includes(`"${token}`) || text.includes(`${token}/`)) {
      issues.push(`${relative}: references legacy module ${token}`);
    }
  }
}

function checkLayerImports(relative, imports) {
  if (isDomainLayer(relative)) {
    for (const item of imports) {
      if (isForbiddenDomainImport(item)) {
        issues.push(`${relative}: domain layer imports forbidden outer dependency ${item}`);
      }
      if (isSameServiceOuterImport(relative, item)) {
        issues.push(`${relative}: domain layer imports outer service layer ${item}`);
      }
    }
  }

  if (isApplicationLayer(relative)) {
    for (const item of imports) {
      if (isForbiddenApplicationImport(item)) {
        issues.push(`${relative}: application layer imports forbidden infrastructure dependency ${item}`);
      }
      if (isSameServiceInfrastructureImport(relative, item)) {
        issues.push(`${relative}: application layer imports local infrastructure ${item}`);
      }
    }
  }

  if (isInterfacesLayer(relative)) {
    for (const item of imports) {
      if (isPersistenceImport(item) || isSameServicePersistenceImport(relative, item)) {
        issues.push(`${relative}: interfaces layer imports persistence dependency ${item}`);
      }
    }
  }
}

function parseGoImports(text) {
  const imports = [];
  const blockPattern = /import\s*\(([\s\S]*?)\)/g;
  let match;
  while ((match = blockPattern.exec(text)) !== null) {
    for (const item of match[1].matchAll(/"([^"]+)"/g)) {
      imports.push(item[1]);
    }
  }

  const singlePattern = /import\s+(?:[.\w]+\s+)?"([^"]+)"/g;
  while ((match = singlePattern.exec(text)) !== null) {
    imports.push(match[1]);
  }
  return imports;
}

function isDomainLayer(relative) {
  return relative.includes("/internal/domain/");
}

function isApplicationLayer(relative) {
  return relative.includes("/internal/application/");
}

function isInterfacesLayer(relative) {
  return relative.includes("/internal/interfaces/");
}

function isForbiddenDomainImport(importPath) {
  return matchesAny(importPath, [
    "gorm.io/",
    "github.com/redis/go-redis",
    "github.com/rabbitmq/amqp091-go",
    "google.golang.org/grpc",
    "github.com/gin-gonic/gin",
    "net/http",
    "database/sql",
    "smart-recruit-platform-go/",
    "smart-recruit-proto/",
    "smart-recruit-domain-go/repository",
    "smart-recruit-domain-go/service",
  ]);
}

function isForbiddenApplicationImport(importPath) {
  return matchesAny(importPath, [
    "gorm.io/",
    "github.com/redis/go-redis",
    "github.com/rabbitmq/amqp091-go",
    "google.golang.org/grpc",
    "github.com/gin-gonic/gin",
    "net/http",
    "database/sql",
    "smart-recruit-domain-go/repository",
  ]);
}

function isPersistenceImport(importPath) {
  return matchesAny(importPath, ["gorm.io/", "smart-recruit-domain-go/repository"]);
}

function isSameServiceOuterImport(relative, importPath) {
  const service = relative.split("/")[0];
  return matchesAny(importPath, [
    `${service}/internal/application/`,
    `${service}/internal/infrastructure/`,
    `${service}/internal/interfaces/`,
    `${service}/internal/runtime/`,
  ]);
}

function isSameServiceInfrastructureImport(relative, importPath) {
  const service = relative.split("/")[0];
  return importPath.startsWith(`${service}/internal/infrastructure/`);
}

function isSameServicePersistenceImport(relative, importPath) {
  const service = relative.split("/")[0];
  return importPath.startsWith(`${service}/internal/infrastructure/persistence/`);
}

function matchesAny(importPath, patterns) {
  return patterns.some((pattern) => importPath === pattern || importPath.startsWith(pattern));
}

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
