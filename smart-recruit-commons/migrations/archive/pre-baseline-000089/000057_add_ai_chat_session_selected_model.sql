ALTER TABLE `ai_chat_sessions`
  ADD COLUMN `selected_model_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '下一轮请求选择的模型ID，0表示跟随默认模型' AFTER `latest_context_usage_json`,
  ADD KEY `idx_ai_chat_sessions_selected_model` (`selected_model_id`);

UPDATE `ai_chat_sessions`
SET `selected_model_id` = COALESCE(
  CAST(JSON_UNQUOTE(JSON_EXTRACT(
    CASE WHEN JSON_VALID(`latest_context_usage_json`) THEN `latest_context_usage_json` ELSE '{}' END,
    '$.model_id'
  )) AS UNSIGNED),
  0
)
WHERE `latest_context_usage_json` IS NOT NULL;
