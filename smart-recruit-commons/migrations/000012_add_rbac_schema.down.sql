-- Down: 000012_add_rbac_schema

DROP TABLE IF EXISTS `authorization_audit_logs`;
DROP TABLE IF EXISTS `user_data_scopes`;
DROP TABLE IF EXISTS `user_roles`;
DROP TABLE IF EXISTS `role_permissions`;
DROP TABLE IF EXISTS `permissions`;
DROP TABLE IF EXISTS `roles`;

ALTER TABLE `users`
  DROP COLUMN `token_version`,
  DROP COLUMN `status`,
  DROP COLUMN `account_type`;
