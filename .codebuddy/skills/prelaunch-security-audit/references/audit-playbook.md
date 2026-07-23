# Smart Recruit Prelaunch Security Audit Playbook

## Contents

1. Evidence rules
2. Repository attack-surface routing
3. Mandatory review domains
4. Validation command catalogue
5. Dynamic-testing authorization levels
6. Completion checklist

## 1. Evidence rules

- Prefer current code, tests, schema, protobuf, and deployment manifests over prose.
- Cite repository-relative file paths and exact lines or symbols.
- Redact secret values and personal information. A report may say “tracked credential-like value at path X” but must not reproduce it.
- Treat a scanner result as a lead until reachability, affected version, and call/configuration path are confirmed.
- Distinguish `PASS`, `FAIL`, `NOT_RUN`, `BLOCKED`, and `NOT_APPLICABLE`.
- A missing control or missing production evidence is not a passing result.
- Record dirty-worktree limitations and preserve all user changes.

## 2. Repository attack-surface routing

| Surface | Primary paths | Review focus |
|---|---|---|
| Browser clients | `hr-frontend/`, `user-frontend/`, `interviewer-frontend/`, `platform-frontend/`, `packages/shared/` | token storage, route guards, XSS, CSRF, unsafe HTML, URL handling, sensitive caching |
| HTTP boundary | `smart-recruit-gateway/router/`, `handler/`, `middleware/`, `config/`, `rpc/` | route coverage, authn/authz order, limits, CORS/CSP, errors, public admin endpoints, metadata forwarding |
| Identity | `smart-recruit-identity-service/`, `smart-recruit-commons/pkg/jwt/`, `pkg/authz/` | passwords, refresh rotation/reuse, JWT validation, token invalidation, RBAC, scopes, audit |
| Recruitment data | `smart-recruit-recruitment-service/`, recruitment proto/schema/migrations | ownership, tenant filters, state transitions, exports, candidate privacy |
| Interviews and offers | `smart-recruit-interview-service/`, `smart-recruit-offer-service/` | assignment checks, lifecycle authorization, compensation confidentiality, idempotency |
| Notifications/workers | `smart-recruit-notification-service/`, `smart-recruit-worker-service/`, MQ/outbox/inbox code | replay, poison messages, retry/DLQ, template injection, recipient authorization |
| AI control/runtime | `smart-recruit-ai-agent-service/`, `smart-recruit-commons/ai/` | prompt injection, model/tool allowlists, memory poisoning, output handling, cost/DoS, credential leakage |
| MCP | MCP domain, policy, persistence, Gateway handlers, platform UI | SSRF, DNS rebinding, private networks, stdio allowlists, explicit selection, confirmation, redaction, audit |
| Resume/OSS | Gateway resume handlers, `smart-recruit-commons/oss/`, `resumeparser/` | presign binding, file validation, malware/polyglots, decompression bombs, sandboxing, URL leakage |
| Analytics | `smart-recruit-analytics-service/` | aggregation tenant scope, inference leakage, projection replay and stale authorization |
| Billing | `smart-recruit-billing-service/`, billing handlers/migrations/frontends | callback signature, amount/currency binding, replay/idempotency, refund approval, entitlement races |
| Persistence | `db.sql`, Commons migrations, service persistence adapters, table ownership manifests | injection, least privilege, tenant enforcement, encryption, backup/restore, retention |
| Runtime/deployment | `docker/`, `deploy/`, `smart-recruit-deploy/`, platform runtime packages | network exposure, TLS, service identity, non-root, capabilities, resources, secrets, observability |
| Supply chain | `go.mod`, `go.sum`, workspace manifests/locks, Dockerfiles, `.github/workflows/` | vulnerable dependencies, pinning, action permissions, SBOM, signing, provenance, secret scanning |

## 3. Mandatory review domains

### A. Architecture and threat model

- Verify asset, actor, entry-point, trust-boundary, and outbound-dependency inventories.
- Model anonymous, authenticated, malicious tenant, compromised admin, compromised service, malicious file, malicious model/tool output, and supply-chain attackers.
- Confirm public, internal, management, health, metrics, debug, and observability surfaces.

### B. Authentication and session management

- Review password hashing and reset/invite/login enumeration controls.
- Verify JWT algorithm pinning plus expiry, issuer, audience, client, tenant, and account-state semantics.
- Test access/refresh lifetimes, hash-only refresh storage, rotation, concurrency, reuse detection, family revocation, logout, password/role changes, and Redis/cache failure behavior.
- Check Cookie `HttpOnly`, `Secure`, `SameSite`, domain/path isolation and header/cookie precedence.
- Review brute-force controls, MFA/step-up needs, audit and alerts.

### C. Authorization, tenant isolation, and business logic

- Build a role × permission × data-scope × endpoint matrix.
- Trace actor and tenant identity from Gateway through gRPC to owner service and persistence predicate.
- Test horizontal and vertical authorization across candidate, interviewer, HR, tenant admin, and platform admin identities.
- Replace IDs in path, query, body, cursor, batch elements, gRPC metadata, events, exports, analytics, and search filters.
- Validate job/application/interview/offer/invite/subscription/refund state machines, concurrency, replay, and idempotency.

### D. Gateway, API, browser, and frontend

- Enumerate routes and middleware ordering; identify missing auth, permission, tenant, body, quota, risk, or timeout controls.
- Review injection, mass assignment, duplicate JSON keys, unknown fields, integer/size bounds, pagination and batch amplification.
- Review CORS allowlists, CSRF defenses, CSP, HSTS, trusted proxies, Host handling, TLS termination, cache control, error minimization, SSE lifecycle, Swagger/metrics/debug exposure.
- Search frontends for `v-html`, `innerHTML`, unsafe URL construction, open redirects, secrets in Vite variables, token persistence, sensitive local/session storage, and source maps.

### E. Internal gRPC, service identity, and MQ

- Require production fail-closed internal authentication and TLS; assess shared-token blast radius and rotation.
- Verify metadata cannot establish identity before caller authentication; prefer workload identity/mTLS for strong service identity.
- Review health/reflection exposure, message limits, deadlines, keepalive, retry, and error details.
- Review RabbitMQ credentials/TLS/vhosts/ACLs, message schema, idempotency, retry limits, poison-message handling, DLQ access, and replay authorization.

### F. Resume, file, parser, and OSS

- Bind presign/confirm to actor, object, size, content type, checksum, expiry, and one-time session.
- Verify extension/MIME/magic validation, filename/object-key normalization, private bucket policy, non-enumerability, and short-lived URLs.
- Test safe fixtures for polyglots, malformed PDF/DOCX, archive/decompression bombs, parser time/memory limits, malware quarantine, and network-isolated non-root parsing.
- Verify deletion and retention across originals, parsed text, embeddings, caches, evidence, and backups.

### G. AI, LLM, Prompt, Memory, Tool, and MCP

- Test direct/indirect prompt injection from messages, resumes, jobs, memory, retrieval, model output, and MCP/tool output.
- Confirm authorization uses server identity/resource facts rather than model-produced IDs or arguments.
- Verify explicit intersection of published release, Agent binding, requested selection, runtime implementation, policy, confirmation, and resource authorization.
- Test MCP URL parsing, redirect validation, DNS resolution/rebinding, loopback/private/link-local/cloud-metadata blocks, stdio command/arg/env allowlists, output limits, redaction, rate limits, and audit.
- Review prompt/model/release immutability, safe fallback, memory isolation, output validation, max rounds/tokens/concurrency/time/cost, and provider error/credential leakage.
- Ensure untrusted model output is validated before HTML, SQL, email, filename, URL, command, or workflow use.

### H. Billing and external integrations

- Validate callback signature before parsing business fields; bind merchant/order/amount/currency/status.
- Reject replay, duplicate settlement, status regression, and client-authoritative entitlement data.
- Verify refund permission, review/approval, amount bounds, idempotency, outbox settlement, reconciliation, and audit.
- Review SMTP/OSS/AI/payment endpoints for SSRF, TLS verification, credential scopes, timeouts, and redacted errors.

### I. Data, privacy, database, logging, and recovery

- Classify candidate, interviewer, HR, compensation, resume, AI evidence, billing, and credential fields.
- Verify minimization, purpose limitation, tenant scoping, encryption in transit/at rest, database least privilege, retention, deletion, export, backup, restore, and production-data isolation.
- Search logs/traces/metrics/audit for tokens, cookies, URLs, resumes, prompt/model bodies, provider errors, emails, phones, salary and payment data.
- Verify audit integrity, access restrictions, correlation, retention, time synchronization, and alerts.
- Require tested recovery objectives and restoration evidence for critical stores.

### J. Secrets, configuration, cryptography, and key lifecycle

- Scan the working tree and Git history without revealing values.
- Reject placeholders, empty/default credentials, insecure-development flags, plaintext private keys, shared production credentials, and secrets in ConfigMaps/images/logs.
- Verify KMS/Vault or equivalent storage, envelope encryption where relevant, environment isolation, least privilege, rotation, revocation, break-glass, and incident replacement.
- Review TLS verification and cryptographic algorithms; do not invent custom cryptography.

### K. Dependencies, CI/CD, images, and provenance

- Run reachability-aware Go checks for every module and dependency checks for every frontend workspace.
- Review SAST, secret scan, license policy, action pinning, workflow permissions, protected environments, artifact integrity, SBOM, image scanning/signing, provenance, immutable digests, and release reproducibility.
- Inspect Docker build context and layers for secrets, dev tools, root runtime, writable filesystem, unnecessary packages, mutable bases, and missing health/runtime controls.

### L. Kubernetes, network, runtime, and observability

- Verify dedicated ServiceAccounts, disabled token automount, non-root UID/GID, no privilege escalation, read-only root filesystem, dropped capabilities, RuntimeDefault seccomp, resource/PID/ephemeral limits, probes, disruption and rollout controls.
- Require default-deny ingress/egress NetworkPolicies with explicit DNS, service, datastore, and third-party egress.
- Review Pod Security Admission, RBAC, Secret encryption/injection, metadata-service blocking, Ingress/WAF/TLS, admin-plane isolation, image admission, node/runtime isolation, and audit logging.
- Treat Compose defaults as local-only unless production safety is independently demonstrated.

### M. Availability, abuse, detection, and incident readiness

- Review distributed rate limits, cache/Redis failure modes, request/body/upload/SSE/gRPC/AI bounds, circuit breakers, retry storms, queue backpressure, dependency timeouts, and cost abuse.
- Verify alerts for auth attacks, authorization denials, bulk access/export, token reuse, secret/config changes, MCP private-network attempts, AI spend spikes, payment anomalies, degraded revocation, and backup failures.
- Require actionable runbooks for account compromise, secret leak, cross-tenant exposure, malicious files, prompt/tool abuse, supply-chain compromise, payment incidents, isolation, evidence preservation, notification, and recovery.

## 4. Validation command catalogue

Use only commands relevant to the current scope and available tooling. Record every result.

```bash
git status --short
git branch --show-current
git rev-parse HEAD
rg --files
rg -n --hidden -S '<targeted-security-patterns>'
```

For each Go module:

```bash
go test ./...
go vet ./...
govulncheck ./...
```

For frontend workspaces:

```bash
pnpm install --frozen-lockfile
pnpm --filter <app> typecheck
pnpm --filter <app> test
pnpm audit
```

Optional, when installed or authorized:

```bash
gitleaks detect --source . --redact
gosec ./...
semgrep scan --config auto
trivy fs .
trivy config .
trivy image <immutable-image-ref>
syft <immutable-image-ref> -o cyclonedx-json
cosign verify <immutable-image-ref>
```

Do not install or download tools silently. Do not claim a clean result for tools that did not run.

## 5. Dynamic-testing authorization levels

| Level | Allowed by default | Examples |
|---|---|---|
| L0 | Local read-only inspection | source/config review, local static scan, existing test execution |
| L1 | Local isolated fixtures | unit/integration negative tests, fake-provider prompt fixtures, parser fixtures with resource bounds |
| L2 | Explicitly authorized staging | authenticated DAST, tenant matrix, controlled uploads, safe SSRF canaries, callback replay tests |
| L3 | Separately authorized high impact | load/stress, brute force, exploit chains, cluster breakout simulation, destructive recovery drills |
| Production | Exact written scope required | only named endpoints/accounts/windows/rates; never infer authorization |

## 6. Completion checklist

- Scope, commit, dirty state, test authorization, standards, and limitations recorded.
- Attack surface and sensitive-data flows documented.
- Every mandatory domain has a coverage verdict and evidence.
- Every finding has severity, confidence, evidence, attack scenario, impact, recommendation, and verification.
- Release blockers map to remediation tasks.
- Every remediation task has bounded scope, acceptance criteria, tests, dependencies, and rollback notes.
- Markdown and HTML reports contain identical verdict and findings.
- No secrets or personal data appear in any artifact.
- The renderer succeeds and all three output files exist.

