-- Down migration: 000035_add_ai_chat_history_process_content

ALTER TABLE ai_chat_history
    DROP COLUMN `process_content`;
