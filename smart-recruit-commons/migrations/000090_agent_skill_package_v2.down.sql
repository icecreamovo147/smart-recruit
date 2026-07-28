-- The Package v2 cutover intentionally deleted test Skill data, embeddings,
-- and chat Skill-ID snapshots. A down migration can restore only the previous
-- empty schema shape; restoring deleted data or retired capability releases
-- requires a database backup and an explicit republish operation.

ALTER TABLE `agent_skills`
  DROP FOREIGN KEY `fk_agent_skills_current_version`;

DROP TABLE `agent_skill_version_sections`;
DROP TABLE `agent_skill_versions`;
DROP TABLE `agent_skills`;

CREATE TABLE `agent_skills` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `display_name` VARCHAR(128) NOT NULL,
  `description` TEXT,
  `current_version_id` BIGINT NULL,
  `is_enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `is_manual_invocable` TINYINT(1) NOT NULL DEFAULT 1,
  `trigger_keywords` JSON NULL,
  `agent_type` VARCHAR(64) NOT NULL DEFAULT 'hr_recruiting_agent',
  `category` VARCHAR(64) NOT NULL DEFAULT 'general',
  `scenario` VARCHAR(128) NOT NULL DEFAULT '',
  `priority` INT NOT NULL DEFAULT 0,
  `risk_level` VARCHAR(32) NOT NULL DEFAULT 'medium',
  `required_capabilities` JSON NULL,
  `output_schema` JSON NULL,
  `evaluation_criteria` JSON NULL,
  `semantic_tags` JSON NULL,
  `created_by` BIGINT NULL,
  `updated_by` BIGINT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skills_name` (`name`),
  KEY `idx_agent_skills_enabled` (`is_enabled`),
  KEY `idx_agent_skills_governance` (`agent_type`, `category`, `priority`),
  KEY `idx_agent_skills_current_version` (`current_version_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Governed Agent SKILL.md registry';

CREATE TABLE `agent_skill_versions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `skill_id` BIGINT NOT NULL,
  `version` VARCHAR(64) NOT NULL,
  `flow_json` JSON NULL,
  `skill_md` MEDIUMTEXT NOT NULL,
  `frontmatter_json` JSON NULL,
  `body_markdown` MEDIUMTEXT,
  `change_note` TEXT,
  `created_by` BIGINT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_agent_skill_versions_skill_version` (`skill_id`, `version`),
  KEY `idx_agent_skill_versions_skill` (`skill_id`),
  CONSTRAINT `fk_agent_skill_versions_skill`
    FOREIGN KEY (`skill_id`) REFERENCES `agent_skills` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Immutable Agent SKILL.md versions';

ALTER TABLE `agent_skills`
  ADD CONSTRAINT `fk_agent_skills_current_version`
  FOREIGN KEY (`current_version_id`) REFERENCES `agent_skill_versions` (`id`) ON DELETE SET NULL;

ALTER TABLE `ai_chat_history`
  CHANGE COLUMN `agent_skill_version_ids_json` `agent_skill_ids_json`
    TEXT NULL COMMENT '本条用户消息选择的 Agent Skill ID 快照(JSON数组)';
