#!/usr/bin/env node
import fs from "node:fs";
import process from "node:process";

const args = parseArgs(process.argv.slice(2));
const services = ["identity", "recruitment", "interview", "offer", "notification", "ai-agent", "analytics"];
const seedGateway = fs.readFileSync("smart-recruit-deploy/nacos/seed-config/gateway.yaml", "utf8");
const seedServices = fs.readFileSync("smart-recruit-deploy/nacos/seed-config/services.yaml", "utf8");
const compose = fs.readFileSync("smart-recruit-deploy/docker-compose.microservices.yml", "utf8");
const issues = [];
const warnings = [];
const routeAudit = [];

for (const service of services) {
  const yamlKey = service === "ai-agent" ? "aiAgent" : service;
  const routeModeEnv = `${envName(service)}_ROUTE_MODE`;
  const grpcAddrEnv = `${envName(service)}_GRPC_ADDR`;
  const cutoverTask = cutoverTaskFor(service);
  const evidencePath = `.spec/microservice-runtime-implementation/reports/${cutoverTask}-evidence.json`;
  if (!seedGateway.includes(`${yamlKey}: logic`)) issues.push(`gateway seed must retain ${yamlKey}: logic rollback mode`);
  if (!seedGateway.includes(`rollbackRouteMode: logic`)) issues.push("gateway seed must document rollbackRouteMode: logic");
  if (!seedServices.includes(`${yamlKey}:`)) issues.push(`services seed missing ${yamlKey}`);
  if (!compose.includes(`${routeModeEnv}: ${service}`)) issues.push(`compose missing direct route mode ${routeModeEnv}: ${service}`);
  if (!compose.includes(`${grpcAddrEnv}: smart-recruit-${service}-service:`) && service !== "identity") {
    issues.push(`compose missing static gRPC target for ${service}`);
  }
  if (service === "identity" && !compose.includes("IDENTITY_GRPC_ADDR: smart-recruit-identity-service:50061")) {
    issues.push("compose missing static gRPC target for identity");
  }
  if (!fs.existsSync(evidencePath)) {
    issues.push(`missing cutover evidence ${evidencePath}`);
  } else {
    const evidence = JSON.parse(fs.readFileSync(evidencePath, "utf8"));
    if (evidence.review?.verdict !== "通过") issues.push(`${cutoverTask} review verdict is not passing`);
  }
  routeAudit.push({service, cutoverTask, rollbackMode: "logic", directMode: service});
}

const smokeEvidencePath = ".spec/microservice-runtime-implementation/reports/TASK-MRI-029-evidence.json";
let liveSmokePassed = false;
if (fs.existsSync(smokeEvidencePath)) {
  const smoke = JSON.parse(fs.readFileSync(smokeEvidencePath, "utf8"));
  liveSmokePassed = (smoke.checks || []).some((check) => check.command === "bash scripts/compose-microservice-smoke.sh" && check.status === "passed");
  if (!liveSmokePassed) warnings.push("live compose smoke has not passed; monolith fallback retirement is blocked");
} else {
  warnings.push("TASK-MRI-029 smoke evidence is missing; monolith fallback retirement is blocked");
}

const fallbackRetained = seedGateway.includes("staticDiscovery: true") && services.every((service) => seedGateway.includes(`${service === "ai-agent" ? "aiAgent" : service}: logic`));
if (!fallbackRetained) issues.push("static fallback or logic rollback route modes are not retained");

const retirementReady = issues.length === 0 && liveSmokePassed;
const report = {
  schemaVersion: 1,
  retirementReady,
  recommendation: retirementReady ? "fallback_retirement_allowed" : "retain_monolith_fallback",
  fallbackRetained,
  liveSmokePassed,
  routeAudit,
  issues,
  warnings
};

if (args.output) {
  fs.writeFileSync(args.output, JSON.stringify(report, null, 2) + "\n");
}

if (issues.length > 0) {
  for (const issue of issues) console.error(`fallback_gate: ${issue}`);
  process.exit(1);
}

console.log(`fallback_gate: PASS recommendation=${report.recommendation}`);
if (warnings.length > 0) {
  for (const warning of warnings) console.log(`fallback_gate_warning: ${warning}`);
}

function cutoverTaskFor(service) {
  return {
    identity: "TASK-MRI-011",
    recruitment: "TASK-MRI-013",
    offer: "TASK-MRI-015",
    interview: "TASK-MRI-017",
    notification: "TASK-MRI-019",
    "ai-agent": "TASK-MRI-021",
    analytics: "TASK-MRI-023"
  }[service];
}

function envName(service) {
  return service.replace("-", "_").toUpperCase();
}

function parseArgs(argv) {
  const parsed = {};
  for (let i = 0; i < argv.length; i++) {
    switch (argv[i]) {
      case "--output":
        parsed.output = argv[++i];
        break;
      case "--check":
        break;
      default:
        console.error(`fallback_gate: unknown argument ${argv[i]}`);
        process.exit(2);
    }
  }
  return parsed;
}
