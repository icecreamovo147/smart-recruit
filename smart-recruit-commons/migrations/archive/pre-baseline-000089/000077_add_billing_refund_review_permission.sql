INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`, `created_at`, `updated_at`) VALUES
  ('platform.billing.refund.review', 'platform_billing_refund', 'review', '查看并审批平台退款', NOW(), NOW())
ON DUPLICATE KEY UPDATE `resource` = VALUES(`resource`), `action` = VALUES(`action`), `description` = VALUES(`description`), `updated_at` = NOW();

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role JOIN `permissions` permission ON permission.permission_key = 'platform.billing.refund.review'
WHERE role.role_key IN ('platform_admin', 'platform_operator');
