CREATE TABLE IF NOT EXISTS `mcp_tool_policies` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `server_id` BIGINT NOT NULL COMMENT 'FK to mcp_servers.id',
    `tool_name` VARCHAR(128) NOT NULL,
    `effect` VARCHAR(32) NOT NULL DEFAULT 'allow' COMMENT 'allow / deny',
    `risk_level` VARCHAR(32) DEFAULT 'medium',
    `require_confirmation` TINYINT(1) NOT NULL DEFAULT 0,
    `allowed_roles_json` JSON NULL,
    `allowed_scopes_json` JSON NULL,
    `required_args_json` JSON NULL,
    `denied_args_json` JSON NULL,
    `arg_rules_json` JSON NULL,
    `redact_fields_json` JSON NULL,
    `rate_limit_window_seconds` INT NOT NULL DEFAULT 0,
    `rate_limit_max_calls` INT NOT NULL DEFAULT 0,
    `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `created_by_hr_id` BIGINT NULL,
    `updated_by_hr_id` BIGINT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_mcp_tool_policy_server_tool` (`server_id`, `tool_name`),
    KEY `idx_mcp_tool_policies_enabled` (`is_enabled`),
    CONSTRAINT `fk_mcp_tool_policies_server` FOREIGN KEY (`server_id`) REFERENCES `mcp_servers`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `mcp_tool_logs`
    ADD COLUMN `policy_id` BIGINT NULL AFTER `session_id`,
    ADD COLUMN `policy_decision` VARCHAR(32) NOT NULL DEFAULT 'allow' AFTER `policy_id`,
    ADD COLUMN `policy_reason` VARCHAR(512) NULL AFTER `policy_decision`,
    ADD COLUMN `policy_snapshot_json` JSON NULL AFTER `policy_reason`,
    ADD INDEX `idx_mcp_tool_logs_server_tool_created` (`server_id`, `tool_name`, `created_at`);
