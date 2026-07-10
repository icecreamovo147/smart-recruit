-- Migration: 000029_add_ai_chat_history_model
-- Description: Persist the actual model used for assistant chat messages.

ALTER TABLE ai_chat_history
    ADD COLUMN `model_id` BIGINT NULL COMMENT 'assistant 实际使用的模型ID' AFTER `content`,
    ADD COLUMN `model_name` VARCHAR(128) NULL COMMENT 'assistant 实际使用的模型名称' AFTER `model_id`;
