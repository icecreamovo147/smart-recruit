ALTER TABLE `billing_webhook_events`
  DROP COLUMN `last_attempt_at`,
  DROP COLUMN `retry_count`;

ALTER TABLE `billing_refunds`
  DROP CHECK `chk_billing_refunds_active_slot`,
  DROP INDEX `idx_billing_refunds_reconcile`,
  DROP INDEX `uk_billing_refunds_payment_active`,
  DROP INDEX `uk_billing_refunds_order_idempotency`,
  DROP CHECK `chk_billing_refunds_status`,
  DROP COLUMN `rejected_reason`,
  DROP COLUMN `last_error`,
  DROP COLUMN `reconcile_attempts`,
  DROP COLUMN `next_reconcile_at`,
  DROP COLUMN `active_slot`,
  DROP COLUMN `idempotency_key`,
  ADD CONSTRAINT `chk_billing_refunds_status` CHECK (`status` IN ('requested', 'reviewing', 'processing', 'succeeded', 'failed', 'rejected'));

ALTER TABLE `billing_payments`
  DROP CHECK `chk_billing_payments_active_slot`,
  DROP INDEX `uk_billing_payments_return_token`,
  DROP INDEX `idx_billing_payments_reconcile`,
  DROP INDEX `uk_billing_payments_order_active`,
  DROP CHECK `chk_billing_payments_source_app`,
  DROP CHECK `chk_billing_payments_status`,
  DROP COLUMN `last_queried_at`,
  DROP COLUMN `last_reconcile_error`,
  DROP COLUMN `reconcile_attempts`,
  DROP COLUMN `next_reconcile_at`,
  DROP COLUMN `expires_at`,
  DROP COLUMN `return_token_hash`,
  DROP COLUMN `source_app`,
  DROP COLUMN `active_slot`,
  ADD CONSTRAINT `chk_billing_payments_status` CHECK (`status` IN ('created', 'pending', 'succeeded', 'failed', 'closed', 'refunded'));

ALTER TABLE `billing_orders`
  DROP CHECK `chk_billing_orders_active_slot`,
  DROP INDEX `uk_billing_orders_owner_active`,
  DROP COLUMN `active_slot`;
