-- Down migration: 000029_add_ai_chat_history_model

ALTER TABLE ai_chat_history
    DROP COLUMN `model_name`,
    DROP COLUMN `model_id`;
