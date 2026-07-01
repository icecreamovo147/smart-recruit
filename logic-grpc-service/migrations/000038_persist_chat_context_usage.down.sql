ALTER TABLE `ai_chat_history`
  DROP COLUMN `context_usage_json`;

ALTER TABLE `ai_chat_sessions`
  DROP COLUMN `latest_context_usage_json`;
