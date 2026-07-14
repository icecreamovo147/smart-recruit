#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const root = process.argv[2] ? path.resolve(process.argv[2]) : process.cwd();
const scanRoots = [
  "smart-recruit-commons",
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
const legacydomainUsage = {
  directories: [],
  imports: [],
};

for (const file of listFiles(scanRoots)) {
  const text = fs.readFileSync(file, "utf8");
  const relative = toPosix(path.relative(root, file));
  checkLegacyReferences(relative, text);
  collectLegacydomainUsage(relative, text);

  if (!file.endsWith(".go")) continue;
  const imports = parseGoImports(text);
  checkLayerImports(relative, imports);
}

recordLegacydomainRetirementIssues();

if (issues.length > 0) {
  console.error("backend_boundary_result: FAIL");
  for (const issue of issues.sort()) console.error(`- ${issue}`);
  process.exit(1);
}

console.log("backend_boundary_result: PASS");
printLegacydomainRetirementReport();

function checkLegacyReferences(relative, text) {
  for (const token of forbidden) {
    if (text.includes(`"${token}`) || text.includes(`${token}/`)) {
      issues.push(`${relative}: references legacy module ${token}`);
    }
  }
}

function collectLegacydomainUsage(relative, text) {
  if (relative.includes("/internal/legacydomain/")) {
    const rootPath = relative.replace(/\/internal\/legacydomain\/.*/, "/internal/legacydomain");
    if (!legacydomainUsage.directories.includes(rootPath)) {
      legacydomainUsage.directories.push(rootPath);
    }
  }
  if (!relative.endsWith(".go")) return;
  if (relative.includes("/internal/legacydomain/")) return;
  const imports = parseGoImports(text).filter((item) => item.includes("/internal/legacydomain/"));
  for (const importPath of imports) {
    legacydomainUsage.imports.push(`${relative}: ${importPath}`);
  }
}

function recordLegacydomainRetirementIssues() {
  const directories = legacydomainUsage.directories.sort();
  const imports = legacydomainUsage.imports.sort();
  for (const item of directories) {
    issues.push(`${item}: internal/legacydomain directory is forbidden after legacydomain retirement`);
  }
  for (const item of imports) {
    issues.push(`${item}: internal/legacydomain import is forbidden after legacydomain retirement`);
  }
}

function printLegacydomainRetirementReport() {
  const directories = legacydomainUsage.directories.sort();
  const imports = legacydomainUsage.imports.sort();
  if (directories.length === 0 && imports.length === 0) {
    console.log("legacydomain_retirement_enforcement: PASS (no legacydomain directories or imports)");
    return;
  }
  console.log(`legacydomain_retirement_enforcement: FAIL (${directories.length} roots, ${imports.length} import sites)`);
  for (const item of directories) console.log(`- legacy root: ${item}`);
  for (const item of imports) console.log(`- legacy import: ${item}`);
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
    "smart-recruit-commons/repository",
    "smart-recruit-commons/service",
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
    "smart-recruit-commons/repository",
  ]);
}

function isPersistenceImport(importPath) {
  return matchesAny(importPath, ["gorm.io/", "smart-recruit-commons/repository"]);
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
