-- Enforce one open order and one active Alipay attempt, and persist recovery state.

ALTER TABLE `billing_orders`
  ADD COLUMN `active_slot` TINYINT UNSIGNED NULL AFTER `status`;
UPDATE `billing_orders` SET `active_slot` = 1 WHERE `status` IN ('pending', 'paying');
ALTER TABLE `billing_orders`
  ADD UNIQUE KEY `uk_billing_orders_owner_active` (`owner_type`, `owner_id`, `active_slot`),
  ADD CONSTRAINT `chk_billing_orders_active_slot` CHECK (`active_slot` IS NULL OR `active_slot` = 1);

ALTER TABLE `billing_payments`
  DROP CHECK `chk_billing_payments_status`,
  ADD COLUMN `active_slot` TINYINT UNSIGNED NULL AFTER `status`,
  ADD COLUMN `source_app` VARCHAR(16) NOT NULL DEFAULT 'hr' AFTER `scene`,
  ADD COLUMN `return_token_hash` CHAR(64) DEFAULT NULL AFTER `pay_payload`,
  ADD COLUMN `expires_at` DATETIME(3) DEFAULT NULL AFTER `return_token_hash`,
  ADD COLUMN `next_reconcile_at` DATETIME(3) DEFAULT NULL AFTER `expires_at`,
  ADD COLUMN `reconcile_attempts` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `next_reconcile_at`,
  ADD COLUMN `last_reconcile_error` VARCHAR(500) DEFAULT NULL AFTER `reconcile_attempts`,
  ADD COLUMN `last_queried_at` DATETIME(3) DEFAULT NULL AFTER `last_reconcile_error`,
  ADD CONSTRAINT `chk_billing_payments_status` CHECK (`status` IN ('created', 'pending', 'closing', 'unknown', 'succeeded', 'failed', 'closed', 'refunded')),
  ADD CONSTRAINT `chk_billing_payments_source_app` CHECK (`source_app` IN ('hr', 'candidate'));
UPDATE `billing_payments` payment
JOIN `billing_orders` orders ON orders.id = payment.order_id
SET payment.active_slot = IF(payment.status IN ('created', 'pending', 'closing', 'unknown'), 1, NULL),
    payment.source_app = IF(orders.owner_type = 'user', 'candidate', 'hr'),
    payment.expires_at = orders.expires_at,
    payment.next_reconcile_at = IF(payment.status IN ('created', 'pending', 'closing', 'unknown'), UTC_TIMESTAMP(3), NULL),
    payment.pay_payload = NULL;
ALTER TABLE `billing_payments`
  ADD UNIQUE KEY `uk_billing_payments_order_active` (`order_id`, `active_slot`),
  ADD KEY `idx_billing_payments_reconcile` (`payment_environment`, `next_reconcile_at`, `status`),
  ADD UNIQUE KEY `uk_billing_payments_return_token` (`return_token_hash`);
ALTER TABLE `billing_payments`
  ADD CONSTRAINT `chk_billing_payments_active_slot` CHECK (`active_slot` IS NULL OR `active_slot` = 1);

ALTER TABLE `billing_refunds`
  DROP CHECK `chk_billing_refunds_status`,
  ADD COLUMN `idempotency_key` VARCHAR(128) DEFAULT NULL AFTER `refund_no`,
  ADD COLUMN `active_slot` TINYINT UNSIGNED NULL AFTER `status`,
  ADD COLUMN `next_reconcile_at` DATETIME(3) DEFAULT NULL AFTER `succeeded_at`,
  ADD COLUMN `reconcile_attempts` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `next_reconcile_at`,
  ADD COLUMN `last_error` VARCHAR(500) DEFAULT NULL AFTER `reconcile_attempts`,
  ADD COLUMN `rejected_reason` VARCHAR(500) DEFAULT NULL AFTER `last_error`,
  ADD CONSTRAINT `chk_billing_refunds_status` CHECK (`status` IN ('requested', 'reviewing', 'approved', 'processing', 'unknown', 'succeeded', 'failed', 'rejected'));
UPDATE `billing_refunds`
SET `idempotency_key` = `refund_no`,
    `active_slot` = IF(`status` IN ('requested', 'reviewing', 'approved', 'processing', 'unknown'), 1, NULL),
    `next_reconcile_at` = IF(`status` IN ('processing', 'unknown'), UTC_TIMESTAMP(3), NULL);
ALTER TABLE `billing_refunds`
  MODIFY COLUMN `idempotency_key` VARCHAR(128) NOT NULL,
  ADD UNIQUE KEY `uk_billing_refunds_order_idempotency` (`order_id`, `idempotency_key`),
  ADD UNIQUE KEY `uk_billing_refunds_payment_active` (`payment_id`, `active_slot`),
  ADD KEY `idx_billing_refunds_reconcile` (`next_reconcile_at`, `status`);
ALTER TABLE `billing_refunds`
  ADD CONSTRAINT `chk_billing_refunds_active_slot` CHECK (`active_slot` IS NULL OR `active_slot` = 1);

ALTER TABLE `billing_webhook_events`
  ADD COLUMN `retry_count` INT UNSIGNED NOT NULL DEFAULT 0 AFTER `failure_reason`,
  ADD COLUMN `last_attempt_at` DATETIME(3) DEFAULT NULL AFTER `retry_count`;
