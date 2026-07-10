-- 000028_add_mcp_servers.sql
-- P1-006: Add MCP server management and tool call logging tables

CREATE TABLE IF NOT EXISTS `mcp_servers` (
    `id`            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name`          VARCHAR(128) NOT NULL COMMENT 'MCP server display name',
    `transport`     VARCHAR(16)  NOT NULL COMMENT 'Transport type: stdio / sse / http',
    `command_or_url` TEXT         NOT NULL COMMENT 'Command (stdio) or URL (sse/http)',
    `args`          JSON         COMMENT 'Command arguments (JSON array string)',
    `env_vars`      JSON         COMMENT 'Environment variables (JSON object)',
    `timeout_seconds` INT        NOT NULL DEFAULT 30 COMMENT 'Connection timeout',
    `is_enabled`    TINYINT(1)   NOT NULL DEFAULT 0 COMMENT 'Default disabled for safety',
    `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX `idx_mcp_servers_enabled` (`is_enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='MCP server registry';

CREATE TABLE IF NOT EXISTS `mcp_tool_logs` (
    `id`            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `server_id`     BIGINT       NOT NULL COMMENT 'FK → mcp_servers.id',
    `tool_name`     VARCHAR(128) NOT NULL COMMENT 'Called tool name',
    `args_json`     JSON         COMMENT 'Desensitized tool arguments',
    `result_content` TEXT         COMMENT 'Desensitized/truncated tool result',
    `duration_ms`   INT          NOT NULL DEFAULT 0 COMMENT 'Execution time in ms',
    `error_msg`     VARCHAR(512) COMMENT 'Error message if any',
    `called_by_hr_id` BIGINT     COMMENT 'HR user who triggered the call',
    `session_id`    BIGINT       COMMENT 'AI chat session ID',
    `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_mcp_tool_logs_server` (`server_id`),
    INDEX `idx_mcp_tool_logs_session` (`session_id`),
    CONSTRAINT `fk_mcp_tool_logs_server` FOREIGN KEY (`server_id`) REFERENCES `mcp_servers`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='MCP tool call audit logs';
