DELETE role_permission FROM `role_permissions` role_permission
JOIN `permissions` permission ON permission.id = role_permission.permission_id
WHERE permission.permission_key = 'platform.billing.refund.review';
DELETE FROM `permissions` WHERE `permission_key` = 'platform.billing.refund.review';
