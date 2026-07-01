ALTER TABLE `ai_chat_sessions`
  ADD COLUMN `latest_context_usage_json` TEXT NULL
  COMMENT 'Latest ContextUsageInfo snapshot JSON for this session'
  AFTER `application_id`;

ALTER TABLE `ai_chat_history`
  ADD COLUMN `context_usage_json` TEXT NULL
  COMMENT 'ContextUsageInfo snapshot JSON for this message'
  AFTER `process_content`;
