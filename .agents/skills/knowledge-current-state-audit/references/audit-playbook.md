# Knowledge Current-State Audit Playbook

## Contents

1. Evidence method
2. Mandatory coverage dimensions
3. Claim extraction
4. Drift attribution
5. Coverage analysis
6. Completion gate

## 1. Evidence method

Use this order for every material conclusion:

1. Security constraints and nearest `AGENTS.md`.
2. Explicitly activated active SPEC/Harness contract, if any.
3. Current protobuf definitions, schema/migrations, code, tests, and runtime evidence.
4. Accepted ADRs.
5. Active knowledge.

For each active document:

1. Parse frontmatter and identify `applies_to` and exact `source_refs`.
2. Extract claims whose incorrectness could mislead implementation, review, testing, operations, or security decisions.
3. Verify each critical claim against at least one authoritative source; use multiple sources for cross-service behavior.
4. Record exact repository-relative paths and line/symbol evidence.
5. Assign one document verdict based on the worst unresolved material claim.

Do not treat file existence, recent dates, repeated prose, or another ordinary knowledge document as sufficient proof.

## 2. Mandatory coverage dimensions

Give each stable dimension one coverage status: `REVIEWED`, `PARTIAL`, `NOT_REVIEWED`, or `NOT_APPLICABLE`.

| ID | Dimension | Minimum evidence |
|---|---|---|
| `governance-tooling` | Knowledge protocol, lifecycle, validator, index, manifest | `AGENTS.md`, `.knowledge/README.md`, schemas/scripts |
| `architecture-service-boundaries` | Top-level modules, responsibilities, RPC and table ownership | module roots, service code, deployment ownership |
| `frontend-shared-contracts` | Frontend apps, shared package, routes, API clients, validation | workspace/config, routers, APIs, tests |
| `gateway-api-protobuf` | HTTP routes, middleware, handlers, RPC clients, protobuf contracts | router/handler/rpc/proto/generated code |
| `persistence-migrations` | Baseline schema, migrations, models, repositories, ownership | `db.sql`, migrations, persistence code |
| `auth-rbac-security` | Tokens, sessions, permission catalog, data scopes, frontend gates | Gateway/Identity/authz/migrations/frontend |
| `recruitment-notification` | Jobs, applications, interviews, offers, events, outbox, labels | services/proto/migrations/frontends/tests |
| `ai-agent-mcp-retrieval` | Agent runtime, configuration, prompt, tools, memory, retrieval, fallback | AI service, commons AI, Gateway, platform/HR UI |
| `resume-sensitive-data` | Upload, parsing, OSS, profiles, matching, privacy boundaries | Gateway/commons/recruitment/AI/frontend |
| `billing-payment-credits` | Plans, payment lifecycle, refunds, credit metering/settlement | billing service, migrations, Gateway, UI, AI meter |
| `development-deployment-operations` | Local commands, binaries, config, Compose/Kubernetes, health | scripts, service mains, deploy assets |
| `cross-document-route-coverage` | Duplicated facts, missing routes, new modules and operational surfaces | all active metadata, manifest, repository inventory |

## 3. Claim extraction

Prioritize these claim types:

- service ownership and forbidden dependencies;
- route, handler, RPC, protobuf, event, and state-machine mappings;
- database tables, fields, migrations, repository models, and ownership;
- authentication, authorization, tenancy, privacy, and sensitive-data behavior;
- provider/configuration source, precedence, secrets, fallback, retry, timeout, and failure behavior;
- ports, service names, health checks, images, commands, and runbook steps;
- frontend route/menu/API/shared-package behavior;
- recurring pitfalls and their claimed prevention or regression test.

For large documents, sample explanatory prose lightly but verify every normative, enumerated, tabular, command, path, security, public-contract, and cross-module claim.

## 4. Drift attribution

Classify the source of drift:

- `PRE_EXISTING`: mismatch exists at `HEAD`.
- `WORKTREE_INTRODUCED`: `HEAD` knowledge matched, but current uncommitted changes introduce mismatch.
- `MIXED`: both committed and uncommitted changes contribute.
- `UNKNOWN`: history or baseline evidence is insufficient.

Use `git log`, `git blame`, and snapshot reads where useful. Never overwrite or switch away user changes.

## 5. Coverage analysis

Check both directions:

- Source-to-knowledge: high-risk or core source paths must route to useful active knowledge.
- Knowledge-to-source: each active document must cite sufficient current authoritative evidence.

Report:

- missing routes;
- routes to missing, inactive, or irrelevant documents;
- overly broad routes that load excessive context;
- new top-level modules and independent operational commands;
- areas described only by broad overview knowledge despite high change/risk;
- current-state files incorrectly placed in Inbox/archive or unregistered formal directories.

## 6. Completion gate

The audit is complete only when:

- every active document has exactly one verdict;
- every mandatory dimension has a coverage status and evidence/limitation;
- every open finding contains claim-level evidence and attribution;
- every open finding maps to at least one remediation task;
- every task has bounded paths, level, target, dependencies, acceptance criteria, and tests;
- structural checks are recorded, including failures or blockers;
- Markdown/HTML reports and the remediation plan render from one validated JSON source.
