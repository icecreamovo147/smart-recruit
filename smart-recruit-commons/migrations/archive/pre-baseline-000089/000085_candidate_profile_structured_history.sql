-- P1: candidate-owned structured education and experience
CREATE TABLE IF NOT EXISTS `candidate_educations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人 users.id',
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
  KEY `idx_candidate_educations_user` (`user_id`),
  CONSTRAINT `fk_candidate_educations_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人自维护教育经历';

CREATE TABLE IF NOT EXISTS `candidate_experiences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '候选人 users.id',
  `company` VARCHAR(128) NOT NULL,
  `title` VARCHAR(128) NULL,
  `location` VARCHAR(128) NULL,
  `start_date` DATE NULL,
  `end_date` DATE NULL,
  `is_current` TINYINT NOT NULL DEFAULT 0,
  `description` TEXT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_candidate_experiences_user` (`user_id`),
  CONSTRAINT `fk_candidate_experiences_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='候选人自维护工作经历';
