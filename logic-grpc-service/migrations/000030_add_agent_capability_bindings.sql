-- Migration: 000030_add_agent_capability_bindings
-- Description: Add unified agent capability bindings for builtin/MCP/SKILL catalog governance.

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `agent_capability_bindings` (
    `id`                 BIGINT       NOT NULL AUTO_INCREMENT,
    `agent_id`           BIGINT       NOT NULL COMMENT 'FK to agent_configs.id',
    `capability_source`  VARCHAR(32)  NOT NULL COMMENT 'builtin / mcp / skill',
    `capability_key`     VARCHAR(256) NOT NULL COMMENT 'builtin: tool_name; mcp: server_id:tool_name',
    `is_enabled`         TINYINT(1)   NOT NULL DEFAULT 1 COMMENT 'Whether the capability is enabled for this agent',
    `priority`           INT          NOT NULL DEFAULT 0 COMMENT 'Capability ordering hint',
    `policy_json`        JSON         COMMENT 'Optional capability policy payload',
    `created_at`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `uk_agent_capability` (`agent_id`, `capability_source`, `capability_key`),
    INDEX `idx_agent_capability_source` (`capability_source`),
    INDEX `idx_agent_capability_key` (`capability_key`),
    CONSTRAINT `fk_agent_capability_bindings_agent` FOREIGN KEY (`agent_id`) REFERENCES `agent_configs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent to unified capability binding assignments';

INSERT IGNORE INTO `agent_capability_bindings` (
    `agent_id`,
    `capability_source`,
    `capability_key`,
    `is_enabled`,
    `priority`,
    `created_at`,
    `updated_at`
)
SELECT
    `agent_id`,
    'builtin',
    `tool_name`,
    `is_enabled`,
    0,
    `created_at`,
    `created_at`
FROM `agent_tool_bindings`;

COMMIT;
