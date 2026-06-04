-- 000018: Add interview schedule fields and interview_feedback table (Phase 2)

ALTER TABLE `interview_schedules`
  ADD COLUMN `title` VARCHAR(128) DEFAULT NULL COMMENT '面试标题，如 初试/复试/终面' AFTER `round_no`,
  ADD COLUMN `mode` VARCHAR(32) DEFAULT NULL COMMENT '面试模式：video / phone / onsite' AFTER `title`,
  ADD COLUMN `meeting_url` VARCHAR(512) DEFAULT NULL COMMENT '视频会议链接' AFTER `mode`,
  ADD COLUMN `location` VARCHAR(256) DEFAULT NULL COMMENT '面试地点（线下）' AFTER `meeting_url`,
  ADD COLUMN `duration_minutes` INT DEFAULT NULL COMMENT '面试时长（分钟）' AFTER `location`,
  ADD COLUMN `candidate_note` VARCHAR(1024) DEFAULT NULL COMMENT '给候选人的注意事项' AFTER `duration_minutes`,
  ADD COLUMN `internal_note` VARCHAR(1024) DEFAULT NULL COMMENT '内部备注（候选人不可见）' AFTER `candidate_note`,
  ADD COLUMN `cancel_reason` VARCHAR(512) DEFAULT NULL COMMENT '取消原因' AFTER `internal_note`;

CREATE TABLE IF NOT EXISTS `interview_feedback` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `interview_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 interview_schedules.id',
  `application_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 applications.id',
  `interviewer_id` BIGINT UNSIGNED NOT NULL COMMENT '面试官用户ID',
  `recommendation` VARCHAR(32) DEFAULT NULL COMMENT '推荐结论：positive / negative / pending',
  `score` INT DEFAULT NULL COMMENT '评分（0-10）',
  `dimension_scores_json` TEXT DEFAULT NULL COMMENT '维度评分 JSON，如 {"communication":4,"technical":5}',
  `comments` TEXT DEFAULT NULL COMMENT '面试评语',
  `submitted_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '提交时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_interview_feedback_once` (`interview_id`, `application_id`, `interviewer_id`),
  KEY `idx_feedback_interview` (`interview_id`),
  KEY `idx_feedback_interviewer` (`interviewer_id`),
  KEY `idx_feedback_application` (`application_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='面试反馈表';
