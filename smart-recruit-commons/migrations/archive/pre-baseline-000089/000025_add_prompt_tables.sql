-- Migration: 000025_add_prompt_tables
-- Description: Add prompt_templates and prompt_versions tables for Prompt template management

START TRANSACTION;

CREATE TABLE IF NOT EXISTS `prompt_templates` (
    `id`              BIGINT       NOT NULL AUTO_INCREMENT,
    `name`            VARCHAR(256) NOT NULL COMMENT 'Template name',
    `content`         TEXT         NOT NULL COMMENT 'Prompt template content with {{variable}} placeholders',
    `variables`       JSON         COMMENT 'JSON array of variable names, e.g. ["hr_id","session_id"]',
    `version`         INT          NOT NULL DEFAULT 1 COMMENT 'Current version number',
    `is_active`       TINYINT(1)   NOT NULL DEFAULT 1 COMMENT 'Whether template is active',
    `agent_type`      VARCHAR(64)  NOT NULL COMMENT 'hr_agent / candidate_assistant',
    `prompt_role`     VARCHAR(32)  NOT NULL DEFAULT 'system' COMMENT 'system / user',
    `created_by`      BIGINT       COMMENT 'Creator user ID',
    `updated_by`      BIGINT       COMMENT 'Last updater user ID',
    `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_agent_type` (`agent_type`),
    INDEX `idx_is_active` (`is_active`),
    INDEX `idx_agent_type_active` (`agent_type`, `is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Prompt template definitions';

CREATE TABLE IF NOT EXISTS `prompt_versions` (
    `id`              BIGINT       NOT NULL AUTO_INCREMENT,
    `template_id`     BIGINT       NOT NULL COMMENT 'FK to prompt_templates.id',
    `version`         INT          NOT NULL COMMENT 'Version number',
    `content`         TEXT         NOT NULL COMMENT 'Snapshot of prompt content at this version',
    `changed_by`      BIGINT       COMMENT 'User ID who made the change',
    `change_note`     VARCHAR(512) COMMENT 'Description of what changed',
    `created_at`      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_template_version` (`template_id`, `version`),
    CONSTRAINT `fk_prompt_versions_template` FOREIGN KEY (`template_id`) REFERENCES `prompt_templates` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Prompt version history for audit and rollback';

COMMIT;
