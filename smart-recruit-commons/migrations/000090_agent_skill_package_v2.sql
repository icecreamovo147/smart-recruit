-- Pre-launch cutover from the test-only Agent SKILL.md registry to the
-- immutable Agent Skill Package v2 schema. This migration intentionally
-- destroys all existing Agent Skill data. Capability release snapshots and
-- audit rows are retained as historical evidence.

-- A release that references the destroyed test Skill versions must not remain
-- runnable. Retire only the currently published tenant HR chat/run releases;
-- older release rows and their immutable snapshots remain untouched.
UPDATE `platform_ai_capability_versions` version
JOIN `platform_ai_capabilities` capability
  ON capability.`current_published_version_id` = version.`id`
SET version.`status` = 'retired',
    version.`retired_at` = COALESCE(version.`retired_at`, NOW(3)),
    version.`updated_at` = NOW(3)
WHERE capability.`audience` = 'tenant_hr'
  AND capability.`capability_key` IN ('ai.chat', 'ai.agent_run');

UPDATE `platform_ai_capabilities`
SET `current_published_version_id` = NULL,
    `updated_at` = NOW(3)
WHERE `audience` = 'tenant_hr'
  AND `capability_key` IN ('ai.chat', 'ai.agent_run');

-- Embeddings for the old registry, v2 packages from interrupted development,
-- and their sections must all be regenerated from canonical Package v2 data.
DELETE FROM `ai_embeddings`
WHERE `object_type` IN ('agent_skill', 'agent_skill_version', 'agent_skill_section');

-- Historical chat messages retain their display-name snapshot, but the old
-- mutable registry IDs are not valid Package v2 evidence.
UPDATE `ai_chat_history`
SET `agent_skill_ids_json` = NULL
WHERE `agent_skill_ids_json` IS NOT NULL;

ALTER TABLE `ai_chat_history`
  CHANGE COLUMN `agent_skill_ids_json` `agent_skill_version_ids_json`
    TEXT NULL COMMENT '本条用户消息选择的 Agent Skill 版本 ID 快照(JSON数组)';

-- Break the circular current-version relationship before deleting the
-- test-only registry data and replacing both tables.
UPDATE `agent_skills`
SET `current_version_id` = NULL
WHERE `current_version_id` IS NOT NULL;

ALTER TABLE `agent_skills`
  DROP FOREIGN KEY `fk_agent_skills_current_version`;

DELETE FROM `agent_skill_versions`;
DELETE FROM `agent_skills`;

DROP TABLE `agent_skill_versions`;
DROP TABLE `agent_skills`;

CREATE TABLE `agent_skills` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `display_name` VARCHAR(128) NOT NULL,
  `description` TEXT NULL,
  `current_version_id` BIGINT NULL,
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `is_manual_invocable` TINYINT(1) NOT NULL DEFAULT 1,
  `created_by` BIGINT NULL,
  `updated_by` BIGINT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skills_name` (`name`),
  KEY `idx_agent_skills_enabled` (`is_enabled`),
  KEY `idx_agent_skills_current_version` (`current_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Agent Skill Package v2 registry';

CREATE TABLE `agent_skill_versions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `skill_id` BIGINT NOT NULL,
  `version` VARCHAR(64) NOT NULL,
  `manifest_json` JSON NOT NULL,
  `core_markdown` MEDIUMTEXT NOT NULL,
  `compiled_markdown` MEDIUMTEXT NOT NULL,
  `authoring_json` JSON NULL,
  `compiled_hash` CHAR(64) NOT NULL,
  `core_estimated_tokens` INT UNSIGNED NOT NULL,
  `change_note` TEXT NULL,
  `created_by` BIGINT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skill_versions_skill_version` (`skill_id`, `version`),
  KEY `idx_agent_skill_versions_skill` (`skill_id`),
  KEY `idx_agent_skill_versions_compiled_hash` (`compiled_hash`),
  CONSTRAINT `fk_agent_skill_versions_skill`
    FOREIGN KEY (`skill_id`) REFERENCES `agent_skills` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_agent_skill_versions_core_tokens`
    CHECK (`core_estimated_tokens` BETWEEN 1 AND 800)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Immutable Agent Skill Package v2 versions';

CREATE TABLE `agent_skill_version_sections` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `skill_version_id` BIGINT NOT NULL,
  `section_key` VARCHAR(128) NOT NULL,
  `title` VARCHAR(256) NOT NULL,
  `description` TEXT NULL,
  `content_markdown` MEDIUMTEXT NOT NULL,
  `trigger_terms_json` JSON NULL,
  `semantic_tags_json` JSON NULL,
  `planner_intents_json` JSON NULL,
  `priority` INT NOT NULL DEFAULT 0,
  `ordinal` INT UNSIGNED NOT NULL,
  `estimated_tokens` INT UNSIGNED NOT NULL,
  `content_hash` CHAR(64) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skill_version_sections_key` (`skill_version_id`, `section_key`),
  KEY `idx_agent_skill_version_sections_order` (`skill_version_id`, `ordinal`, `id`),
  KEY `idx_agent_skill_version_sections_priority` (`skill_version_id`, `priority`, `id`),
  CONSTRAINT `fk_agent_skill_version_sections_version`
    FOREIGN KEY (`skill_version_id`) REFERENCES `agent_skill_versions` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_agent_skill_version_sections_key`
    CHECK (REGEXP_LIKE(`section_key`, '^[a-z][a-z0-9_-]{1,127}$', 'c')),
  CONSTRAINT `chk_agent_skill_version_sections_tokens`
    CHECK (`estimated_tokens` BETWEEN 1 AND 1200)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='On-demand reference sections for Agent Skill Package v2';

ALTER TABLE `agent_skills`
  ADD CONSTRAINT `fk_agent_skills_current_version`
  FOREIGN KEY (`current_version_id`) REFERENCES `agent_skill_versions` (`id`) ON DELETE SET NULL;
