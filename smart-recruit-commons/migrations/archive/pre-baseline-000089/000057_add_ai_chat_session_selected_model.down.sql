ALTER TABLE `ai_chat_sessions`
  DROP KEY `idx_ai_chat_sessions_selected_model`,
  DROP COLUMN `selected_model_id`;
