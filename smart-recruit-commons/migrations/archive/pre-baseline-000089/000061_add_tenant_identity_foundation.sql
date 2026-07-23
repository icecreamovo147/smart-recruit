-- 000061_add_tenant_identity_foundation.sql
-- Establishes the shared-schema tenant control plane without removing legacy RBAC paths.

CREATE TABLE IF NOT EXISTS `tenants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_key` CHAR(36) NOT NULL COMMENT 'Immutable external tenant identifier',
  `slug` VARCHAR(64) NOT NULL COMMENT 'Immutable public tenant slug',
  `name` VARCHAR(128) NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'provisioning' COMMENT 'provisioning/active/suspended/disabled',
  `timezone` VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  `locale` VARCHAR(32) NOT NULL DEFAULT 'zh-CN',
  `is_default` TINYINT(1) NOT NULL DEFAULT 0,
  `default_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `is_default` = 1 THEN 1 ELSE NULL END) STORED,
  `created_by` BIGINT UNSIGNED NULL COMMENT 'Platform actor users.id',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenants_tenant_key` (`tenant_key`),
  UNIQUE KEY `uk_tenants_slug` (`slug`),
  UNIQUE KEY `uk_tenants_default` (`default_key`),
  KEY `idx_tenants_status` (`status`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Enterprise tenant directory';

INSERT INTO `tenants`
  (`tenant_key`, `slug`, `name`, `status`, `timezone`, `locale`, `is_default`)
VALUES
  ('00000000-0000-4000-8000-000000000001', 'default', '默认企业', 'active', 'Asia/Shanghai', 'zh-CN', 1)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `status` = 'active',
  `is_default` = 1;

CREATE TABLE IF NOT EXISTS `tenant_memberships` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'invited/active/suspended/left',
  `joined_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_membership_user` (`tenant_id`, `user_id`),
  KEY `idx_tenant_memberships_user_status` (`user_id`, `status`, `tenant_id`),
  KEY `idx_tenant_memberships_tenant_status` (`tenant_id`, `status`, `user_id`),
  CONSTRAINT `fk_tenant_memberships_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_tenant_memberships_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Global-user membership in an enterprise tenant';

ALTER TABLE `roles`
  ADD COLUMN `scope_type` VARCHAR(32) NOT NULL DEFAULT 'identity' AFTER `description`;

UPDATE `roles` SET `scope_type` = 'identity' WHERE `role_key` = 'candidate';
UPDATE `roles` SET `scope_type` = 'tenant' WHERE `role_key` IN ('recruiter', 'recruiting_admin', 'interviewer');
UPDATE `roles` SET `scope_type` = 'platform' WHERE `role_key` = 'system_admin';

INSERT INTO `roles` (`role_key`, `name`, `description`, `scope_type`, `is_system`, `created_at`, `updated_at`)
VALUES ('platform_admin', '平台管理员', '管理租户、平台安全、全局目录与跨租户运营', 'platform', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `scope_type` = 'platform',
  `is_system` = 1,
  `updated_at` = NOW();

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`)
SELECT platform_role.id, rp.permission_id, NOW()
FROM `roles` platform_role
JOIN `roles` legacy_role ON legacy_role.role_key = 'system_admin'
JOIN `role_permissions` rp ON rp.role_id = legacy_role.id
WHERE platform_role.role_key = 'platform_admin';

CREATE TABLE IF NOT EXISTS `tenant_membership_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `membership_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `assigned_by` BIGINT UNSIGNED NULL,
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` DATETIME NULL,
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_membership_role_active` (`membership_id`, `role_id`, `active_key`),
  KEY `idx_tenant_membership_roles_role` (`role_id`, `revoked_at`),
  CONSTRAINT `fk_tenant_membership_roles_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_tenant_membership_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant-scoped roles assigned to a membership';

CREATE TABLE IF NOT EXISTS `tenant_membership_data_scopes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `membership_id` BIGINT UNSIGNED NOT NULL,
  `scope_key` VARCHAR(64) NOT NULL,
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '',
  `resource_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `assigned_by` BIGINT UNSIGNED NULL,
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` DATETIME NULL,
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_membership_scope_active` (`membership_id`, `scope_key`, `resource_type`, `resource_id`, `active_key`),
  KEY `idx_tenant_membership_scopes_scope` (`scope_key`, `resource_type`, `resource_id`),
  CONSTRAINT `fk_tenant_membership_scopes_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tenant-scoped data grants assigned to a membership';

CREATE TABLE IF NOT EXISTS `platform_user_roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  `assigned_by` BIGINT UNSIGNED NULL,
  `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` DATETIME NULL,
  `active_key` TINYINT GENERATED ALWAYS AS (CASE WHEN `revoked_at` IS NULL THEN 1 ELSE NULL END) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_user_role_active` (`user_id`, `role_id`, `active_key`),
  KEY `idx_platform_user_roles_role` (`role_id`, `revoked_at`),
  CONSTRAINT `fk_platform_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_platform_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Platform-scoped role assignments';

CREATE TABLE IF NOT EXISTS `tenant_invitations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `email` VARCHAR(128) NOT NULL,
  `token_hash` CHAR(64) NOT NULL,
  `role_keys_json` JSON NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT 'pending/accepted/revoked/expired',
  `expires_at` DATETIME NOT NULL,
  `created_by` BIGINT UNSIGNED NOT NULL,
  `accepted_by` BIGINT UNSIGNED NULL,
  `accepted_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tenant_invitations_token_hash` (`token_hash`),
  KEY `idx_tenant_invitations_tenant_status` (`tenant_id`, `status`, `expires_at`),
  KEY `idx_tenant_invitations_email_status` (`email`, `status`, `expires_at`),
  CONSTRAINT `fk_tenant_invitations_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  CONSTRAINT `fk_tenant_invitations_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_tenant_invitations_acceptor` FOREIGN KEY (`accepted_by`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Single-use tenant staff invitations';

INSERT IGNORE INTO `tenant_memberships` (`tenant_id`, `user_id`, `status`, `joined_at`, `created_at`, `updated_at`)
SELECT tenant.id, users.id, 'active', NOW(), NOW(), NOW()
FROM `tenants` tenant
JOIN `users` users ON users.account_type = 'staff'
WHERE tenant.is_default = 1;

INSERT IGNORE INTO `tenant_membership_roles` (`membership_id`, `role_id`, `assigned_by`, `assigned_at`)
SELECT membership.id, user_role.role_id, user_role.assigned_by, user_role.assigned_at
FROM `tenant_memberships` membership
JOIN `user_roles` user_role ON user_role.user_id = membership.user_id AND user_role.revoked_at IS NULL
JOIN `roles` role ON role.id = user_role.role_id AND role.scope_type = 'tenant'
JOIN `tenants` tenant ON tenant.id = membership.tenant_id AND tenant.is_default = 1;

INSERT IGNORE INTO `tenant_membership_data_scopes`
  (`membership_id`, `scope_key`, `resource_type`, `resource_id`, `assigned_by`, `assigned_at`)
SELECT membership.id, scope.scope_key, scope.resource_type, scope.resource_id, scope.assigned_by, scope.assigned_at
FROM `tenant_memberships` membership
JOIN `user_data_scopes` scope ON scope.user_id = membership.user_id AND scope.revoked_at IS NULL
JOIN `tenants` tenant ON tenant.id = membership.tenant_id AND tenant.is_default = 1;

INSERT IGNORE INTO `platform_user_roles` (`user_id`, `role_id`, `assigned_by`, `assigned_at`)
SELECT user_role.user_id, platform_role.id, user_role.assigned_by, user_role.assigned_at
FROM `user_roles` user_role
JOIN `roles` legacy_role ON legacy_role.id = user_role.role_id AND legacy_role.role_key = 'system_admin'
JOIN `roles` platform_role ON platform_role.role_key = 'platform_admin'
WHERE user_role.revoked_at IS NULL;

ALTER TABLE `refresh_tokens`
  ADD COLUMN `client_app` VARCHAR(32) NOT NULL DEFAULT '' AFTER `family_id`,
  ADD COLUMN `active_tenant_id` BIGINT UNSIGNED NULL AFTER `client_app`,
  ADD COLUMN `membership_id` BIGINT UNSIGNED NULL AFTER `active_tenant_id`,
  ADD KEY `idx_refresh_tokens_active_tenant` (`active_tenant_id`, `user_id`),
  ADD KEY `idx_refresh_tokens_membership` (`membership_id`),
  ADD CONSTRAINT `fk_refresh_tokens_active_tenant` FOREIGN KEY (`active_tenant_id`) REFERENCES `tenants` (`id`),
  ADD CONSTRAINT `fk_refresh_tokens_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`);

ALTER TABLE `invite_codes`
  ADD COLUMN `tenant_id` BIGINT UNSIGNED NULL AFTER `id`;

UPDATE `invite_codes` invite
JOIN `tenants` tenant ON tenant.is_default = 1
SET invite.tenant_id = tenant.id
WHERE invite.tenant_id IS NULL;

ALTER TABLE `invite_codes`
  MODIFY COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL,
  ADD KEY `idx_invite_codes_tenant_active` (`tenant_id`, `is_active`, `expires_at`),
  ADD CONSTRAINT `fk_invite_codes_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`);

ALTER TABLE `authorization_audit_logs`
  ADD COLUMN `tenant_id` BIGINT UNSIGNED NULL AFTER `id`,
  ADD COLUMN `membership_id` BIGINT UNSIGNED NULL AFTER `tenant_id`,
  ADD KEY `idx_authorization_audit_tenant_created` (`tenant_id`, `created_at`),
  ADD CONSTRAINT `fk_authorization_audit_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`),
  ADD CONSTRAINT `fk_authorization_audit_membership` FOREIGN KEY (`membership_id`) REFERENCES `tenant_memberships` (`id`);

ALTER TABLE `users`
  MODIFY COLUMN `account_type` VARCHAR(32) NOT NULL DEFAULT 'candidate' COMMENT 'candidate | staff | platform | service';
