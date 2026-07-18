ALTER TABLE `ai_chat_sessions`
  ADD COLUMN `session_type` VARCHAR(32) NOT NULL DEFAULT 'general' COMMENT '会话类型：general/resume/job_match/interview/offer/progress' AFTER `application_id`,
  ADD COLUMN `source_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '来源类型：job/application/resume/interview/offer等' AFTER `session_type`,
  ADD COLUMN `source_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '来源业务ID' AFTER `source_type`,
  ADD COLUMN `source_title` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '来源标题' AFTER `source_id`,
  ADD COLUMN `summary` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '会话摘要' AFTER `source_title`,
  ADD COLUMN `last_message_preview` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '最近消息摘要' AFTER `summary`,
  ADD COLUMN `message_count` INT NOT NULL DEFAULT 0 COMMENT '会话消息数' AFTER `last_message_preview`,
  ADD KEY `idx_owner_type_updated` (`owner_role`, `owner_id`, `session_type`, `updated_at`),
  ADD KEY `idx_owner_source` (`owner_role`, `owner_id`, `source_type`, `source_id`);
