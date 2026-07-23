-- AI billing foundation for tenant and candidate owners, including sandbox-isolated payments.

CREATE TABLE IF NOT EXISTS `billing_products` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `product_key` VARCHAR(64) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `description` VARCHAR(500) DEFAULT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `product_type` VARCHAR(24) NOT NULL DEFAULT 'subscription',
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_products_key` (`product_key`),
  KEY `idx_billing_products_owner_status` (`owner_type`, `status`),
  CONSTRAINT `chk_billing_products_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_billing_products_type` CHECK (`product_type` IN ('subscription', 'credit_pack')),
  CONSTRAINT `chk_billing_products_status` CHECK (`status` IN ('draft', 'active', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Billing product catalogue';

CREATE TABLE IF NOT EXISTS `billing_price_versions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `product_id` BIGINT UNSIGNED NOT NULL,
  `version` INT UNSIGNED NOT NULL,
  `billing_term` VARCHAR(16) NOT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `included_credits` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `entitlement_snapshot` JSON DEFAULT NULL,
  `platform_plan_version_id` BIGINT UNSIGNED DEFAULT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `effective_at` DATETIME(3) DEFAULT NULL,
  `retired_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_price_versions_product_version` (`product_id`, `version`),
  KEY `idx_billing_price_versions_status_effective` (`status`, `effective_at`),
  KEY `idx_billing_price_versions_platform_plan` (`platform_plan_version_id`),
  CONSTRAINT `fk_billing_price_versions_product` FOREIGN KEY (`product_id`) REFERENCES `billing_products` (`id`),
  CONSTRAINT `fk_billing_price_versions_platform_plan` FOREIGN KEY (`platform_plan_version_id`) REFERENCES `platform_plan_versions` (`id`),
  CONSTRAINT `chk_billing_price_versions_term` CHECK (`billing_term` IN ('monthly', 'yearly', 'one_time')),
  CONSTRAINT `chk_billing_price_versions_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_billing_price_versions_status` CHECK (`status` IN ('draft', 'published', 'retired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable billing price versions';

CREATE TABLE IF NOT EXISTS `billing_subscriptions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `product_id` BIGINT UNSIGNED NOT NULL,
  `price_version_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `term` VARCHAR(16) NOT NULL,
  `current_period_start` DATETIME(3) NOT NULL,
  `current_period_end` DATETIME(3) NOT NULL,
  `next_price_version_id` BIGINT UNSIGNED DEFAULT NULL,
  `next_term` VARCHAR(16) DEFAULT NULL,
  `cancel_at_period_end` TINYINT(1) NOT NULL DEFAULT 0,
  `activated_by_order_id` BIGINT UNSIGNED DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_billing_subscriptions_owner_status` (`owner_type`, `owner_id`, `status`),
  KEY `idx_billing_subscriptions_period_end` (`status`, `current_period_end`),
  CONSTRAINT `fk_billing_subscriptions_product` FOREIGN KEY (`product_id`) REFERENCES `billing_products` (`id`),
  CONSTRAINT `fk_billing_subscriptions_price` FOREIGN KEY (`price_version_id`) REFERENCES `billing_price_versions` (`id`),
  CONSTRAINT `fk_billing_subscriptions_next_price` FOREIGN KEY (`next_price_version_id`) REFERENCES `billing_price_versions` (`id`),
  CONSTRAINT `chk_billing_subscriptions_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_billing_subscriptions_status` CHECK (`status` IN ('pending', 'active', 'past_due', 'expired', 'cancelled')),
  CONSTRAINT `chk_billing_subscriptions_term` CHECK (`term` IN ('monthly', 'yearly')),
  CONSTRAINT `chk_billing_subscriptions_next_term` CHECK (`next_term` IS NULL OR `next_term` IN ('monthly', 'yearly')),
  CONSTRAINT `chk_billing_subscriptions_period` CHECK (`current_period_end` > `current_period_start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant and candidate billing subscriptions';

CREATE TABLE IF NOT EXISTS `billing_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_no` VARCHAR(64) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `order_type` VARCHAR(24) NOT NULL,
  `product_id` BIGINT UNSIGNED NOT NULL,
  `price_version_id` BIGINT UNSIGNED NOT NULL,
  `subscription_id` BIGINT UNSIGNED DEFAULT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `payment_environment` VARCHAR(16) NOT NULL DEFAULT 'sandbox',
  `idempotency_key` VARCHAR(128) NOT NULL,
  `expires_at` DATETIME(3) NOT NULL,
  `paid_at` DATETIME(3) DEFAULT NULL,
  `closed_at` DATETIME(3) DEFAULT NULL,
  `metadata` JSON DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_orders_no` (`order_no`),
  UNIQUE KEY `uk_billing_orders_owner_idempotency` (`owner_type`, `owner_id`, `idempotency_key`),
  KEY `idx_billing_orders_owner_created` (`owner_type`, `owner_id`, `created_at`),
  KEY `idx_billing_orders_status_expiry` (`status`, `expires_at`),
  CONSTRAINT `fk_billing_orders_product` FOREIGN KEY (`product_id`) REFERENCES `billing_products` (`id`),
  CONSTRAINT `fk_billing_orders_price` FOREIGN KEY (`price_version_id`) REFERENCES `billing_price_versions` (`id`),
  CONSTRAINT `fk_billing_orders_subscription` FOREIGN KEY (`subscription_id`) REFERENCES `billing_subscriptions` (`id`),
  CONSTRAINT `chk_billing_orders_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_billing_orders_type` CHECK (`order_type` IN ('subscribe', 'renew', 'upgrade', 'credit_pack')),
  CONSTRAINT `chk_billing_orders_status` CHECK (`status` IN ('pending', 'paying', 'paid', 'closed', 'refunding', 'refunded')),
  CONSTRAINT `chk_billing_orders_environment` CHECK (`payment_environment` IN ('sandbox', 'production')),
  CONSTRAINT `chk_billing_orders_currency` CHECK (`currency` = 'CNY')
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Billing orders with payment-environment isolation';

ALTER TABLE `billing_subscriptions`
  ADD CONSTRAINT `fk_billing_subscriptions_activation_order`
  FOREIGN KEY (`activated_by_order_id`) REFERENCES `billing_orders` (`id`);

CREATE TABLE IF NOT EXISTS `billing_payments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `payment_no` VARCHAR(64) NOT NULL,
  `channel` VARCHAR(24) NOT NULL DEFAULT 'alipay',
  `scene` VARCHAR(16) NOT NULL,
  `payment_environment` VARCHAR(16) NOT NULL DEFAULT 'sandbox',
  `channel_trade_no` VARCHAR(128) DEFAULT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `status` VARCHAR(24) NOT NULL DEFAULT 'created',
  `pay_payload` MEDIUMTEXT DEFAULT NULL,
  `paid_at` DATETIME(3) DEFAULT NULL,
  `closed_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_payments_no` (`payment_no`),
  UNIQUE KEY `uk_billing_payments_channel_trade` (`channel`, `payment_environment`, `channel_trade_no`),
  KEY `idx_billing_payments_order_status` (`order_id`, `status`),
  CONSTRAINT `fk_billing_payments_order` FOREIGN KEY (`order_id`) REFERENCES `billing_orders` (`id`),
  CONSTRAINT `chk_billing_payments_channel` CHECK (`channel` = 'alipay'),
  CONSTRAINT `chk_billing_payments_scene` CHECK (`scene` IN ('desktop', 'wap')),
  CONSTRAINT `chk_billing_payments_environment` CHECK (`payment_environment` IN ('sandbox', 'production')),
  CONSTRAINT `chk_billing_payments_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_billing_payments_status` CHECK (`status` IN ('created', 'pending', 'succeeded', 'failed', 'closed', 'refunded'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Payment attempts; sandbox and production never mix';

CREATE TABLE IF NOT EXISTS `billing_refunds` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `refund_no` VARCHAR(64) NOT NULL,
  `order_id` BIGINT UNSIGNED NOT NULL,
  `payment_id` BIGINT UNSIGNED NOT NULL,
  `amount_fen` BIGINT UNSIGNED NOT NULL,
  `reason` VARCHAR(500) NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'requested',
  `review_mode` VARCHAR(16) NOT NULL DEFAULT 'automatic',
  `channel_refund_no` VARCHAR(128) DEFAULT NULL,
  `requested_by` BIGINT UNSIGNED NOT NULL,
  `reviewed_by` BIGINT UNSIGNED DEFAULT NULL,
  `reviewed_at` DATETIME(3) DEFAULT NULL,
  `succeeded_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_refunds_no` (`refund_no`),
  KEY `idx_billing_refunds_order_status` (`order_id`, `status`),
  CONSTRAINT `fk_billing_refunds_order` FOREIGN KEY (`order_id`) REFERENCES `billing_orders` (`id`),
  CONSTRAINT `fk_billing_refunds_payment` FOREIGN KEY (`payment_id`) REFERENCES `billing_payments` (`id`),
  CONSTRAINT `chk_billing_refunds_status` CHECK (`status` IN ('requested', 'reviewing', 'processing', 'succeeded', 'failed', 'rejected')),
  CONSTRAINT `chk_billing_refunds_review` CHECK (`review_mode` IN ('automatic', 'manual'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Refund requests and channel results';

CREATE TABLE IF NOT EXISTS `billing_webhook_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `channel` VARCHAR(24) NOT NULL DEFAULT 'alipay',
  `payment_environment` VARCHAR(16) NOT NULL DEFAULT 'sandbox',
  `event_key` VARCHAR(191) NOT NULL,
  `event_type` VARCHAR(64) NOT NULL,
  `signature_verified` TINYINT(1) NOT NULL DEFAULT 0,
  `payload_sha256` CHAR(64) NOT NULL,
  `payload` JSON NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'received',
  `failure_reason` VARCHAR(500) DEFAULT NULL,
  `processed_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_webhook_events_key` (`channel`, `payment_environment`, `event_key`),
  KEY `idx_billing_webhook_events_status_created` (`status`, `created_at`),
  CONSTRAINT `chk_billing_webhook_events_channel` CHECK (`channel` = 'alipay'),
  CONSTRAINT `chk_billing_webhook_events_environment` CHECK (`payment_environment` IN ('sandbox', 'production')),
  CONSTRAINT `chk_billing_webhook_events_status` CHECK (`status` IN ('received', 'verified', 'processed', 'ignored', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Idempotent payment callback inbox';

CREATE TABLE IF NOT EXISTS `ai_rate_cards` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `version` INT UNSIGNED NOT NULL,
  `currency` CHAR(3) NOT NULL DEFAULT 'CNY',
  `input_micros_per_1k_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `output_micros_per_1k_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `cached_input_micros_per_1k_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `credit_micros` BIGINT UNSIGNED NOT NULL DEFAULT 1000000,
  `status` VARCHAR(16) NOT NULL DEFAULT 'draft',
  `effective_at` DATETIME(3) DEFAULT NULL,
  `retired_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_rate_cards_model_version` (`provider_key`, `model_key`, `version`),
  KEY `idx_ai_rate_cards_effective` (`provider_key`, `model_key`, `status`, `effective_at`),
  CONSTRAINT `chk_ai_rate_cards_status` CHECK (`status` IN ('draft', 'published', 'retired')),
  CONSTRAINT `chk_ai_rate_cards_currency` CHECK (`currency` = 'CNY'),
  CONSTRAINT `chk_ai_rate_cards_credit_micros` CHECK (`credit_micros` > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable AI supplier cost and credit conversion rates';

CREATE TABLE IF NOT EXISTS `ai_usage_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `tenant_id` BIGINT UNSIGNED DEFAULT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `reservation_id` BIGINT UNSIGNED DEFAULT NULL,
  `operation` VARCHAR(64) NOT NULL,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `provider_call_seq` INT UNSIGNED NOT NULL DEFAULT 1,
  `input_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `output_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `cached_input_tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `supplier_cost_micros` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `credits_charged` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `usage_source` VARCHAR(16) NOT NULL DEFAULT 'provider',
  `provider_request_id` VARCHAR(191) DEFAULT NULL,
  `occurred_at` DATETIME(3) NOT NULL,
  `metadata` JSON DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_usage_events_event_id` (`event_id`),
  UNIQUE KEY `uk_ai_usage_events_provider_call` (`reservation_id`, `provider_call_seq`),
  KEY `idx_ai_usage_events_owner_occurred` (`owner_type`, `owner_id`, `occurred_at`),
  KEY `idx_ai_usage_events_tenant_occurred` (`tenant_id`, `occurred_at`),
  KEY `idx_ai_usage_events_provider_model` (`provider_key`, `model_key`, `occurred_at`),
  CONSTRAINT `chk_ai_usage_events_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_usage_events_owner_tenant` CHECK ((`owner_type` = 'tenant' AND `tenant_id` = `owner_id`) OR (`owner_type` = 'user')),
  CONSTRAINT `chk_ai_usage_events_source` CHECK (`usage_source` IN ('provider', 'estimated'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable billable AI usage facts';

CREATE TABLE IF NOT EXISTS `ai_credit_grants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `grant_type` VARCHAR(24) NOT NULL,
  `source_type` VARCHAR(24) NOT NULL,
  `source_id` BIGINT UNSIGNED DEFAULT NULL,
  `total_credits` BIGINT UNSIGNED NOT NULL,
  `remaining_credits` BIGINT UNSIGNED NOT NULL,
  `valid_from` DATETIME(3) NOT NULL,
  `expires_at` DATETIME(3) DEFAULT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_grants_source` (`owner_type`, `owner_id`, `source_type`, `source_id`, `valid_from`),
  KEY `idx_ai_credit_grants_consume` (`owner_type`, `owner_id`, `status`, `expires_at`, `valid_from`),
  CONSTRAINT `chk_ai_credit_grants_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_credit_grants_type` CHECK (`grant_type` IN ('free_monthly', 'subscription_monthly', 'credit_pack', 'manual_adjustment')),
  CONSTRAINT `chk_ai_credit_grants_source` CHECK (`source_type` IN ('subscription', 'order', 'manual', 'system')),
  CONSTRAINT `chk_ai_credit_grants_balance` CHECK (`remaining_credits` <= `total_credits`),
  CONSTRAINT `chk_ai_credit_grants_status` CHECK (`status` IN ('active', 'exhausted', 'expired', 'revoked'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Expiring AI credit buckets consumed earliest-expiry first';

CREATE TABLE IF NOT EXISTS `ai_credit_reservations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `reservation_no` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `capability` VARCHAR(96) NOT NULL,
  `operation` VARCHAR(64) NOT NULL,
  `provider_key` VARCHAR(64) NOT NULL,
  `model_key` VARCHAR(128) NOT NULL,
  `idempotency_key` VARCHAR(191) NOT NULL,
  `reserved_credits` BIGINT UNSIGNED NOT NULL,
  `settled_credits` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `enforcement_mode` VARCHAR(16) NOT NULL DEFAULT 'shadow',
  `expires_at` DATETIME(3) NOT NULL,
  `settled_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_reservations_no` (`reservation_no`),
  UNIQUE KEY `uk_ai_credit_reservations_owner_idempotency` (`owner_type`, `owner_id`, `idempotency_key`),
  KEY `idx_ai_credit_reservations_expiry` (`status`, `expires_at`),
  CONSTRAINT `chk_ai_credit_reservations_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_credit_reservations_status` CHECK (`status` IN ('active', 'settled', 'cancelled', 'expired')),
  CONSTRAINT `chk_ai_credit_reservations_mode` CHECK (`enforcement_mode` IN ('shadow', 'enforce'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Idempotent pre-provider AI credit reservations';

ALTER TABLE `ai_usage_events`
  ADD CONSTRAINT `fk_ai_usage_events_reservation`
  FOREIGN KEY (`reservation_id`) REFERENCES `ai_credit_reservations` (`id`);

CREATE TABLE IF NOT EXISTS `ai_credit_ledger` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `entry_id` CHAR(36) NOT NULL,
  `owner_type` VARCHAR(16) NOT NULL,
  `owner_id` BIGINT UNSIGNED NOT NULL,
  `grant_id` BIGINT UNSIGNED DEFAULT NULL,
  `reservation_id` BIGINT UNSIGNED DEFAULT NULL,
  `usage_event_id` BIGINT UNSIGNED DEFAULT NULL,
  `entry_type` VARCHAR(24) NOT NULL,
  `credits_delta` BIGINT NOT NULL,
  `balance_after` BIGINT NOT NULL,
  `idempotency_key` VARCHAR(191) NOT NULL,
  `description` VARCHAR(500) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_credit_ledger_entry_id` (`entry_id`),
  UNIQUE KEY `uk_ai_credit_ledger_owner_idempotency` (`owner_type`, `owner_id`, `idempotency_key`),
  KEY `idx_ai_credit_ledger_owner_created` (`owner_type`, `owner_id`, `created_at`),
  CONSTRAINT `fk_ai_credit_ledger_grant` FOREIGN KEY (`grant_id`) REFERENCES `ai_credit_grants` (`id`),
  CONSTRAINT `fk_ai_credit_ledger_reservation` FOREIGN KEY (`reservation_id`) REFERENCES `ai_credit_reservations` (`id`),
  CONSTRAINT `fk_ai_credit_ledger_usage` FOREIGN KEY (`usage_event_id`) REFERENCES `ai_usage_events` (`id`),
  CONSTRAINT `chk_ai_credit_ledger_owner` CHECK (`owner_type` IN ('tenant', 'user')),
  CONSTRAINT `chk_ai_credit_ledger_type` CHECK (`entry_type` IN ('grant', 'reserve', 'release', 'consume', 'expire', 'refund', 'adjustment'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Append-only AI credit accounting ledger';

-- Existing tenants stay grandfathered on V1 until explicitly migrated. V2 adds
-- observable AI entitlements and can be assigned during the paid pilot.
INSERT INTO `platform_plan_versions` (`plan_id`, `version`, `status`, `effective_at`, `change_note`)
SELECT `id`, 2, 'published', NOW(3), 'AI billing shadow defaults; review before enforcement'
FROM `platform_plans`
WHERE `plan_key` IN ('starter', 'growth', 'enterprise')
ON DUPLICATE KEY UPDATE `plan_id` = VALUES(`plan_id`);

INSERT INTO `platform_plan_entitlements` (`plan_version_id`, `entitlement_key`, `value_type`, `value_json`, `enforcement_mode`)
SELECT v2.id, v1e.entitlement_key, v1e.value_type, v1e.value_json, v1e.enforcement_mode
FROM `platform_plan_versions` v2
JOIN `platform_plan_versions` v1 ON v1.plan_id = v2.plan_id AND v1.version = 1
JOIN `platform_plan_entitlements` v1e ON v1e.plan_version_id = v1.id
WHERE v2.version = 2
ON DUPLICATE KEY UPDATE `plan_version_id` = VALUES(`plan_version_id`);

INSERT INTO `platform_plan_entitlements` (`plan_version_id`, `entitlement_key`, `value_type`, `value_json`, `enforcement_mode`)
SELECT version.id, entitlement.entitlement_key, entitlement.value_type, entitlement.value_json, 'observe'
FROM `platform_plan_versions` version
JOIN `platform_plans` plan ON plan.id = version.plan_id AND version.version = 2
JOIN (
  SELECT 'starter' plan_key, 'ai.hr.enabled' entitlement_key, 'boolean' value_type, CAST('true' AS JSON) value_json UNION ALL
  SELECT 'starter', 'ai.chat.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.resume_parse.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.match_evaluation.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.application_analysis.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.agent_run.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'starter', 'ai.credits.monthly', 'integer', CAST(100 AS JSON) UNION ALL
  SELECT 'starter', 'ai.concurrent_runs.max', 'integer', CAST(1 AS JSON) UNION ALL
  SELECT 'starter', 'ai.single_run.max_credits', 'integer', CAST(20 AS JSON) UNION ALL
  SELECT 'growth', 'ai.hr.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.chat.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.resume_parse.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.match_evaluation.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.application_analysis.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.agent_run.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'growth', 'ai.credits.monthly', 'integer', CAST(1000 AS JSON) UNION ALL
  SELECT 'growth', 'ai.concurrent_runs.max', 'integer', CAST(3 AS JSON) UNION ALL
  SELECT 'growth', 'ai.single_run.max_credits', 'integer', CAST(100 AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.hr.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.chat.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.resume_parse.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.match_evaluation.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.application_analysis.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.agent_run.enabled', 'boolean', CAST('true' AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.credits.monthly', 'integer', CAST(10000 AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.concurrent_runs.max', 'integer', CAST(10 AS JSON) UNION ALL
  SELECT 'enterprise', 'ai.single_run.max_credits', 'integer', CAST(500 AS JSON)
) entitlement ON entitlement.plan_key = plan.plan_key
ON DUPLICATE KEY UPDATE `plan_version_id` = VALUES(`plan_version_id`);

INSERT INTO `billing_products` (`product_key`, `name`, `description`, `owner_type`, `product_type`, `status`) VALUES
  ('tenant_starter', '企业基础版', '包含基础 AI 权益的企业套餐', 'tenant', 'subscription', 'draft'),
  ('tenant_growth', '企业成长版', '包含成长 AI 权益的企业套餐', 'tenant', 'subscription', 'draft'),
  ('tenant_enterprise', '企业版', '包含企业级 AI 权益的企业套餐', 'tenant', 'subscription', 'draft'),
  ('candidate_free', '求职者免费版', '每月提供少量持续可用的 AI 额度', 'user', 'subscription', 'active'),
  ('candidate_pro', '求职者 Pro', '面向高频求职场景的 AI 套餐', 'user', 'subscription', 'draft'),
  ('tenant_credit_pack', '企业 AI 加量包', '购买后十二个月内有效', 'tenant', 'credit_pack', 'draft'),
  ('candidate_credit_pack', '求职者 AI 加量包', '购买后十二个月内有效', 'user', 'credit_pack', 'draft')
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `description` = VALUES(`description`);

INSERT INTO `billing_price_versions` (`product_id`, `version`, `billing_term`, `amount_fen`, `currency`, `included_credits`, `entitlement_snapshot`, `status`, `effective_at`)
SELECT `id`, 1, 'monthly', 0, 'CNY', 100, JSON_OBJECT('ai.chat.enabled', true, 'ai.credits.monthly', 100), 'published', NOW(3)
FROM `billing_products`
WHERE `product_key` = 'candidate_free'
ON DUPLICATE KEY UPDATE `product_id` = VALUES(`product_id`);

INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`, `created_at`, `updated_at`)
VALUES ('billing.manage', 'billing', 'manage', '管理当前租户的 AI 套餐、订单、支付与退款', NOW(), NOW())
ON DUPLICATE KEY UPDATE `resource` = VALUES(`resource`), `action` = VALUES(`action`), `description` = VALUES(`description`), `updated_at` = NOW();

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key = 'billing.manage'
WHERE role.role_key IN ('recruiting_admin', 'system_admin');
