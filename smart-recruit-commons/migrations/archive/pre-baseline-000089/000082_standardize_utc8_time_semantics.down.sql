SET @utc8_batch = '000082_standardize_utc8_time_semantics';

UPDATE `platform_plan_versions` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'platform_plan_versions' AND audit.column_name = 'effective_at' AND audit.row_pk = CAST(target.id AS CHAR) SET target.effective_at = audit.old_value;
UPDATE `billing_price_versions` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_price_versions' AND audit.column_name = 'effective_at' AND audit.row_pk = CAST(target.id AS CHAR) SET target.effective_at = audit.old_value;
UPDATE `ai_rate_cards` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'ai_rate_cards' AND audit.column_name = 'effective_at' AND audit.row_pk = CAST(target.id AS CHAR) SET target.effective_at = audit.old_value;
UPDATE `billing_subscriptions` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_subscriptions' AND audit.column_name = 'current_period_start' AND audit.row_pk = CAST(target.id AS CHAR) SET target.current_period_start = audit.old_value;
UPDATE `billing_subscriptions` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_subscriptions' AND audit.column_name = 'current_period_end' AND audit.row_pk = CAST(target.id AS CHAR) SET target.current_period_end = audit.old_value;
UPDATE `billing_payments` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_payments' AND audit.column_name = 'next_reconcile_at' AND audit.row_pk = CAST(target.id AS CHAR) SET target.next_reconcile_at = audit.old_value;
UPDATE `billing_payments` target JOIN `utc8_time_conversion_audit` audit ON audit.batch_id = @utc8_batch AND audit.table_name = 'billing_payments' AND audit.column_name = 'last_queried_at' AND audit.row_pk = CAST(target.id AS CHAR) SET target.last_queried_at = audit.old_value;

DELETE FROM `utc8_time_conversion_audit` WHERE `batch_id` = @utc8_batch;
DROP TABLE `utc8_time_conversion_audit`;
