-- Convert only DATETIME values whose historical writer can be proven to have
-- persisted UTC wall-clock values. Every mutation is recorded before update;
-- ambiguous rows are intentionally left untouched for the read-only preflight.

CREATE TABLE IF NOT EXISTS `utc8_time_conversion_audit` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `batch_id` VARCHAR(96) NOT NULL,
  `table_name` VARCHAR(96) NOT NULL,
  `row_pk` VARCHAR(191) NOT NULL,
  `column_name` VARCHAR(96) NOT NULL,
  `old_value` DATETIME(6) NOT NULL,
  `new_value` DATETIME(6) NOT NULL,
  `reason` VARCHAR(500) NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_utc8_time_conversion_cell` (`batch_id`, `table_name`, `row_pk`, `column_name`),
  KEY `idx_utc8_time_conversion_lookup` (`table_name`, `row_pk`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Auditable UTC wall-clock to Asia/Shanghai conversions';

SET @utc8_batch = '000082_standardize_utc8_time_semantics';

-- Platform seed publications are migration-owned when published_by is NULL.
INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'platform_plan_versions', CAST(`id` AS CHAR), 'effective_at', `effective_at`, DATE_ADD(`effective_at`, INTERVAL 8 HOUR),
       'migration-owned platform plan publication used UTC DATETIME'
FROM `platform_plan_versions`
WHERE `effective_at` IS NOT NULL AND `published_by` IS NULL AND `status` IN ('published', 'retired');

-- Migration 000081 proved immediate Billing publications by matching creation
-- and effective timestamps. Scheduled publications remain deliberately absent.
INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'billing_price_versions', CAST(`id` AS CHAR), 'effective_at', `effective_at`, DATE_ADD(`effective_at`, INTERVAL 8 HOUR),
       '000081 immediate price publication used the UTC database clock'
FROM `billing_price_versions`
WHERE `effective_at` IS NOT NULL AND `status` IN ('published', 'retired')
  AND ABS(TIMESTAMPDIFF(SECOND, `created_at`, `effective_at`)) <= 5;

INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'ai_rate_cards', CAST(`id` AS CHAR), 'effective_at', `effective_at`, DATE_ADD(`effective_at`, INTERVAL 8 HOUR),
       '000081 immediate rate-card publication used the UTC database clock'
FROM `ai_rate_cards`
WHERE `effective_at` IS NOT NULL AND `status` IN ('published', 'retired')
  AND ABS(TIMESTAMPDIFF(SECOND, `created_at`, `effective_at`)) <= 5;

-- Pre-order enterprise compatibility subscriptions have no activation order;
-- their Billing writer explicitly normalized the supplied clock with UTC().
INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'billing_subscriptions', CAST(`id` AS CHAR), 'current_period_start', `current_period_start`, DATE_ADD(`current_period_start`, INTERVAL 8 HOUR),
       'legacy tenant subscription without activation order used UTC application clock'
FROM `billing_subscriptions`
WHERE `owner_type` = 'tenant' AND `activated_by_order_id` IS NULL;

INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'billing_subscriptions', CAST(`id` AS CHAR), 'current_period_end', `current_period_end`, DATE_ADD(`current_period_end`, INTERVAL 8 HOUR),
       'legacy tenant subscription without activation order used UTC application clock'
FROM `billing_subscriptions`
WHERE `owner_type` = 'tenant' AND `activated_by_order_id` IS NULL;

-- These columns were introduced by 000076 and written only by the historical
-- reconciliation code's s.now().UTC() path.
INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'billing_payments', CAST(`id` AS CHAR), 'next_reconcile_at', `next_reconcile_at`, DATE_ADD(`next_reconcile_at`, INTERVAL 8 HOUR),
       '000076 payment reconciliation scheduler used UTC application clock'
FROM `billing_payments` WHERE `next_reconcile_at` IS NOT NULL;

INSERT IGNORE INTO `utc8_time_conversion_audit`
  (`batch_id`, `table_name`, `row_pk`, `column_name`, `old_value`, `new_value`, `reason`)
SELECT @utc8_batch, 'billing_payments', CAST(`id` AS CHAR), 'last_queried_at', `last_queried_at`, DATE_ADD(`last_queried_at`, INTERVAL 8 HOUR),
       '000076 payment reconciliation query used UTC application clock'
FROM `billing_payments` WHERE `last_queried_at` IS NOT NULL;

UPDATE `platform_plan_versions` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'platform_plan_versions' AND audit.column_name = 'effective_at' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.effective_at = audit.new_value;

UPDATE `billing_price_versions` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_price_versions' AND audit.column_name = 'effective_at' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.effective_at = audit.new_value;

UPDATE `ai_rate_cards` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'ai_rate_cards' AND audit.column_name = 'effective_at' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.effective_at = audit.new_value;

UPDATE `billing_subscriptions` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_subscriptions' AND audit.column_name = 'current_period_start' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.current_period_start = audit.new_value;

UPDATE `billing_subscriptions` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_subscriptions' AND audit.column_name = 'current_period_end' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.current_period_end = audit.new_value;

UPDATE `billing_payments` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_payments' AND audit.column_name = 'next_reconcile_at' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.next_reconcile_at = audit.new_value;

UPDATE `billing_payments` target
JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_payments' AND audit.column_name = 'last_queried_at' AND audit.row_pk = CAST(target.id AS CHAR)
SET target.last_queried_at = audit.new_value;
