-- 000042_add_structured_resume_and_match_tables.sql
-- Description: Add durable structured resume profiles and candidate match evaluations.

CREATE TABLE IF NOT EXISTS `resume_parse_runs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_id` BIGINT UNSIGNED NOT NULL COMMENT 'Source resumes.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT 'Candidate users.id',
  `agent_run_id` BIGINT NULL COMMENT 'Optional agent_runs.id that produced this parse',
  `status` VARCHAR(32) NOT NULL DEFAULT 'running' COMMENT 'running / succeeded / failed',
  `parser_version` VARCHAR(64) NULL COMMENT 'Parser or prompt version used',
  `input_hash` VARCHAR(128) NULL COMMENT 'Hash of parse input for idempotency/audit',
  `error_message` TEXT NULL COMMENT 'Parse failure details',
  `started_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` DATETIME NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_parse_runs_resume` (`resume_id`),
  KEY `idx_resume_parse_runs_user` (`user_id`),
  KEY `idx_resume_parse_runs_agent_run` (`agent_run_id`),
  KEY `idx_resume_parse_runs_status` (`status`),
  CONSTRAINT `fk_resume_parse_runs_resume` FOREIGN KEY (`resume_id`) REFERENCES `resumes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_resume_parse_runs_agent_run` FOREIGN KEY (`agent_run_id`) REFERENCES `agent_runs` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Resume structured parse run audit';

CREATE TABLE IF NOT EXISTS `resume_profiles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_id` BIGINT UNSIGNED NOT NULL COMMENT 'Source resumes.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT 'Candidate users.id',
  `parse_run_id` BIGINT UNSIGNED NOT NULL COMMENT 'Producing resume_parse_runs.id',
  `version` INT NOT NULL DEFAULT 1 COMMENT 'Monotonic version per resume',
  `is_current` TINYINT NOT NULL DEFAULT 1 COMMENT 'Whether this is the current structured profile for the resume',
  `current_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_current` = 1 THEN 1 ELSE NULL END
  ) STORED,
  `full_name` VARCHAR(128) NULL,
  `email` VARCHAR(128) NULL,
  `phone` VARCHAR(64) NULL,
  `location` VARCHAR(128) NULL,
  `headline` VARCHAR(256) NULL,
  `summary` TEXT NULL,
  `total_experience_years` DECIMAL(5,2) NULL,
  `highest_degree` VARCHAR(64) NULL,
  `raw_json` JSON NULL COMMENT 'Full structured parser output',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resume_profile_version` (`resume_id`, `version`),
  UNIQUE KEY `uk_resume_profile_current` (`resume_id`, `current_key`),
  UNIQUE KEY `uk_resume_profiles_parse_run` (`parse_run_id`),
  KEY `idx_resume_profiles_resume` (`resume_id`),
  KEY `idx_resume_profiles_user` (`user_id`),
  KEY `idx_resume_profiles_current` (`is_current`),
  CONSTRAINT `fk_resume_profiles_resume` FOREIGN KEY (`resume_id`) REFERENCES `resumes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_resume_profiles_parse_run` FOREIGN KEY (`parse_run_id`) REFERENCES `resume_parse_runs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Versioned structured resume profile';

CREATE TABLE IF NOT EXISTS `resume_educations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `school` VARCHAR(128) NOT NULL,
  `degree` VARCHAR(64) NULL,
  `major` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `description` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_educations_profile` (`resume_profile_id`),
  CONSTRAINT `fk_resume_educations_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Structured resume education entries';

CREATE TABLE IF NOT EXISTS `resume_experiences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `company` VARCHAR(128) NOT NULL,
  `title` VARCHAR(128) NULL,
  `location` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `is_current` TINYINT NOT NULL DEFAULT 0,
  `description` TEXT NULL,
  `achievements_json` JSON NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_experiences_profile` (`resume_profile_id`),
  CONSTRAINT `fk_resume_experiences_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Structured resume work experience entries';

CREATE TABLE IF NOT EXISTS `resume_projects` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `role` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `description` TEXT NULL,
  `technologies_json` JSON NULL,
  `highlights_json` JSON NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_resume_projects_profile` (`resume_profile_id`),
  CONSTRAINT `fk_resume_projects_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Structured resume project entries';

CREATE TABLE IF NOT EXISTS `resume_skills` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_profile_id` BIGINT UNSIGNED NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `category` VARCHAR(64) NULL,
  `level` VARCHAR(32) NULL,
  `years` DECIMAL(5,2) NULL,
  `evidence` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resume_skill_name` (`resume_profile_id`, `name`),
  KEY `idx_resume_skills_profile` (`resume_profile_id`),
  KEY `idx_resume_skills_category` (`category`),
  CONSTRAINT `fk_resume_skills_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Structured resume skill entries';

CREATE TABLE IF NOT EXISTS `candidate_match_evaluations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT 'applications.id',
  `job_id` BIGINT UNSIGNED NOT NULL COMMENT 'jobs.id',
  `candidate_user_id` BIGINT UNSIGNED NOT NULL COMMENT 'users.id for candidate',
  `resume_profile_id` BIGINT UNSIGNED NOT NULL COMMENT 'resume_profiles.id used for matching',
  `agent_run_id` BIGINT NULL COMMENT 'Optional agent_runs.id that produced this evaluation',
  `evaluation_version` INT NOT NULL DEFAULT 1 COMMENT 'Monotonic version per application',
  `is_latest` TINYINT NOT NULL DEFAULT 1 COMMENT 'Whether this is the latest match evaluation for the application',
  `latest_key` TINYINT GENERATED ALWAYS AS (
    CASE WHEN `is_latest` = 1 THEN 1 ELSE NULL END
  ) STORED,
  `overall_score` DECIMAL(6,2) NULL,
  `recommendation` VARCHAR(32) NULL COMMENT 'strong_match / possible_match / weak_match / reject',
  `summary` TEXT NULL,
  `strengths_json` JSON NULL,
  `risks_json` JSON NULL,
  `score_breakdown_json` JSON NULL,
  `model_name` VARCHAR(128) NULL,
  `evaluated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_candidate_match_version` (`application_id`, `evaluation_version`),
  UNIQUE KEY `uk_candidate_match_latest` (`application_id`, `latest_key`),
  KEY `idx_candidate_match_application` (`application_id`),
  KEY `idx_candidate_match_job` (`job_id`),
  KEY `idx_candidate_match_candidate` (`candidate_user_id`),
  KEY `idx_candidate_match_resume_profile` (`resume_profile_id`),
  KEY `idx_candidate_match_agent_run` (`agent_run_id`),
  KEY `idx_candidate_match_latest` (`is_latest`),
  CONSTRAINT `fk_candidate_match_application` FOREIGN KEY (`application_id`) REFERENCES `applications` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_candidate_match_resume_profile` FOREIGN KEY (`resume_profile_id`) REFERENCES `resume_profiles` (`id`) ON DELETE RESTRICT,
  CONSTRAINT `fk_candidate_match_agent_run` FOREIGN KEY (`agent_run_id`) REFERENCES `agent_runs` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Versioned candidate-job match evaluation';

CREATE TABLE IF NOT EXISTS `candidate_match_evidence` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `evaluation_id` BIGINT UNSIGNED NOT NULL,
  `evidence_type` VARCHAR(64) NOT NULL,
  `dimension` VARCHAR(64) NULL,
  `source_table` VARCHAR(64) NULL,
  `source_id` BIGINT UNSIGNED NULL,
  `snippet` TEXT NULL,
  `weight` DECIMAL(6,3) NULL,
  `score_impact` DECIMAL(6,2) NULL,
  `metadata_json` JSON NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_candidate_match_evidence_eval` (`evaluation_id`),
  KEY `idx_candidate_match_evidence_type` (`evidence_type`),
  CONSTRAINT `fk_candidate_match_evidence_eval` FOREIGN KEY (`evaluation_id`) REFERENCES `candidate_match_evaluations` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Evidence supporting candidate match evaluations';
