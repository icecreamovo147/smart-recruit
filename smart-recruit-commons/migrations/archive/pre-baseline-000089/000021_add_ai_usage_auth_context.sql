-- ── Phase 6: AI Usage Auth Context ─────────────────────────────────────────
-- Records the RBAC context for each AI/third-party usage log so that audit
-- queries can show actor permission, scope, and resource details.

CREATE TABLE IF NOT EXISTS `ai_usage_auth_contexts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `usage_log_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 third_party_usage_logs.id',
  `actor_user_id` BIGINT UNSIGNED NOT NULL COMMENT '操作人用户ID',
  `account_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '账号类型：candidate / staff / service',
  `role_keys` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '逗号分隔的角色key列表',
  `permission_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '触发该次操作的权限key',
  `scope_keys` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '逗号分隔的数据范围key列表',
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源类型，如 ai / application / job',
  `resource_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '资源ID，0表示全局',
  `decision` VARCHAR(32) NOT NULL DEFAULT 'allowed' COMMENT '授权决策：allowed / denied',
  `request_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求追踪ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_audit_context_actor` (`actor_user_id`, `created_at`),
  KEY `idx_audit_context_permission` (`permission_key`, `created_at`),
  KEY `idx_audit_context_usage_log` (`usage_log_id`),
  KEY `idx_audit_context_request` (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI使用审计RBAC上下文表';
