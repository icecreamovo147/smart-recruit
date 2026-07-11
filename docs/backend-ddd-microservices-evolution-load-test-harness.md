# Backend DDD Microservices Evolution Load Test Harness

Status: TASK-BDME-051 initial harness and dry-run evidence
Last verified: 2026-07-11

## Purpose

This document records the initial backend load-test harness for the DDD/microservices evolution. It provides a repeatable command surface for the agreed starting targets:

- gateway ordinary APIs at 200 QPS;
- core writes at 50 QPS;
- AI/Embedding submissions at 10-20 concurrent tasks;
- ordinary API P95 below 300 ms;
- complex query P95 below 1 s;
- AI/Embedding work submitted asynchronously so the main transaction path is not held open by provider execution.

## Harness

The harness is `scripts/backend-load-test.mjs`. It uses Node.js built-ins only and can run in two modes:

- `--dry-run`: emit the target matrix and environment limits without sending traffic.
- live mode: send HTTP requests to `--base-url` and measure request status and P95 latency.

Default dry run:

```bash
node scripts/backend-load-test.mjs \
  --dry-run \
  --output docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json
```

Example live smoke run against a local gateway:

```bash
node scripts/backend-load-test.mjs \
  --base-url http://127.0.0.1:8080 \
  --duration 10 \
  --gateway-qps 200
```

Example live run with authenticated write and AI/Embedding submission paths:

```bash
LOAD_TEST_AUTH_TOKEN=<test-token> \
node scripts/backend-load-test.mjs \
  --base-url https://gateway.example.internal \
  --duration 60 \
  --gateway-qps 200 \
  --write-qps 50 \
  --write-path /api/v1/<safe-test-write-endpoint> \
  --ai-concurrency 20 \
  --ai-path /api/v1/<safe-test-ai-submit-endpoint> \
  --output docs/backend-ddd-microservices-evolution-load-test-live-evidence.json
```

Write and AI paths are intentionally caller-supplied. The repository does not contain production-safe credentials, disposable tenant data, or a dedicated write fixture endpoint, so the checked-in harness must not guess a mutating route.

## Initial Evidence

TASK-BDME-051 executed the dry-run command and wrote:

`docs/backend-ddd-microservices-evolution-load-test-initial-evidence.json`

The dry run validates that the harness can generate the agreed target matrix. It does not claim live latency compliance because this environment does not provide a running backend stack, authenticated test users, seeded write fixtures, AI provider credentials, or isolated load-test infrastructure.

## AI/Embedding Async Validation

The live AI/Embedding scenario measures submission latency for concurrent tasks rather than provider completion time. A passing live run requires submission P95 to remain below the configured `--ai-submit-ms` target, showing that the request path is accepting asynchronous work instead of waiting for provider execution.

Current code evidence for asynchronous AI/Embedding work remains:

- `logic-grpc-service/service/ai_agent_runtime.go` composes durable agent-run and embedding runtime workers.
- `logic-grpc-service/cmd/worker-services` names agent-run and embedding worker workloads for later independent scaling.
- `docs/backend-ddd-microservices-evolution-ai-agent-runtime-extraction.md` records the extracted AI Agent runtime boundary and remaining cutover controls.

## Environment Limits

The current repository environment can run the harness dry-run and unit checks. It cannot produce authoritative P95 numbers without:

- a running gateway/logic/worker stack;
- seeded representative jobs, applications, candidate data, and safe write fixtures;
- authenticated test tokens for HR/candidate/staff paths;
- configured Redis, MySQL, RabbitMQ, object storage, and optional AI provider credentials;
- an isolated environment where 200 QPS reads and 50 QPS writes will not mutate shared development data.

When those limits are removed, save live evidence as JSON with the harness `--output` flag and attach the exact command, environment, git SHA, duration, target QPS/concurrency, P95 results, and any skipped scenarios.
