CREATE TABLE IF NOT EXISTS agent_skills (
  id BIGINT NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  display_name VARCHAR(128) NOT NULL,
  description TEXT,
  current_version_id BIGINT NULL,
  is_enabled TINYINT(1) NOT NULL DEFAULT 1,
  is_manual_invocable TINYINT(1) NOT NULL DEFAULT 1,
  trigger_keywords JSON NULL,
  created_by BIGINT NULL,
  updated_by BIGINT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_skills_name (name),
  KEY idx_agent_skills_enabled (is_enabled),
  KEY idx_agent_skills_current_version (current_version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS agent_skill_versions (
  id BIGINT NOT NULL AUTO_INCREMENT,
  skill_id BIGINT NOT NULL,
  version VARCHAR(64) NOT NULL,
  flow_json JSON NULL,
  skill_md MEDIUMTEXT NOT NULL,
  frontmatter_json JSON NULL,
  body_markdown MEDIUMTEXT,
  change_note TEXT,
  created_by BIGINT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_skill_versions_skill_version (skill_id, version),
  KEY idx_agent_skill_versions_skill (skill_id),
  CONSTRAINT fk_agent_skill_versions_skill FOREIGN KEY (skill_id) REFERENCES agent_skills(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
