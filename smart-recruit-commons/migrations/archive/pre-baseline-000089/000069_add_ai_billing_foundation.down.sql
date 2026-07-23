DROP TABLE IF EXISTS `ai_credit_ledger`;
ALTER TABLE `ai_usage_events` DROP FOREIGN KEY `fk_ai_usage_events_reservation`;
DROP TABLE IF EXISTS `ai_credit_reservations`;
DROP TABLE IF EXISTS `ai_credit_grants`;
DROP TABLE IF EXISTS `ai_usage_events`;
DROP TABLE IF EXISTS `ai_rate_cards`;
DROP TABLE IF EXISTS `billing_webhook_events`;
DROP TABLE IF EXISTS `billing_refunds`;
DROP TABLE IF EXISTS `billing_payments`;
ALTER TABLE `billing_subscriptions` DROP FOREIGN KEY `fk_billing_subscriptions_activation_order`;
DROP TABLE IF EXISTS `billing_orders`;
DROP TABLE IF EXISTS `billing_subscriptions`;
DROP TABLE IF EXISTS `billing_price_versions`;
DROP TABLE IF EXISTS `billing_products`;
DELETE mapping FROM `role_permissions` mapping JOIN `permissions` permission ON permission.id = mapping.permission_id WHERE permission.permission_key = 'billing.manage';
DELETE FROM `permissions` WHERE `permission_key` = 'billing.manage';
DELETE entitlement
FROM `platform_plan_entitlements` entitlement
JOIN `platform_plan_versions` version ON version.id = entitlement.plan_version_id
WHERE version.version = 2 AND version.change_note = 'AI billing shadow defaults; review before enforcement';
DELETE FROM `platform_plan_versions`
WHERE version = 2 AND change_note = 'AI billing shadow defaults; review before enforcement';
