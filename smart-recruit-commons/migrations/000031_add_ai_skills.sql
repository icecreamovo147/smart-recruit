-- 000031_add_ai_skills.sql
-- Description: Add versioned SKILL registry and tool catalog.

CREATE TABLE IF NOT EXISTS `ai_skills` (
    `id`                 BIGINT       NOT NULL AUTO_INCREMENT,
    `name`               VARCHAR(128) NOT NULL COMMENT 'Stable skill key used in capability keys',
    `display_name`       VARCHAR(256) NOT NULL,
    `description`        TEXT,
    `source_type`        VARCHAR(32)  NOT NULL DEFAULT 'local' COMMENT 'local / git / http / mcp / builtin',
    `source_uri`         TEXT,
    `current_version_id` BIGINT       NULL COMMENT 'FK to ai_skill_versions.id, nullable until first version',
    `is_enabled`         TINYINT(1)   NOT NULL DEFAULT 1,
    `created_at`         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ai_skills_name` (`name`),
    KEY `idx_ai_skills_enabled` (`is_enabled`),
    KEY `idx_ai_skills_current_version` (`current_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Versioned SKILL registry';

CREATE TABLE IF NOT EXISTS `ai_skill_versions` (
    `id`                 BIGINT      NOT NULL AUTO_INCREMENT,
    `skill_id`           BIGINT      NOT NULL,
    `version`            VARCHAR(64) NOT NULL,
    `manifest_json`      JSON        NOT NULL,
    `instruction`        TEXT,
    `input_schema_json`  JSON        NULL,
    `output_schema_json` JSON        NULL,
    `runtime_type`       VARCHAR(32) NOT NULL DEFAULT 'prompt' COMMENT 'prompt / tool / workflow / http',
    `created_at`         TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ai_skill_versions_skill_version` (`skill_id`, `version`),
    KEY `idx_ai_skill_versions_skill` (`skill_id`),
    CONSTRAINT `fk_ai_skill_versions_skill` FOREIGN KEY (`skill_id`) REFERENCES `ai_skills` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Immutable SKILL versions';

CREATE TABLE IF NOT EXISTS `ai_skill_tools` (
    `id`                  BIGINT       NOT NULL AUTO_INCREMENT,
    `skill_version_id`    BIGINT       NOT NULL,
    `tool_name`           VARCHAR(128) NOT NULL,
    `description`         TEXT,
    `input_schema_json`   JSON         NULL,
    `runtime_config_json` JSON         NULL,
    `is_enabled`          TINYINT(1)   NOT NULL DEFAULT 1,
    `created_at`          TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`          TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ai_skill_tools_version_tool` (`skill_version_id`, `tool_name`),
    KEY `idx_ai_skill_tools_enabled` (`is_enabled`),
    CONSTRAINT `fk_ai_skill_tools_version` FOREIGN KEY (`skill_version_id`) REFERENCES `ai_skill_versions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Executable tools declared by SKILL versions';

ALTER TABLE `ai_skills`
    ADD CONSTRAINT `fk_ai_skills_current_version`
    FOREIGN KEY (`current_version_id`) REFERENCES `ai_skill_versions` (`id`) ON DELETE SET NULL;
