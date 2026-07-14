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
const runtimeStubUsage = {
  emptySuccess: [],
  storeNilSuccess: [],
  unimplementedNativeServers: [],
};

const allowedRuntimeStubFindings = new Set([
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:unavailableEmbeddingConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:unavailableLlmConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopAIService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopAgentConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopAgentSkillService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopEmbeddingConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopLlmConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopMCPService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopPromptService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopRecruitingIntelligenceService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/cmd/ai-agent-service/main.go:noopSkillService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:CompareCandidatesForJob:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:CreateAgentRun:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:DeleteSession:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:GetActiveAgentRun:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:GetActiveAgentRun:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:GetAgentRuns:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:GetAgentRuns:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:GetToolTraces:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:GetToolTraces:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListAgents:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListAgents:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListAgentSkills:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListAgentSkills:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListAvailableAgentSkills:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListAvailableAgentSkills:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListCapabilities:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListEmbeddingModels:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListEmbeddingModels:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListEmbeddingProviders:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListEmbeddingProviders:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListMCPServers:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListMCPServers:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListMCPToolLogs:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListMCPToolPolicies:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListModels:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListModels:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListProviders:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListProviders:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListPromptTemplates:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListPromptTemplates:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:ListSkills:empty-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:UpdateSession:store-nil-success",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeAIService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeAgentConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeAgentSkillService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeEmbeddingConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeLlmConfigService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeMCPService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativePromptService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeRecruitingIntelligenceService:unimplemented-native-server",
  "smart-recruit-ai-agent-service/internal/interfaces/grpc/native_servers.go:nativeSkillService:unimplemented-native-server",
]);

for (const file of listFiles(scanRoots)) {
  const text = fs.readFileSync(file, "utf8");
  const relative = toPosix(path.relative(root, file));
  checkLegacyReferences(relative, text);
  collectLegacydomainUsage(relative, text);
  collectRuntimeStubUsage(relative, text);

  if (!file.endsWith(".go")) continue;
  const imports = parseGoImports(text);
  checkLayerImports(relative, imports);
}

recordLegacydomainRetirementIssues();
recordRuntimeStubIssues();

if (issues.length > 0) {
  console.error("backend_boundary_result: FAIL");
  for (const issue of issues.sort()) console.error(`- ${issue}`);
  process.exit(1);
}

console.log("backend_boundary_result: PASS");
printLegacydomainRetirementReport();
printRuntimeStubReport();

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

function collectRuntimeStubUsage(relative, text) {
  if (!relative.startsWith("smart-recruit-ai-agent-service/")) return;
  if (!relative.endsWith(".go") || relative.endsWith("_test.go")) return;

  collectUnimplementedNativeServers(relative, text);
  for (const fn of parseGoFunctions(text)) {
    const labelBase = `${relative}:${fn.name}`;
    if (hasStoreNilSuccessFallback(fn.body)) {
      runtimeStubUsage.storeNilSuccess.push(`${labelBase}:store-nil-success`);
    }
    if (hasUnconditionalEmptySuccess(fn.body)) {
      runtimeStubUsage.emptySuccess.push(`${labelBase}:empty-success`);
    }
  }
}

function collectUnimplementedNativeServers(relative, text) {
  const typePattern = /type\s+(\w+)\s+struct\s*\{([\s\S]*?)\n\}/g;
  let match;
  while ((match = typePattern.exec(text)) !== null) {
    const typeName = match[1];
    const body = match[2];
    if (!/\bpb\.Unimplemented\w+ServiceServer\b/.test(body)) continue;
    if (!/^(native|noop|unavailable)/.test(typeName)) continue;
    runtimeStubUsage.unimplementedNativeServers.push(`${relative}:${typeName}:unimplemented-native-server`);
  }
}

function parseGoFunctions(text) {
  const functions = [];
  const pattern = /func\s+(?:\([^)]*\)\s*)?(\w+)\s*\([^)]*\)\s*(?:\([^)]*\)|\*\w+(?:\.\w+)?|\w+(?:\.\w+)?)?\s*\{/g;
  let match;
  while ((match = pattern.exec(text)) !== null) {
    const name = match[1];
    const bodyStart = pattern.lastIndex - 1;
    const bodyEnd = findMatchingBrace(text, bodyStart);
    if (bodyEnd === -1) continue;
    functions.push({ name, body: text.slice(bodyStart, bodyEnd + 1) });
    pattern.lastIndex = bodyEnd + 1;
  }
  return functions;
}

function findMatchingBrace(text, openIndex) {
  let depth = 0;
  for (let i = openIndex; i < text.length; i++) {
    const ch = text[i];
    if (ch === "{") depth++;
    if (ch === "}") {
      depth--;
      if (depth === 0) return i;
    }
  }
  return -1;
}

function hasStoreNilSuccessFallback(body) {
  return /if\s+[\w.]+\.?store\s*==\s*nil\s*\{[\s\S]*?(?:Code:\s*0,\s*Msg:\s*"success"|commonOK\(\)|fallbackAgentRun)/.test(body);
}

function hasUnconditionalEmptySuccess(body) {
  return /return\s+&pb\.\w+\{Code:\s*0,\s*Msg:\s*"success"\},\s*nil/.test(body);
}

function recordRuntimeStubIssues() {
  const findings = [
    ...runtimeStubUsage.unimplementedNativeServers,
    ...runtimeStubUsage.storeNilSuccess,
    ...runtimeStubUsage.emptySuccess,
  ].sort();
  for (const finding of findings) {
    if (!allowedRuntimeStubFindings.has(finding)) {
      issues.push(`${finding}: AI Agent runtime stub is not in the finalization baseline`);
    }
  }
}

function printRuntimeStubReport() {
  const findings = [
    ...runtimeStubUsage.unimplementedNativeServers,
    ...runtimeStubUsage.storeNilSuccess,
    ...runtimeStubUsage.emptySuccess,
  ].sort();
  if (findings.length === 0) {
    console.log("runtime_stub_guardrail: PASS (no targeted AI Agent runtime stubs found)");
    return;
  }
  console.log(`runtime_stub_guardrail: PASS (${findings.length} known targeted AI Agent runtime stub finding(s), 0 new)`);
  for (const finding of findings) console.log(`- known runtime stub: ${finding}`);
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
