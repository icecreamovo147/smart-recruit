#!/usr/bin/env node
import fs from "node:fs";
import { performance } from "node:perf_hooks";

const defaults = {
  baseUrl: "http://127.0.0.1:8080",
  durationSeconds: 10,
  gatewayQPS: 200,
  writeQPS: 50,
  aiConcurrency: 20,
  ordinaryP95TargetMs: 300,
  complexP95TargetMs: 1000,
  aiSubmitTargetMs: 1000,
};

const args = parseArgs(process.argv.slice(2));
const config = {
  ...defaults,
  baseUrl: args["base-url"] || process.env.LOAD_TEST_BASE_URL || defaults.baseUrl,
  durationSeconds: numberArg(args.duration, defaults.durationSeconds),
  gatewayQPS: numberArg(args["gateway-qps"], defaults.gatewayQPS),
  writeQPS: numberArg(args["write-qps"], defaults.writeQPS),
  aiConcurrency: numberArg(args["ai-concurrency"], defaults.aiConcurrency),
  ordinaryP95TargetMs: numberArg(args["ordinary-p95-ms"], defaults.ordinaryP95TargetMs),
  complexP95TargetMs: numberArg(args["complex-p95-ms"], defaults.complexP95TargetMs),
  aiSubmitTargetMs: numberArg(args["ai-submit-ms"], defaults.aiSubmitTargetMs),
  dryRun: Boolean(args["dry-run"]),
  output: args.output || "",
  authToken: args["auth-token"] || process.env.LOAD_TEST_AUTH_TOKEN || "",
  writePath: args["write-path"] || process.env.LOAD_TEST_WRITE_PATH || "",
  aiPath: args["ai-path"] || process.env.LOAD_TEST_AI_PATH || "",
};

const startedAt = new Date().toISOString();
const plan = buildPlan(config);
const result = config.dryRun ? dryRunResult(plan, config, startedAt) : await runPlan(plan, config, startedAt);
const json = JSON.stringify(result, null, 2);
if (config.output) {
  fs.writeFileSync(config.output, `${json}\n`);
}
console.log(json);

function buildPlan(config) {
  return [
    {
      name: "gateway-ordinary-read",
      kind: "ordinary-api",
      method: "GET",
      path: "/health",
      qps: config.gatewayQPS,
      durationSeconds: config.durationSeconds,
      p95TargetMs: config.ordinaryP95TargetMs,
      enabled: true,
    },
    {
      name: "gateway-complex-readiness",
      kind: "complex-query",
      method: "GET",
      path: "/readyz",
      qps: Math.min(config.gatewayQPS, 50),
      durationSeconds: config.durationSeconds,
      p95TargetMs: config.complexP95TargetMs,
      enabled: true,
    },
    {
      name: "core-write-configured",
      kind: "core-write",
      method: "POST",
      path: config.writePath,
      qps: config.writeQPS,
      durationSeconds: config.durationSeconds,
      p95TargetMs: config.ordinaryP95TargetMs,
      enabled: config.writePath !== "",
      disabledReason: config.writePath === "" ? "Set --write-path to run authenticated 50 QPS core writes." : "",
      body: "{}",
    },
    {
      name: "ai-async-submit-configured",
      kind: "ai-async-submit",
      method: "POST",
      path: config.aiPath,
      concurrency: config.aiConcurrency,
      p95TargetMs: config.aiSubmitTargetMs,
      enabled: config.aiPath !== "",
      disabledReason: config.aiPath === "" ? "Set --ai-path to validate 10-20 concurrent AI/Embedding submissions." : "",
      body: "{}",
    },
  ];
}

function dryRunResult(plan, config, startedAt) {
  return {
    status: "dry_run",
    started_at: startedAt,
    finished_at: new Date().toISOString(),
    base_url: config.baseUrl,
    targets: targetSummary(config),
    scenarios: plan,
    environment_limits: [
      "Dry run does not require a running backend stack or authenticated test data.",
      "Live core write and AI/Embedding checks require --write-path/--ai-path plus suitable auth/test fixtures.",
    ],
  };
}

async function runPlan(plan, config, startedAt) {
  const scenarioResults = [];
  for (const scenario of plan) {
    if (!scenario.enabled) {
      scenarioResults.push({ name: scenario.name, status: "skipped", reason: scenario.disabledReason });
      continue;
    }
    if (scenario.kind === "ai-async-submit") {
      scenarioResults.push(await runConcurrentScenario(scenario, config));
    } else {
      scenarioResults.push(await runQPSScenario(scenario, config));
    }
  }
  return {
    status: scenarioResults.every((r) => r.status === "passed" || r.status === "skipped") ? "completed" : "failed",
    started_at: startedAt,
    finished_at: new Date().toISOString(),
    base_url: config.baseUrl,
    targets: targetSummary(config),
    scenarios: scenarioResults,
  };
}

async function runQPSScenario(scenario, config) {
  const total = Math.max(1, Math.round(scenario.qps * scenario.durationSeconds));
  const intervalMs = 1000 / scenario.qps;
  const latencies = [];
  const statuses = new Map();
  const started = performance.now();
  const requests = [];
  for (let i = 0; i < total; i++) {
    const delay = Math.max(0, started + i * intervalMs - performance.now());
    requests.push(wait(delay).then(() => oneRequest(scenario, config, latencies, statuses)));
  }
  await Promise.all(requests);
  return summarizeScenario(scenario, latencies, statuses, total);
}

async function runConcurrentScenario(scenario, config) {
  const latencies = [];
  const statuses = new Map();
  await Promise.all(
    Array.from({ length: scenario.concurrency }, () => oneRequest(scenario, config, latencies, statuses)),
  );
  return summarizeScenario(scenario, latencies, statuses, scenario.concurrency);
}

async function oneRequest(scenario, config, latencies, statuses) {
  const started = performance.now();
  try {
    const response = await fetch(new URL(scenario.path, config.baseUrl), {
      method: scenario.method,
      headers: requestHeaders(config),
      body: scenario.method === "GET" ? undefined : scenario.body,
    });
    statuses.set(String(response.status), (statuses.get(String(response.status)) || 0) + 1);
    await response.arrayBuffer();
  } catch (error) {
    statuses.set("error", (statuses.get("error") || 0) + 1);
  } finally {
    latencies.push(performance.now() - started);
  }
}

function summarizeScenario(scenario, latencies, statuses, total) {
  const p95 = percentile(latencies, 0.95);
  const failures = [...statuses.entries()]
    .filter(([status]) => status === "error" || Number(status) >= 500)
    .reduce((sum, [, count]) => sum + count, 0);
  return {
    name: scenario.name,
    kind: scenario.kind,
    status: failures === 0 && p95 <= scenario.p95TargetMs ? "passed" : "failed",
    requests: total,
    p95_ms: round(p95),
    p95_target_ms: scenario.p95TargetMs,
    statuses: Object.fromEntries([...statuses.entries()].sort()),
  };
}

function requestHeaders(config) {
  const headers = { "content-type": "application/json" };
  if (config.authToken) {
    headers.authorization = `Bearer ${config.authToken}`;
  }
  return headers;
}

function targetSummary(config) {
  return {
    gateway_qps: config.gatewayQPS,
    core_write_qps: config.writeQPS,
    ai_embedding_concurrency: config.aiConcurrency,
    ordinary_api_p95_ms: config.ordinaryP95TargetMs,
    complex_query_p95_ms: config.complexP95TargetMs,
    ai_submit_p95_ms: config.aiSubmitTargetMs,
  };
}

function percentile(values, p) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(sorted.length - 1, Math.ceil(sorted.length * p) - 1);
  return sorted[index];
}

function parseArgs(argv) {
  const parsed = {};
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (!arg.startsWith("--")) continue;
    const key = arg.slice(2);
    if (key === "dry-run") {
      parsed[key] = true;
      continue;
    }
    parsed[key] = argv[++i] || "";
  }
  return parsed;
}

function numberArg(value, fallback) {
  if (value === undefined || value === "") return fallback;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function round(value) {
  return Math.round(value * 100) / 100;
}
