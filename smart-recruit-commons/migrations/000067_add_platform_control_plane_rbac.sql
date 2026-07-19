-- Platform control-plane roles and permissions.

INSERT INTO `roles` (`role_key`, `name`, `description`, `scope_type`, `is_system`, `created_at`, `updated_at`) VALUES
  ('platform_operator', '平台运营管理员', '管理租户运营、订阅、用量和告警，不管理平台账号或发布套餐', 'platform', 1, NOW(), NOW()),
  ('platform_auditor', '平台审计员', '只读查看平台运营、租户、用量和审计数据', 'platform', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `scope_type` = 'platform',
  `is_system` = 1,
  `updated_at` = NOW();

INSERT INTO `permissions` (`permission_key`, `resource`, `action`, `description`, `created_at`, `updated_at`) VALUES
  ('platform.dashboard.read', 'platform_dashboard', 'read', '查看平台运营总览', NOW(), NOW()),
  ('platform.tenant.read', 'platform_tenant', 'read', '查看平台租户和聚合诊断信息', NOW(), NOW()),
  ('platform.tenant.manage', 'platform_tenant', 'manage', '创建和变更平台租户生命周期', NOW(), NOW()),
  ('platform.member.manage', 'platform_member', 'manage', '管理租户成员状态和主管理员', NOW(), NOW()),
  ('platform.plan.read', 'platform_plan', 'read', '查看平台套餐和权益', NOW(), NOW()),
  ('platform.plan.manage', 'platform_plan', 'manage', '维护套餐草稿和租户覆盖', NOW(), NOW()),
  ('platform.plan.publish', 'platform_plan', 'publish', '发布或退役套餐版本', NOW(), NOW()),
  ('platform.subscription.manage', 'platform_subscription', 'manage', '管理租户套餐订阅', NOW(), NOW()),
  ('platform.usage.read', 'platform_usage', 'read', '查看跨租户用量和配额', NOW(), NOW()),
  ('platform.alert.read', 'platform_alert', 'read', '查看平台运营告警', NOW(), NOW()),
  ('platform.alert.manage', 'platform_alert', 'manage', '认领和处理平台运营告警', NOW(), NOW()),
  ('platform.audit.read', 'platform_audit', 'read', '查看平台控制面操作审计', NOW(), NOW()),
  ('platform.user.manage', 'platform_user', 'manage', '管理平台账号和平台角色', NOW(), NOW())
ON DUPLICATE KEY UPDATE
  `resource` = VALUES(`resource`),
  `action` = VALUES(`action`),
  `description` = VALUES(`description`),
  `updated_at` = NOW();

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key LIKE 'platform.%'
WHERE role.role_key = 'platform_admin';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key IN (
  'platform.dashboard.read', 'platform.tenant.read', 'platform.tenant.manage',
  'platform.member.manage', 'platform.plan.read', 'platform.plan.manage',
  'platform.subscription.manage', 'platform.usage.read', 'platform.alert.read',
  'platform.alert.manage', 'platform.audit.read'
)
WHERE role.role_key = 'platform_operator';

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT role.id, permission.id, NOW()
FROM `roles` role
JOIN `permissions` permission ON permission.permission_key IN (
  'platform.dashboard.read', 'platform.tenant.read', 'platform.plan.read',
  'platform.usage.read', 'platform.alert.read', 'platform.audit.read'
)
WHERE role.role_key = 'platform_auditor';
