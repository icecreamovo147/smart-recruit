---
schema_version: 1
id: ai-credit-lifecycle
title: AI credit reservation and settlement lifecycle
kind: domain
status: active
owners:
  - engineering-platform
tags:
  - ai
  - billing
  - credits
  - outbox
applies_to:
  - smart-recruit-ai-agent-service/**
  - smart-recruit-billing-service/**
  - smart-recruit-commons/migrations/000069_add_ai_billing_foundation.sql
  - smart-recruit-commons/migrations/000079_add_ai_billing_settlement_outbox.sql
  - smart-recruit-commons/migrations/000088_harden_memory_billing_integrity.sql
  - smart-recruit-gateway/handler/response.go
source_refs:
  - smart-recruit-ai-agent-service/internal/interfaces/grpc/billing_meter.go
  - smart-recruit-ai-agent-service/internal/infrastructure/persistence/billing_settlement_outbox.go
  - smart-recruit-ai-agent-service/internal/application/recruiting_intelligence/structured_runtime.go
  - smart-recruit-billing-service/internal/application/service/billing.go
  - smart-recruit-billing-service/internal/infrastructure/persistence/gorm_repository.go
  - smart-recruit-billing-service/internal/infrastructure/persistence/entitlement_policy.go
  - smart-recruit-commons/migrations/000079_add_ai_billing_settlement_outbox.sql
  - smart-recruit-commons/migrations/000088_harden_memory_billing_integrity.sql
  - smart-recruit-deploy/mysql-table-ownership.json
last_verified: 2026-07-23
review_after: 2026-10-19
---

# AI Credit Reservation and Settlement Lifecycle

Billing owns entitlements, price snapshots, credit grants, reservations, rate cards, usage events, and the append-only credit ledger. AI Agent owns provider invocation and the durable delivery state that connects a completed invocation to Billing settlement or cancellation.

## Runtime contract

`AI_BILLING_MODE` is the process-wide rollout switch. Billing applies it as an explicit override even when a YAML configuration file is supplied; AI Agent uses the same value to decide whether Billing failures must fail closed. In `enforce`, Billing refuses startup without a published effective rate card and a credit source, and AI Agent rejects a Billing response that reports a different enforcement mode.

Rate-card provider/model keys are selected from enabled `llm_providers` and `llm_models` records. Billing validates the pair again on write and persists the catalog's canonical spelling, so a disabled, renamed, or manually forged runtime target cannot be published through a direct API call.

Platform commercial-control saves for product prices and AI rate cards publish the new immutable version immediately; the previous published version for the same product term or provider/model is retired in the same transaction. Billing binds one `Asia/Shanghai` application clock value for publication and retirement mutations, while entitlement/storefront queries run through a verified MySQL `+08:00` session. Billing `DATETIME` values are therefore UTC+8 wall-clock values; month boundaries retain the Shanghai location instead of being converted to UTC wall time.

JSON entitlement booleans are decoded as their JSON text (`true`/`false`) rather than numerically cast by MySQL. Candidate subscription snapshots must bind `ai.chat.release_version_id` to the published `candidate` capability release; an HR release ID is not interchangeable even when the entitlement key is the same.

Before a provider call, AI Agent checks the capability and requests a reservation with an estimated value of zero. Billing resolves the effective `ai.single_run.max_credits` entitlement, falling back to 20 credits only for legacy price snapshots without that key. In enforce mode it reserves the lesser of that ceiling and the available prepaid balance, rejects zero balance, verifies that the exact provider/model has an effective rate card, and creates exact earliest-expiry Grant allocations before allowing the call.

After reservation, AI Agent durably creates an `ai_billing_settlement_outbox` row before calling the provider. Provider token usage is retained for chat, Agent runs, application analysis, resume parsing, and match evaluation. Deterministic paths that do not invoke a provider cancel the reservation and do not consume credits.

The settlement outbox is owner-polymorphic (`owner_type` plus `owner_id`) and intentionally does not participate in the GORM `tenant_id` boundary plugin. Tenant and candidate isolation is carried by the reservation foreign key and owner identity; background settlement must be able to process both owner types without a request tenant context.

Candidate chat persists the User turn before billing admission and persists a structured failed Assistant turn when model resolution or credit reservation fails. The failure metadata lives in `process_content` (`delivery_status`, `error_code`, and `retryable`), so refreshing a session preserves the actionable failure state without treating it as model output. Exhausted grants remain part of the current-period gross/used summary until expiry even though they no longer contribute available balance.

Credit-pack purchases add independent, expiring grant buckets and do not replace the owner's base monthly subscription. Account summaries evaluate grant validity with the same bound application clock used by reservation and balance queries, which keeps locally parsed MySQL `DATETIME` values consistent across available, total, and used credit fields. Candidate UI presents an additional-credit marker beside the base plan when current grant totals exceed the plan's included monthly credits.

Settlement first persists the complete Billing request in the outbox, then calls Billing synchronously. Failures remain pending and are retried with bounded exponential backoff; stale processing leases are recovered. Billing settlement and cancellation are idempotent. Rows that exhaust retry attempts become `dead` for alerting and manual repair. Billing maintenance does not expire reservations that still have reserved, pending, processing, or dead delivery evidence; an old `reserved` row may represent a crash around provider invocation and requires explicit reconciliation.

## Accounting invariants

- Only `enforce` creates `ai_credit_reservation_allocations` and mutates grant balances; `shadow` records reservations, usage, supplier cost, and calculated credits for calibration.
- Every active enforce reservation is backed by allocations whose open credits exactly equal `reserved_credits`. Settlement may consume only those Grants, releases the unused allocation, and fails the whole transaction if backing is incomplete.
- Cached prompt tokens are a subset of input tokens. Pricing charges `(input-cached)` at the normal input rate, cached tokens at the cached rate, and output tokens at the output rate with checked arithmetic.
- `(reservation_id, provider_call_seq)` and owner idempotency keys prevent duplicate usage and balance mutation.
- Multi-call structured operations preserve every successful provider call with a contiguous call sequence.
- Candidate credits use `owner_type=user`; HR and enterprise credits use `owner_type=tenant`.

## Failure semantics

Missing Billing, mode mismatch, missing outbox persistence, missing exact rate card, disabled capability, or zero balance fails before provider invocation in enforce mode. Gateway maps `insufficient_credits` to business code `40201`; candidate and HR clients show purchase guidance. A provider call whose settlement RPC fails is not treated as unbilled success: the durable outbox retains the exact request until delivery or explicit dead-letter repair.

## Verification

Verified against the allocation-backed reservation/settlement implementation, checked rate-card pricing, outbox delivery, migrations, and gateway contracts on 2026-07-23.
