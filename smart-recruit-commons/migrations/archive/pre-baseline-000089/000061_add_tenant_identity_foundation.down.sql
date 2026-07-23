ALTER TABLE `authorization_audit_logs`
  DROP FOREIGN KEY `fk_authorization_audit_membership`,
  DROP FOREIGN KEY `fk_authorization_audit_tenant`,
  DROP INDEX `idx_authorization_audit_tenant_created`,
  DROP COLUMN `membership_id`,
  DROP COLUMN `tenant_id`;

ALTER TABLE `invite_codes`
  DROP FOREIGN KEY `fk_invite_codes_tenant`,
  DROP INDEX `idx_invite_codes_tenant_active`,
  DROP COLUMN `tenant_id`;

ALTER TABLE `refresh_tokens`
  DROP FOREIGN KEY `fk_refresh_tokens_membership`,
  DROP FOREIGN KEY `fk_refresh_tokens_active_tenant`,
  DROP INDEX `idx_refresh_tokens_membership`,
  DROP INDEX `idx_refresh_tokens_active_tenant`,
  DROP COLUMN `membership_id`,
  DROP COLUMN `active_tenant_id`,
  DROP COLUMN `client_app`;

DROP TABLE IF EXISTS `tenant_invitations`;
DROP TABLE IF EXISTS `platform_user_roles`;
DROP TABLE IF EXISTS `tenant_membership_data_scopes`;
DROP TABLE IF EXISTS `tenant_membership_roles`;
DROP TABLE IF EXISTS `tenant_memberships`;

DELETE FROM `role_permissions`
WHERE `role_id` IN (SELECT `id` FROM `roles` WHERE `role_key` = 'platform_admin');
DELETE FROM `roles` WHERE `role_key` = 'platform_admin';

ALTER TABLE `roles` DROP COLUMN `scope_type`;
DROP TABLE IF EXISTS `tenants`;

ALTER TABLE `users`
  MODIFY COLUMN `account_type` VARCHAR(32) NOT NULL DEFAULT 'candidate' COMMENT 'candidate | staff | service';
