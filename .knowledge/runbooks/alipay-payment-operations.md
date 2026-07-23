---
schema_version: 1
id: alipay-payment-operations
title: Alipay payment operations and recovery
kind: runbook
status: active
owners:
  - engineering-platform
tags:
  - billing
  - alipay
  - payment
  - refund
applies_to:
  - smart-recruit-billing-service/**
  - smart-recruit-gateway/handler/billing.go
  - smart-recruit-commons/migrations/000076_harden_alipay_payment_lifecycle.sql
  - smart-recruit-commons/migrations/000077_add_billing_refund_review_permission.sql
  - smart-recruit-commons/migrations/000088_harden_memory_billing_integrity.sql
  - docs/ai-billing-alipay-sandbox.md
  - smart-recruit-deploy/observability/rules/billing-alerts.yml
source_refs:
  - smart-recruit-billing-service/internal/infrastructure/payment/alipay.go
  - smart-recruit-billing-service/internal/application/service/commerce.go
  - smart-recruit-billing-service/internal/interfaces/grpc/server.go
  - smart-recruit-gateway/handler/billing.go
  - smart-recruit-commons/migrations/000076_harden_alipay_payment_lifecycle.sql
  - smart-recruit-commons/migrations/000088_harden_memory_billing_integrity.sql
  - docs/ai-billing-alipay-sandbox.md
  - smart-recruit-deploy/observability/rules/billing-alerts.yml
last_verified: 2026-07-23
review_after: 2026-10-18
---

# Alipay Payment Operations and Recovery

Billing Service owns payment, notification, reconciliation, and refund state. Gateway only parses HTTP, applies authentication/authorization, returns the exact Alipay acknowledgement, and routes signed browser returns.

## Key verification

`ALIPAY_PRIVATE_KEY` signs application requests. `ALIPAY_VERIFY_PUBLIC_KEY` verifies Alipay notifications and API responses. They must not form the same RSA key pair. A configured private key file must use mode `0600`.

When callbacks fail, inspect `billing_webhook_events.status`, `signature_verified`, `failure_reason`, and `retry_count`. Never copy signatures, buyer identifiers, private keys, or full cashier URLs into tickets or logs.

## Payment recovery

An owner has one open order (`billing_orders.active_slot=1`) and an order has one active payment (`billing_payments.active_slot=1`). Repeated pay requests reuse the active `out_trade_no`. `pending`, `closing`, and `unknown` payments are claimed by `next_reconcile_at` with `FOR UPDATE SKIP LOCKED`; network ambiguity is retried with backoff. A local order is not closed until Alipay reports missing/closed or a close operation succeeds.

Browser return is not settlement evidence. Gateway verifies the signed return, Billing Service rotates `return_token_hash`, and the authenticated owner uses that token to query the exact payment.

## Refund recovery

Refund requests are unique by `(order_id, idempotency_key)` and active by `(payment_id, active_slot)`. The same `refund_no` is always used as Alipay `out_request_no`. A timeout becomes `unknown` and is recovered through `alipay.trade.fastpay.refund.query`; never create a replacement refund number.

Refund creation first changes order-sourced Grants to `refund_frozen`, so new AI reservations cannot race the Alipay call. Existing Grant allocations move the refund to `waiting_usage`; after they settle or cancel, exact Grant ledger consumption determines whether the request can proceed automatically or requires manual review. `unknown` keeps Grants frozen. API rejection, reconciliation failure, or manual rejection restores still-valid balances; confirmed success revokes the remaining source Grants through the idempotent refund finalizer.

Manual refunds are reviewed by platform users with `platform.billing.refund.review`. Approval is blocked while source Grants have open allocations. Rejecting restores the order to `paid`; approving submits the existing refund number.

## Verification

Run Billing and Gateway Go tests, frontend typechecks, Commons migration tests, pinned protobuf generation/sync checks, and the sandbox matrix documented in `docs/ai-billing-alipay-sandbox.md`.
