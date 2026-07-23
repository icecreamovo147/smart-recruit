# Security Fix Verification Checklist

## Root cause and regression

- Confirm the original attack/failure condition before editing when safe.
- Fix the authorization, validation, trust-boundary, lifecycle, configuration, or isolation root cause.
- Add a negative regression test that fails before the fix and passes after it when practical.
- Preserve valid business behavior and error semantics.
- Check adjacent batch, export, search, internal RPC, event, async worker, and degraded/fallback paths.

## Authentication and authorization

- Revalidate current server-side principal, account status, tenant, membership, permission, data scope, and resource ownership.
- Do not trust client IDs, model/tool arguments, forwarded metadata, event payload identity, or cached claims without the owning boundary.
- Fail closed when required identity/authorization dependencies are unavailable unless an explicitly documented availability policy says otherwise.
- Test horizontal, vertical, cross-tenant, stale-token, and role-change cases.

## Input, files, SSRF, and output

- Validate structure, size, type, count, encoding, path, URL, redirect, resolved IP, and resource ownership at the consuming boundary.
- Preserve body/upload/stream/tool round/time/concurrency limits.
- Validate model/tool/file/parser output before HTML, SQL, command, path, email, URL, workflow, or persistence use.
- Do not weaken private-network, cloud-metadata, command, environment, or transport allowlists.

## Secrets and privacy

- Do not add credentials, tokens, private keys, production identifiers, personal data, resumes, prompts, provider bodies, or presigned URLs to code, fixtures, logs, traces, reports, or errors.
- Keep returned credentials masked and logs allowlist-based.
- Check new error paths, metrics labels, audit records, and test snapshots for disclosure.

## AI, Tool, and MCP

- Intersect release entitlement, Agent binding, explicit user selection, runtime implementation, policy, confirmation, and resource authorization.
- Treat prompt, memory, retrieval, resume, model output, Tool output, and MCP output as untrusted data.
- Preserve same-release model fallback, immutable release evidence, max rounds/tokens/time/cost, redaction, and audit.
- Test prompt injection, argument substitution, empty selection, disabled binding, private-network target, redirect, DNS resolution, and unavailable-runner cases.

## Billing and lifecycle

- Bind signatures to the raw callback and validate merchant/order/amount/currency/status.
- Preserve replay protection, idempotency, monotonic state, refund approval, outbox/inbox semantics, and entitlement consistency.
- Test duplicate, reordered, concurrent, partial-failure, and retry cases.

## Deployment and supply chain

- Avoid insecure development flags, default credentials, public management ports, unverified TLS, broad egress, root/privileged runtime, mutable image tags, and excessive workflow permissions.
- Do not update dependencies, images, Actions, lockfiles, or provenance configuration outside approved scope.
- Re-run relevant secret, vulnerability, SAST, image, and IaC checks.

## Completion

- Run task-specific tests and impacted repository checks.
- Run the original security verification.
- Run `git diff --check`.
- Validate changed paths against task scope.
- Review the final diff for disabled controls, fail-open behavior, swallowed errors, broad permissions, sensitive logging, and missing negative tests.
- Record anything not run as `NOT_RUN` or `BLOCKED`.

