CREATE TABLE IF NOT EXISTS `platform_audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `actor_user_id` BIGINT UNSIGNED NOT NULL,
  `action` VARCHAR(128) NOT NULL,
  `resource_type` VARCHAR(64) NOT NULL,
  `resource_id` BIGINT UNSIGNED NULL,
  `target_tenant_id` BIGINT UNSIGNED NULL,
  `before_json` JSON NULL,
  `after_json` JSON NULL,
  `request_id` VARCHAR(128) NOT NULL DEFAULT '',
  `client_ip` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_platform_audit_actor_created` (`actor_user_id`, `created_at`),
  KEY `idx_platform_audit_tenant_created` (`target_tenant_id`, `created_at`),
  KEY `idx_platform_audit_action_created` (`action`, `created_at`),
  CONSTRAINT `fk_platform_audit_actor` FOREIGN KEY (`actor_user_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_platform_audit_target_tenant` FOREIGN KEY (`target_tenant_id`) REFERENCES `tenants` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Immutable platform control-plane audit trail';
