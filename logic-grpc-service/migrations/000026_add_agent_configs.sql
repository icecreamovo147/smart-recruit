-- Migration: 000026_add_agent_configs
-- Description: Add agent_configs and agent_tool_bindings tables for Agent configuration management

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `agent_configs` (
    `id`                    BIGINT       NOT NULL AUTO_INCREMENT,
    `name`                  VARCHAR(128) NOT NULL COMMENT 'Agent internal name (unique)',
    `display_name`          VARCHAR(256) NOT NULL COMMENT 'Agent display name for UI',
    `description`           TEXT         COMMENT 'Agent description',
    `agent_type`            VARCHAR(64)  NOT NULL COMMENT 'hr_recruiting_agent / candidate_assistant / custom',
    `model_id`              BIGINT       COMMENT 'FK to llm_models.id, NULL = use system default',
    `prompt_template_id`    BIGINT       COMMENT 'FK to prompt_templates.id, NULL = use system default',
    `instruction`           TEXT         COMMENT 'Extra instruction appended after Prompt',
    `max_iterations`        INT          NOT NULL DEFAULT 5 COMMENT 'Max tool call iterations',
    `temperature_override`  DOUBLE       COMMENT 'Overrides model default temperature, NULL = use model default',
    `is_default`            TINYINT(1)   NOT NULL DEFAULT 0 COMMENT 'Whether this is the default agent for its agent_type',
    `is_enabled`            TINYINT(1)   NOT NULL DEFAULT 1 COMMENT 'Whether this agent is enabled',
    `created_at`            DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`            DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `uk_name` (`name`),
    INDEX `idx_agent_type` (`agent_type`),
    INDEX `idx_model_id` (`model_id`),
    INDEX `idx_prompt_template_id` (`prompt_template_id`),
    INDEX `idx_agent_type_default` (`agent_type`, `is_default`),
    CONSTRAINT `fk_agent_configs_model` FOREIGN KEY (`model_id`) REFERENCES `llm_models` (`id`) ON DELETE SET NULL,
    CONSTRAINT `fk_agent_configs_prompt` FOREIGN KEY (`prompt_template_id`) REFERENCES `prompt_templates` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent configuration definitions';

CREATE TABLE IF NOT EXISTS `agent_tool_bindings` (
    `id`          BIGINT       NOT NULL AUTO_INCREMENT,
    `agent_id`    BIGINT       NOT NULL COMMENT 'FK to agent_configs.id',
    `tool_name`   VARCHAR(128) NOT NULL COMMENT 'Tool name identifier',
    `is_enabled`  TINYINT(1)   NOT NULL DEFAULT 1 COMMENT 'Whether the tool is enabled for this agent',
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE INDEX `uk_agent_tool` (`agent_id`, `tool_name`),
    INDEX `idx_tool_name` (`tool_name`),
    CONSTRAINT `fk_agent_tool_bindings_agent` FOREIGN KEY (`agent_id`) REFERENCES `agent_configs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent to tool binding assignments';

COMMIT;
