UPDATE `ai_chat_sessions` s
LEFT JOIN (
  SELECT
    `session_id`,
    `owner_role`,
    `owner_id`,
    COUNT(*) AS `message_count`
  FROM `ai_chat_history`
  GROUP BY `session_id`, `owner_role`, `owner_id`
) c ON c.`session_id` = s.`id`
  AND c.`owner_role` = s.`owner_role`
  AND c.`owner_id` = s.`owner_id`
LEFT JOIN `ai_chat_history` h ON h.`id` = (
  SELECT h2.`id`
  FROM `ai_chat_history` h2
  WHERE h2.`session_id` = s.`id`
    AND h2.`owner_role` = s.`owner_role`
    AND h2.`owner_id` = s.`owner_id`
  ORDER BY h2.`created_at` DESC, h2.`id` DESC
  LIMIT 1
)
LEFT JOIN `ai_session_summaries` ss ON ss.`session_id` = s.`id`
SET
  s.`message_count` = COALESCE(c.`message_count`, 0),
  s.`last_message_preview` = COALESCE(LEFT(REPLACE(REPLACE(TRIM(h.`content`), '\r', ' '), '\n', ' '), 500), ''),
  s.`summary` = CASE
    WHEN s.`summary` = '' AND ss.`summary` IS NOT NULL THEN LEFT(ss.`summary`, 500)
    ELSE s.`summary`
  END
WHERE s.`deleted_at` IS NULL
  AND (
    s.`message_count` = 0
    OR s.`last_message_preview` = ''
    OR (s.`summary` = '' AND ss.`summary` IS NOT NULL)
  );
